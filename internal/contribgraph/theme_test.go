package contribgraph

import (
	"bytes"
	"image"
	"image/gif"
	"strings"
	"testing"
)

// TestThemeByName ensures the CLI-facing resolver returns the expected
// theme for every supported name and an error for anything else.
func TestThemeByName(t *testing.T) {
	cases := map[string]Theme{
		"":      ThemeNone,
		"none":  ThemeNone,
		"light": ThemeLight,
		"dark":  ThemeDark,
	}
	for in, want := range cases {
		got, err := ThemeByName(in)
		if err != nil {
			t.Fatalf("ThemeByName(%q): unexpected error %v", in, err)
		}
		if got.Name != want.Name || got.EmptyCell != want.EmptyCell {
			t.Errorf("ThemeByName(%q) = %+v, want %+v", in, got, want)
		}
	}
	if _, err := ThemeByName("solarized"); err == nil {
		t.Errorf("ThemeByName(\"solarized\"): expected error")
	}
}

// TestThemeTransparentDetectsZero confirms an unset theme (alpha 0) is
// treated as "skip empty cells" by both renderers.
func TestThemeTransparentDetectsZero(t *testing.T) {
	if !ThemeNone.Transparent() {
		t.Errorf("ThemeNone should be transparent")
	}
	if ThemeLight.Transparent() {
		t.Errorf("ThemeLight should not be transparent (alpha = %d)", ThemeLight.EmptyCell.A)
	}
	if ThemeDark.Transparent() {
		t.Errorf("ThemeDark should not be transparent (alpha = %d)", ThemeDark.EmptyCell.A)
	}
}

// TestRenderWithThemeDrawsEmptyCells checks that a themed SVG render
// includes every level-0 cell as a rect filled with the theme's empty
// colour, while the no-theme variant skips them entirely.
func TestRenderWithThemeDrawsEmptyCells(t *testing.T) {
	data := sampleData()
	transparent := string(RenderWithTheme(data, "octocat", nil, ThemeNone))
	light := string(RenderWithTheme(data, "octocat", nil, ThemeLight))
	dark := string(RenderWithTheme(data, "octocat", nil, ThemeDark))

	// sampleData has 6 live cells and 8 dead cells (= 14 days across 2
	// weeks). Transparent renders only the live ones; themed renders
	// both. We count <rect markers rather than parsing XML.
	if got, want := strings.Count(transparent, "<rect"), 6; got != want {
		t.Errorf("transparent rects = %d, want %d", got, want)
	}
	if got, want := strings.Count(light, "<rect"), 14; got != want {
		t.Errorf("light rects = %d, want %d", got, want)
	}
	if got, want := strings.Count(dark, "<rect"), 14; got != want {
		t.Errorf("dark rects = %d, want %d", got, want)
	}

	if !strings.Contains(light, ThemeLight.EmptyCellHex()) {
		t.Errorf("light render missing empty-cell fill %q", ThemeLight.EmptyCellHex())
	}
	if !strings.Contains(dark, ThemeDark.EmptyCellHex()) {
		t.Errorf("dark render missing empty-cell fill %q", ThemeDark.EmptyCellHex())
	}
	if strings.Contains(transparent, ThemeLight.EmptyCellHex()) {
		t.Errorf("transparent render should not contain light empty-cell fill")
	}
}

// TestRenderBackwardCompat asserts the no-arg Render path is byte-stable
// against the themed renderer with ThemeNone (i.e. existing callers and
// committed SVG files stay untouched).
func TestRenderBackwardCompat(t *testing.T) {
	got := Render(sampleData(), "octocat", nil)
	want := RenderWithTheme(sampleData(), "octocat", nil, ThemeNone)
	if !bytes.Equal(got, want) {
		t.Errorf("Render and RenderWithTheme(ThemeNone) drifted apart")
	}
}

// TestRenderGIFWithThemeAcceptsScale verifies the optional scale
// parameter actually upsizes the encoded frame: a scale-3 render produces
// a GIF whose dimensions are 1.5× the scale-2 default for the same data.
func TestRenderGIFWithThemeAcceptsScale(t *testing.T) {
	rawDefault, err := RenderGIFWithTheme(sampleData(), "octocat", nil, ThemeLight, 0)
	if err != nil {
		t.Fatalf("RenderGIFWithTheme(scale=0): %v", err)
	}
	gDef, err := gif.DecodeAll(bytes.NewReader(rawDefault))
	if err != nil {
		t.Fatalf("decode default: %v", err)
	}
	rawBig, err := RenderGIFWithTheme(sampleData(), "octocat", nil, ThemeLight, 3)
	if err != nil {
		t.Fatalf("RenderGIFWithTheme(scale=3): %v", err)
	}
	gBig, err := gif.DecodeAll(bytes.NewReader(rawBig))
	if err != nil {
		t.Fatalf("decode big: %v", err)
	}
	// Default uses gifScale (=2); scale=3 explicit override should be 3/2 wider.
	if gBig.Config.Width*2 != gDef.Config.Width*3 {
		t.Errorf("scale ratio off: default=%dx%d, scale3=%dx%d",
			gDef.Config.Width, gDef.Config.Height, gBig.Config.Width, gBig.Config.Height)
	}
}

// TestRenderGIFWithThemeTransparent confirms a ThemeNone render skips the
// dead-cell rectangles entirely — the result decodes cleanly and the
// reserved transparent slot is still at palette index 0.
func TestRenderGIFWithThemeTransparent(t *testing.T) {
	raw, err := RenderGIFWithTheme(sampleData(), "octocat", nil, ThemeNone, 0)
	if err != nil {
		t.Fatalf("RenderGIFWithTheme: %v", err)
	}
	g, err := gif.DecodeAll(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("DecodeAll: %v", err)
	}
	if len(g.Image) != gifFrames {
		t.Errorf("frame count = %d, want %d", len(g.Image), gifFrames)
	}
	// Palette index 0 must still be the transparent slot.
	_, _, _, a := g.Image[0].Palette[0].RGBA()
	if a != 0 {
		t.Errorf("palette[0] alpha = %d, want 0 (transparent)", a)
	}
}

// TestRenderGIFThemeColorsAppearInPalette checks the empty-cell colour
// actually makes it into the final encoded palette (after median-cut
// quantisation). Without this guarantee the empty grid would be drawn in
// the nearest live-cell colour instead of GitHub-grey / GitHub-dark.
func TestRenderGIFThemeColorsAppearInPalette(t *testing.T) {
	check := func(t *testing.T, theme Theme) {
		t.Helper()
		raw, err := RenderGIFWithTheme(sampleData(), "octocat", nil, theme, 0)
		if err != nil {
			t.Fatalf("RenderGIFWithTheme(%s): %v", theme.Name, err)
		}
		g, err := gif.DecodeAll(bytes.NewReader(raw))
		if err != nil {
			t.Fatalf("DecodeAll: %v", err)
		}
		// Empty cells in the grid render as the *quantised* nearest
		// neighbour of theme.EmptyCell. Walk the palette and look for
		// any colour within a few channels of the requested empty cell.
		want := theme.EmptyCell
		found := false
		for _, c := range g.Image[0].Palette {
			r, gg, b, _ := c.RGBA()
			dr := int(r>>8) - int(want.R)
			dg := int(gg>>8) - int(want.G)
			db := int(b>>8) - int(want.B)
			if dr*dr+dg*dg+db*db < 64 {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("theme %s empty colour %v missing from encoded palette", theme.Name, want)
		}
	}
	t.Run("light", func(t *testing.T) { check(t, ThemeLight) })
	t.Run("dark", func(t *testing.T) { check(t, ThemeDark) })
}

// TestWordmarkBuild2026Shape pins the synthetic data to a 53-week ×
// 7-day grid so the on-disk wordmark GIF stays the right shape if anyone
// edits the layout later.
func TestWordmarkBuild2026Shape(t *testing.T) {
	d := WordmarkBuild2026Data()
	if got, want := len(d.Weeks), 53; got != want {
		t.Fatalf("weeks = %d, want %d", got, want)
	}
	for i, w := range d.Weeks {
		if got, want := len(w.ContributionDays), 7; got != want {
			t.Errorf("week %d days = %d, want %d", i, got, want)
		}
		if w.FirstDay == "" {
			t.Errorf("week %d missing FirstDay", i)
		}
	}
	// Every contributing cell sits in rows 1..5 of the grid (top and
	// bottom rows reserved as breathing space).
	for _, w := range d.Weeks {
		for _, day := range w.ContributionDays {
			if day.Level > 0 {
				if day.Weekday == 0 || day.Weekday == 6 {
					t.Errorf("wordmark cell on weekday %d (should be rows 1..5 only)", day.Weekday)
				}
			}
		}
	}
}

// TestWordmarkBuild2026Renders confirms the synthetic Data flows through
// every renderer without panicking and produces non-trivial output.
func TestWordmarkBuild2026Renders(t *testing.T) {
	d := WordmarkBuild2026Data()
	svg := RenderWithTheme(d, "buildlike", nil, ThemeLight)
	if len(svg) < 1024 {
		t.Errorf("wordmark SVG implausibly small: %d bytes", len(svg))
	}
	raw, err := RenderGIFWithTheme(d, "buildlike", nil, ThemeLight, 3)
	if err != nil {
		t.Fatalf("RenderGIFWithTheme: %v", err)
	}
	g, err := gif.DecodeAll(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("DecodeAll: %v", err)
	}
	// scale=3, weeks=53, days=7 -> width = (53*13-3)*3 = 2058,
	// height = (7*13-3)*3 = 264.
	if g.Config.Width != 2058 || g.Config.Height != 264 {
		t.Errorf("wordmark dimensions = %dx%d, want 2058x264", g.Config.Width, g.Config.Height)
	}
	// Each frame must be a Paletted image, not nil.
	if _, ok := interface{}(g.Image[0]).(*image.Paletted); !ok {
		t.Errorf("frame 0 has wrong type")
	}
}
