package contribgraph

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"os"
	"path/filepath"
	"testing"
)

// TestRenderGIFDecodes confirms RenderGIF emits a syntactically valid
// looping GIF with the expected dimensions, frame count and per-frame
// delay. We round-trip through image/gif so any encoder bug surfaces here
// rather than at runtime on a user's machine.
func TestRenderGIFDecodes(t *testing.T) {
	raw, err := RenderGIF(sampleData(), "octocat", nil)
	if err != nil {
		t.Fatalf("RenderGIF: %v", err)
	}
	if len(raw) == 0 {
		t.Fatalf("empty GIF bytes")
	}
	if !bytes.HasPrefix(raw, []byte("GIF87a")) && !bytes.HasPrefix(raw, []byte("GIF89a")) {
		t.Errorf("bytes don't start with a GIF magic: %q", string(raw[:6]))
	}
	g, err := gif.DecodeAll(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("DecodeAll: %v", err)
	}
	if g.LoopCount != 0 {
		t.Errorf("LoopCount = %d, want 0 (infinite)", g.LoopCount)
	}
	if got, want := len(g.Image), gifFrames; got != want {
		t.Errorf("frame count = %d, want %d", got, want)
	}
	// sampleData() has 2 weeks; width = 2*13-3 = 23, height = 7*13-3 = 88,
	// both scaled by gifScale (= 2).
	wantW := (2*13 - 3) * gifScale
	wantH := (7*13 - 3) * gifScale
	if g.Config.Width != wantW {
		t.Errorf("width = %d, want %d", g.Config.Width, wantW)
	}
	if g.Config.Height != wantH {
		t.Errorf("height = %d, want %d", g.Config.Height, wantH)
	}
	wantDelay := gifFrameMs / 10
	for i, d := range g.Delay {
		if d != wantDelay {
			t.Errorf("frame %d delay = %d, want %d", i, d, wantDelay)
			break
		}
	}
}

// TestRenderGIFFramesSharePalette verifies that every decoded frame uses
// the same palette in the same order. If they don't, static regions (the
// empty grey grid, the dim lulls between sparkles) shimmer every frame
// from re-quantisation drift — exactly the bug we built the shared palette
// to avoid.
func TestRenderGIFFramesSharePalette(t *testing.T) {
	raw, err := RenderGIF(sampleData(), "octocat", nil)
	if err != nil {
		t.Fatalf("RenderGIF: %v", err)
	}
	g, err := gif.DecodeAll(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("DecodeAll: %v", err)
	}
	if len(g.Image) < 2 {
		t.Fatalf("not enough frames to compare: %d", len(g.Image))
	}
	first := g.Image[0].Palette
	for i := 1; i < len(g.Image); i++ {
		p := g.Image[i].Palette
		if len(p) != len(first) {
			t.Fatalf("frame %d palette size = %d, want %d", i, len(p), len(first))
		}
		for j := range p {
			r1, g1, b1, a1 := first[j].RGBA()
			r2, g2, b2, a2 := p[j].RGBA()
			if r1 != r2 || g1 != g2 || b1 != b2 || a1 != a2 {
				t.Fatalf("frame %d palette[%d] differs from frame 0", i, j)
			}
		}
	}
}

// TestRenderGIFHasTransparentBackground guards the contract that the
// rendered GIF leaves its background transparent so it can layer cleanly
// over README backgrounds in light or dark mode.
func TestRenderGIFHasTransparentBackground(t *testing.T) {
	raw, err := RenderGIF(sampleData(), "octocat", nil)
	if err != nil {
		t.Fatalf("RenderGIF: %v", err)
	}
	g, err := gif.DecodeAll(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("DecodeAll: %v", err)
	}
	pal := g.Image[0].Palette
	transparentSlots := 0
	for _, c := range pal {
		_, _, _, a := c.RGBA()
		if a == 0 {
			transparentSlots++
		}
	}
	if transparentSlots == 0 {
		t.Errorf("palette has no transparent slot: %v", pal)
	}
}

// TestGenerateFromDataWritesBoth checks the end-to-end disk path: a
// single GenerateFromData call leaves both an SVG and a GIF on disk.
func TestGenerateFromDataWritesBoth(t *testing.T) {
	dir := t.TempDir()
	svgPath := filepath.Join(dir, "contribution-graph.svg")
	svg, err := GenerateFromData(sampleData(), "octocat", svgPath)
	if err != nil {
		t.Fatalf("GenerateFromData: %v", err)
	}
	if len(svg) == 0 {
		t.Fatalf("returned svg is empty")
	}
	if _, err := os.Stat(svgPath); err != nil {
		t.Fatalf("svg not written: %v", err)
	}
	gifPath := filepath.Join(dir, "contribution-graph.gif")
	gifBytes, err := os.ReadFile(gifPath)
	if err != nil {
		t.Fatalf("read gif: %v", err)
	}
	if _, err := gif.DecodeAll(bytes.NewReader(gifBytes)); err != nil {
		t.Errorf("gif on disk does not decode: %v", err)
	}
}

func TestGifPathFor(t *testing.T) {
	cases := map[string]string{
		"foo/bar.svg":             "foo/bar.gif",
		"foo/bar.SVG":             "foo/bar.gif",
		"/abs/path/something.svg": "/abs/path/something.gif",
		"noext":                   "noext.gif",
		"":                        ".gif",
	}
	for in, want := range cases {
		if got := gifPathFor(in); got != want {
			t.Errorf("gifPathFor(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestDrawRoundedRectMasksCorners exercises the corner-clipping logic
// directly: the four corner pixels must be transparent (index 0), while
// the four edge midpoints and the centre must be set to the requested
// index.
func TestDrawRoundedRectMasksCorners(t *testing.T) {
	const w, h, radius = 20, 20, 4
	pal := color.Palette{
		color.RGBA{0, 0, 0, 0},     // 0: transparent
		color.RGBA{255, 0, 0, 255}, // 1: red
	}
	img := image.NewPaletted(image.Rect(0, 0, w, h), pal)
	drawRoundedRect(img, 0, 0, w, h, radius, 1)

	corners := [][2]int{{0, 0}, {w - 1, 0}, {0, h - 1}, {w - 1, h - 1}}
	for _, p := range corners {
		if got := img.ColorIndexAt(p[0], p[1]); got != 0 {
			t.Errorf("corner (%d,%d) = %d, want 0 (transparent)", p[0], p[1], got)
		}
	}
	inside := [][2]int{
		{w / 2, h / 2}, // centre
		{w / 2, 0},     // top edge midpoint
		{w / 2, h - 1}, // bottom edge midpoint
		{0, h / 2},     // left edge midpoint
		{w - 1, h / 2}, // right edge midpoint
	}
	for _, p := range inside {
		if got := img.ColorIndexAt(p[0], p[1]); got != 1 {
			t.Errorf("interior/edge (%d,%d) = %d, want 1", p[0], p[1], got)
		}
	}
}

// TestMedianCutShrinksToMaxN sanity-checks the quantiser: 256 distinct
// inputs reduce to ≤ 16 outputs, and every output stays inside the
// original RGB range so we don't manufacture out-of-gamut colours.
func TestMedianCutShrinksToMaxN(t *testing.T) {
	var colors []color.RGBA
	var weights []int
	for i := 0; i < 16; i++ {
		for j := 0; j < 16; j++ {
			colors = append(colors, color.RGBA{R: uint8(i * 16), G: uint8(j * 16), B: 0, A: 255})
			weights = append(weights, 1)
		}
	}
	out := medianCut(colors, weights, 16)
	if len(out) > 16 {
		t.Fatalf("got %d colours, want ≤ 16", len(out))
	}
	for _, c := range out {
		if c.A != 0xFF {
			t.Errorf("output colour %v not fully opaque", c)
		}
	}
}

// TestRenderGIFDeterministic confirms the seeded RNG actually produces
// the same bytes on consecutive calls. Without this guarantee, the GIF
// would churn on every regeneration and pollute git diffs.
func TestRenderGIFDeterministic(t *testing.T) {
	a, err := RenderGIF(sampleData(), "octocat", nil)
	if err != nil {
		t.Fatalf("first RenderGIF: %v", err)
	}
	b, err := RenderGIF(sampleData(), "octocat", nil)
	if err != nil {
		t.Fatalf("second RenderGIF: %v", err)
	}
	if !bytes.Equal(a, b) {
		t.Errorf("RenderGIF is non-deterministic: %d vs %d bytes", len(a), len(b))
	}
}
