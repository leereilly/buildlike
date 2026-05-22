package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/leereilly/buildlike/internal/ui/palette"
)

var titleArt = []string{
	"  ██████╗ ██╗   ██╗██╗██╗     ██████╗ ",
	"  ██╔══██╗██║   ██║██║██║     ██╔══██╗",
	"  ██████╔╝██║   ██║██║██║     ██║  ██║",
	"  ██╔══██╗██║   ██║██║██║     ██║  ██║",
	"  ██████╔╝╚██████╔╝██║███████╗██████╔╝",
	"  ╚═════╝  ╚═════╝ ╚═╝╚══════╝╚═════╝ ",
	"                                      ",
	"        ~ a charming roguelike ~      ",
}

// RenderTitle draws the title screen. `tip` is the rotating bottom hint and
// `tick` drives the gentle rainbow shimmer on the wordmark.
func RenderTitle(s tcell.Screen, tip string, tick int) {
	Clear(s)
	w, h := s.Size()
	startY := h/2 - len(titleArt)/2 - 2
	if startY < 1 {
		startY = 1
	}
	for row, line := range titleArt {
		startX := (w - runeLen(line)) / 2
		col := 0
		for _, r := range line {
			// Color cycles per-character per-tick so the wordmark "shimmers".
			c := palette.Cycle[(col+row+tick/4)%len(palette.Cycle)]
			st := palette.FG(c).Bold(true)
			if r == ' ' || r == '\u00A0' {
				st = palette.Style(palette.White, palette.Black)
			}
			DrawRune(s, startX+col, startY+row, r, st)
			col++
		}
	}
	prompt := "Press any key to descend.   q to quit."
	DrawString(s, (w-len(prompt))/2, startY+len(titleArt)+2, prompt, palette.FG(palette.White))
	DrawString(s, (w-len(tip))/2, h-2, tip, palette.FG(palette.Magenta))
}

func runeLen(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}
