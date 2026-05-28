package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/leereilly/buildlike/internal/ui/palette"
)

// Clippy is an easter-egg helper that pops up in the bottom-right corner on
// the second floor of a run. The player gets ClippyFreeKeyPresses worth of
// full-brightness keystrokes; after that, each subsequent keystroke fades
// the art one shade closer to black until it disappears entirely.
const (
	ClippyFreeKeyPresses = 5
	ClippyFadeSteps      = 5
)

// clippyArt is the paperclip helper plus its speech bubble. Each line is the
// same character width so the bounding box stays rectangular. Spaces are
// treated as transparent during rendering so the dungeon shows through.
var clippyArt = []string{
	` __                        `,
	`/  \        _______________ `,
	`|  |       /               \`,
	`@  @       | Looks like    |`,
	`|| ||      | you're trying |`,
	`|| ||   <--| to reach the  |`,
	`|\_/|      | final level.  |`,
	`\___/      \_______________/`,
}

// RenderClippy overlays the Clippy helper in the bottom-right corner of the
// screen. `presses` is the number of key presses the player has made so far
// on the current (first) floor. The art renders at full brightness for the
// first ClippyFreeKeyPresses presses, then linearly fades toward black over
// the next ClippyFadeSteps presses, after which nothing is drawn. On
// terminals that are too small to accommodate the art the overlay is
// silently skipped.
func RenderClippy(s tcell.Screen, presses int) {
	fade := presses - ClippyFreeKeyPresses
	if fade < 0 {
		fade = 0
	}
	if fade >= ClippyFadeSteps {
		return
	}
	alpha := 1.0 - float64(fade)/float64(ClippyFadeSteps)

	w, h := s.Size()
	artH := len(clippyArt)
	artW := 0
	for _, line := range clippyArt {
		if n := runeLen(line); n > artW {
			artW = n
		}
	}
	originX := w - artW - 1
	originY := h - artH - 3
	if originX < 1 || originY < 1 {
		return
	}

	style := palette.FG(clippyFadeColor(palette.White, alpha)).Bold(true)
	for dy, line := range clippyArt {
		col := 0
		for _, r := range line {
			if r != ' ' {
				DrawRune(s, originX+col, originY+dy, r, style)
			}
			col++
		}
	}
}

// clippyFadeColor returns c scaled linearly toward black by `alpha`
// (1.0 = c, 0.0 = black). Used to dim the Clippy overlay as the player
// presses keys past the grace window.
func clippyFadeColor(c tcell.Color, alpha float64) tcell.Color {
	if alpha >= 1.0 {
		return c
	}
	if alpha <= 0.0 {
		return palette.Black
	}
	r, g, b := c.RGB()
	fr := int32(float64(r) * alpha)
	fg := int32(float64(g) * alpha)
	fb := int32(float64(b) * alpha)
	return tcell.NewRGBColor(fr, fg, fb)
}
