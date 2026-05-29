package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/leereilly/buildlike/internal/ui/palette"
)

var helpLines = []string{
	"BUILDLIKE — CONTROLS",
	"",
	"  Move           arrows, WASD, hjkl, numpad 1-9",
	"  Diagonals      y u b n   (or numpad 7 9 1 3)",
	"  Wait           . or space or 5",
	"  Ascend stairs  >   (while standing on >)",
	"  Help           ?",
	"  Quit           q or Esc",
	"",
	"GLYPHS",
	"  @  you            b  bug (squash it)",
	"  +  powerup        >  staircase up",
	"  #  wall           .  floor",
	"",
	"Five floors spell B-U-I-L-D. Ship the build.",
	"Press any key to return.",
}

func RenderHelp(s tcell.Screen) {
	Clear(s)
	w, h := s.Size()
	startY := h/2 - len(helpLines)/2
	if startY < 1 {
		startY = 1
	}
	for i, line := range helpLines {
		st := palette.FG(palette.White)
		if i == 0 {
			st = palette.FG(palette.Yellow).Bold(true)
		} else if line == "GLYPHS" {
			st = palette.FG(palette.Yellow).Bold(true)
		}
		DrawString(s, (w-len(line))/2, startY+i, line, st)
	}
}

var tombstone = []string{
	"        _______       ",
	"       /       \\      ",
	"      /  R.I.P. \\     ",
	"     /           \\    ",
	"     |      @     |    ",
	"     |  here lies |    ",
	"     |  a builder |    ",
	"     |            |    ",
	"   __|____________|__  ",
}

func RenderDeath(s tcell.Screen, depth, maxHP int) {
	Clear(s)
	w, h := s.Size()
	startY := h/2 - len(tombstone)/2 - 2
	if startY < 1 {
		startY = 1
	}
	for i, line := range tombstone {
		DrawString(s, (w-len(line))/2, startY+i, line, palette.FG(palette.DimGray))
	}
	msg1 := "The build is red."
	msg2 := ""
	switch depth {
	case 1:
		msg2 = "You are dead."
	case 2:
		msg2 = "You couldn't U-turn fast enough."
	case 3:
		msg2 = "Bettr luck next time."
	case 4:
		msg2 = "L-shaped corridor, L-shaped result."
	case 5:
		msg2 = "So close, yet so far."
	}
	prompt := "Press r to restart, q to quit."
	DrawString(s, (w-len(msg1))/2, startY+len(tombstone)+1, msg1, palette.FG(palette.Red).Bold(true))
	DrawString(s, (w-len(msg2))/2, startY+len(tombstone)+2, msg2, palette.FG(palette.Orange))
	DrawString(s, (w-len(prompt))/2, startY+len(tombstone)+4, prompt, palette.FG(palette.White))
	_ = maxHP
}
