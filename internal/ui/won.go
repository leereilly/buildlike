package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/leereilly/buildlike/internal/ui/palette"
)

// RenderWon is the post-Rick-Roll victory screen.
func RenderWon(s tcell.Screen, maxHP int, vaulted bool) {
	Clear(s)
	w, h := s.Size()
	lines := []string{
		"You shipped the build.",
		"",
		fmt.Sprintf("Final HP capacity: %d", maxHP),
		"",
		"Press r to run again, q to quit.",
	}
	if vaulted {
		lines[2] += "    (vault: claimed)"
	}
	startY := h/2 - len(lines)/2
	for i, line := range lines {
		st := palette.FG(palette.White)
		if i == 0 {
			st = palette.FG(palette.Green).Bold(true)
		}
		DrawString(s, (w-len(line))/2, startY+i, line, st)
	}
}
