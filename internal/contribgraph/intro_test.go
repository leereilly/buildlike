package contribgraph

import (
	"bytes"
	"image/gif"
	"testing"
)

// TestRenderIntroGIFShape pins the encoded GIF to the expected dimensions
// and total frame count so future tweaks to the phase budget surface as
// a loud test failure instead of a subtly different on-disk animation.
func TestRenderIntroGIFShape(t *testing.T) {
	raw, err := RenderIntroGIF(sampleData(), "octocat", nil, ThemeLight, 3)
	if err != nil {
		t.Fatalf("RenderIntroGIF: %v", err)
	}
	g, err := gif.DecodeAll(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("DecodeAll: %v", err)
	}
	if got, want := len(g.Image), introTotalFrames; got != want {
		t.Errorf("frame count = %d, want %d", got, want)
	}
	// Even when the data is tiny (sampleData has 2 weeks), the intro
	// renderer pads the canvas out to the 53-week wordmark width.
	wantWeeks := 53
	wantW := (wantWeeks*13 - 3) * 3
	wantH := (7*13 - 3) * 3
	if g.Config.Width != wantW || g.Config.Height != wantH {
		t.Errorf("dimensions = %dx%d, want %dx%d", g.Config.Width, g.Config.Height, wantW, wantH)
	}
	if g.LoopCount != 0 {
		t.Errorf("LoopCount = %d, want 0 (infinite loop)", g.LoopCount)
	}
}

// TestRenderIntroGIFPhasesDiffer verifies the four-phase animation
// produces distinguishable frames at each phase boundary: frame 0
// (wordmark hold) and a mid-graph drift frame (Phase C) must encode
// non-trivially different content, otherwise the morph collapsed to a
// static loop.
func TestRenderIntroGIFPhasesDiffer(t *testing.T) {
	raw, err := RenderIntroGIF(WordmarkBuild2026Data(), "octocat", nil, ThemeDark, 0)
	if err != nil {
		t.Fatalf("RenderIntroGIF: %v", err)
	}
	g, err := gif.DecodeAll(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("DecodeAll: %v", err)
	}
	// Phase boundaries (numeric: 12, 24, 48). Pick a frame from each
	// phase and assert that at least two of the four are distinct.
	probes := []int{
		0,                                      // Phase A — wordmark hold
		introHoldFrames + introMorphFrames/2,   // Phase B — mid-morph
		introHoldFrames + introMorphFrames + 6, // Phase C — mid-drift
		introHoldFrames + introMorphFrames + introGraphFrames + introOutroFrames/2, // Phase D — mid-outro
	}
	// Each frame is delta-encoded; a stable phase emits a 1x1 stub. So
	// the *size* of each frame's image varies across phases.
	sizes := map[int]int{}
	for _, p := range probes {
		if p >= len(g.Image) {
			t.Fatalf("probe frame %d out of range (have %d)", p, len(g.Image))
		}
		b := g.Image[p].Bounds()
		sizes[b.Dx()*b.Dy()]++
	}
	if len(sizes) < 2 {
		t.Errorf("phases produced the same frame size each (%v) — morph likely collapsed", sizes)
	}
}

// TestRenderIntroGIFDeterministic confirms the intro renderer is byte-
// stable across runs given the same inputs, so the committed GIF assets
// stay reproducible.
func TestRenderIntroGIFDeterministic(t *testing.T) {
	a, err := RenderIntroGIF(WordmarkBuild2026Data(), "octocat", nil, ThemeLight, 3)
	if err != nil {
		t.Fatalf("first RenderIntroGIF: %v", err)
	}
	b, err := RenderIntroGIF(WordmarkBuild2026Data(), "octocat", nil, ThemeLight, 3)
	if err != nil {
		t.Fatalf("second RenderIntroGIF: %v", err)
	}
	if !bytes.Equal(a, b) {
		t.Errorf("RenderIntroGIF is non-deterministic: %d vs %d bytes", len(a), len(b))
	}
}

// TestRenderIntroGIFTransparent confirms ThemeNone leaves the empty
// grid transparent: palette[0] still has alpha 0 and the renderer
// doesn't crash when there are no dead-cell colours to seed the
// quantiser with.
func TestRenderIntroGIFTransparent(t *testing.T) {
	raw, err := RenderIntroGIF(sampleData(), "octocat", nil, ThemeNone, 0)
	if err != nil {
		t.Fatalf("RenderIntroGIF: %v", err)
	}
	g, err := gif.DecodeAll(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("DecodeAll: %v", err)
	}
	_, _, _, a := g.Image[0].Palette[0].RGBA()
	if a != 0 {
		t.Errorf("palette[0] alpha = %d, want 0 (transparent)", a)
	}
}
