package contribgraph

import (
	"fmt"
	"image/color"
)

// Theme controls how the contribution graph renders the "empty" (level-0)
// cells that fill the background between contribution days.
//
// The contributing cells always pick from the Build-themed Palette and are
// unaffected by the theme — only the empty grid changes. A Theme whose
// EmptyCell has alpha 0 (the zero value, also exposed as ThemeNone)
// preserves the historical behaviour: empty cells are omitted entirely so
// the background reads as transparent.
type Theme struct {
	// Name identifies the theme in file paths and CLI flags ("light",
	// "dark"). It is also used as the suffix when GenerateFromData
	// derives per-theme output paths.
	Name string
	// EmptyCell is the RGBA fill applied to every level-0 cell. An
	// alpha of 0 means "skip" — no rect is drawn (SVG) and the cell
	// stays transparent (GIF).
	EmptyCell color.RGBA
}

// Transparent reports whether this theme leaves empty cells unrendered.
// Used by both renderers to short-circuit the empty-cell pass.
func (t Theme) Transparent() bool { return t.EmptyCell.A == 0 }

// EmptyCellHex returns the empty-cell colour as a `#rrggbb` string. Only
// the RGB channels are encoded; callers that care about alpha must check
// Transparent first.
func (t Theme) EmptyCellHex() string {
	return fmt.Sprintf("#%02x%02x%02x", t.EmptyCell.R, t.EmptyCell.G, t.EmptyCell.B)
}

// ThemeNone preserves the historical behaviour: empty cells are skipped
// entirely so the rendered image has a transparent background. This is the
// renderer's default when callers use the no-theme entrypoints.
var ThemeNone = Theme{Name: "none"}

// ThemeLight matches GitHub's light-mode empty-cell colour (#ebedf0) and
// renders cleanly on white or near-white README backgrounds.
var ThemeLight = Theme{
	Name:      "light",
	EmptyCell: color.RGBA{R: 0xEB, G: 0xED, B: 0xF0, A: 0xFF},
}

// ThemeDark matches GitHub's default dark-mode empty-cell colour
// (#161b22) and renders cleanly on the dark-mode README background.
var ThemeDark = Theme{
	Name:      "dark",
	EmptyCell: color.RGBA{R: 0x16, G: 0x1B, B: 0x22, A: 0xFF},
}

// ThemeByName resolves the canonical theme for "light", "dark", "none",
// or "" (treated as ThemeNone). Returns an error for any other name so
// CLI typos surface immediately rather than silently rendering blank
// backgrounds.
func ThemeByName(name string) (Theme, error) {
	switch name {
	case "", "none":
		return ThemeNone, nil
	case "light":
		return ThemeLight, nil
	case "dark":
		return ThemeDark, nil
	default:
		return Theme{}, fmt.Errorf("contribgraph: unknown theme %q (want light, dark, or none)", name)
	}
}
