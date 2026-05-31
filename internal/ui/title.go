package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/leereilly/commit-crawl/internal/ui/palette"
)

var titleArt = []string{
	" ██████╗  ██████╗  ███╗   ███╗ ███╗   ███╗ ██╗ ████████╗",
	"██╔════╝ ██╔═══██╗ ████╗ ████║ ████╗ ████║ ██║ ╚══██╔══╝",
	"██║      ██║   ██║ ██╔████╔██║ ██╔████╔██║ ██║    ██║   ",
	"██║      ██║   ██║ ██║╚██╔╝██║ ██║╚██╔╝██║ ██║    ██║   ",
	"╚██████╗ ╚██████╔╝ ██║ ╚═╝ ██║ ██║ ╚═╝ ██║ ██║    ██║   ",
	" ╚═════╝  ╚═════╝  ╚═╝     ╚═╝ ╚═╝     ╚═╝ ╚═╝    ╚═╝   ",
	"                                                        ",
	"      ██████╗ ██████╗   █████╗  ██╗    ██╗ ██╗          ",
	"     ██╔════╝ ██╔══██╗ ██╔══██╗ ██║    ██║ ██║          ",
	"     ██║      ██████╔╝ ███████║ ██║ █╗ ██║ ██║          ",
	"     ██║      ██╔══██╗ ██╔══██║ ██║███╗██║ ██║          ",
	"     ╚██████╗ ██║  ██║ ██║  ██║ ╚███╔███╔╝ ███████╗     ",
	"      ╚═════╝ ╚═╝  ╚═╝ ╚═╝  ╚═╝  ╚══╝╚══╝  ╚══════╝     ",
	"                                                        ",
	"        ~ A charming roguelike themed on Build ~        ",
}

// RenderTitle draws the title screen. `tip` is the rotating bottom hint and
// `tick` drives the gentle rainbow shimmer on the wordmark. When
// `konamiArmed` is true, a rainbow "INVINCIBLE MODE" line is rendered below
// the prompt as visual confirmation that the Konami code was accepted.
func RenderTitle(s tcell.Screen, tip string, tick int, konamiArmed bool) {
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
	prompt := "Press any key to begin. Q to quit."
	DrawString(s, (w-len(prompt))/2, startY+len(titleArt)+2, prompt, palette.FG(palette.White))
	if konamiArmed {
		msg := "★ KONAMI CODE ACCEPTED — INVINCIBLE MODE ARMED ★"
		c := palette.Cycle[(tick/2)%len(palette.Cycle)]
		DrawString(s, (w-runeLen(msg))/2, startY+len(titleArt)+4, msg, palette.FG(c).Bold(true))
	}
	DrawString(s, (w-len(tip))/2, h-3, tip, palette.FG(palette.Magenta))
	credits := "Made with <3 by @LeeReilly and @GitHubCopilot"
	DrawString(s, (w-len(credits))/2, h-1, credits, palette.FG(palette.DimGray))
}

func runeLen(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}
