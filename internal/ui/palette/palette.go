// Package palette holds the BUILD-logo-derived color values used throughout
// the game. Colors were sampled from the pixelated BUILD logo (rainbow letters
// on pure black) and are exposed as tcell.Color values.
package palette

import "github.com/gdamore/tcell/v2"

var (
	Black         = tcell.NewRGBColor(0, 0, 0)
	Red           = tcell.NewRGBColor(0xE8, 0x41, 0x2A)
	Orange        = tcell.NewRGBColor(0xF0, 0x8A, 0x2C)
	Yellow        = tcell.NewRGBColor(0xF4, 0xC4, 0x30)
	Green         = tcell.NewRGBColor(0x5B, 0xB4, 0x4A)
	Blue          = tcell.NewRGBColor(0x2A, 0x9C, 0xD8)
	Magenta       = tcell.NewRGBColor(0xC8, 0x39, 0x8B)
	DimGray       = tcell.NewRGBColor(0x3A, 0x3A, 0x3A)
	White         = tcell.NewRGBColor(0xEE, 0xEE, 0xEE)
	CopilotCyan   = tcell.NewRGBColor(0x4F, 0xC3, 0xD9)
	CopilotPurple = tcell.NewRGBColor(0xB1, 0x9C, 0xD9)
)

// Cycle is the six-color rainbow used for the Rick Roll color cycle and any
// other "make it sparkle" UI flourish.
var Cycle = []tcell.Color{Red, Orange, Yellow, Green, Blue, Magenta}

// NoColor, when true, makes Style() return the default terminal style so the
// game stays readable on monochrome terminals or when the user passes --no-color.
var NoColor bool

// Style returns a tcell.Style with the given fg/bg, honoring NoColor.
func Style(fg, bg tcell.Color) tcell.Style {
	if NoColor {
		return tcell.StyleDefault
	}
	return tcell.StyleDefault.Foreground(fg).Background(bg)
}

// FG is shorthand for Style(fg, Black).
func FG(fg tcell.Color) tcell.Style { return Style(fg, Black) }
