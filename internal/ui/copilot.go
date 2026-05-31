package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/leereilly/commit-crawl/internal/ui/palette"
)

// Copilot mascot rendered using the exact CLI banner glyphs, positioned
// underneath the level map. Blinks (hides the green eyes) on every keypress.

const (
	CopilotArtW = 10 // widest row in runes (exported for layout in render.go)
	CopilotArtH = 4
	copilotArtW = CopilotArtW
	copilotArtH = CopilotArtH
	mapWidth    = 80 // fixed map width used for centering
)

// CLI banner glyph rows — exact Unicode art from `gh copilot` banner.
var copilotRow1 = "  ╭─╮╭─╮"
var copilotRow2 = "  ╰─╯╰─╯"
var copilotRow3Open = "  █ ▘▝ █"
var copilotRow3Blink = "  █    █" // eyes hidden during blink
var copilotRow4 = "   ▔▔▔▔ "

// RenderCopilot draws the Copilot CLI character flush to the left margin,
// one blank row beneath the level map. levelH is the height of the current
// level (in rows). When blink is true the green eyes are hidden.
func RenderCopilot(s tcell.Screen, blink bool, levelH int) {
	w, h := s.Size()

	// Position: flush left, 1 row below the map bottom (single blank line).
	// Map starts at y=1 (offsetY in drawMap).
	const mapOriginY = 1
	originY := mapOriginY + levelH + 1
	originX := 0

	if originY+copilotArtH > h || originX+copilotArtW > w {
		return
	}

	// Clear background behind the mascot.
	bg := palette.Style(palette.White, palette.Black)
	for dy := 0; dy < copilotArtH; dy++ {
		for dx := 0; dx < copilotArtW; dx++ {
			DrawRune(s, originX+dx, originY+dy, ' ', bg)
		}
	}

	cyanStyle := palette.FG(palette.CopilotCyan).Bold(true)
	purpleStyle := palette.FG(palette.CopilotPurple).Bold(true)
	greenStyle := palette.FG(palette.Green).Bold(true)

	// Row 1 & 2: cyan top boxes
	drawCopilotRow(s, originX, originY+0, copilotRow1, cyanStyle)
	drawCopilotRow(s, originX, originY+1, copilotRow2, cyanStyle)

	// Row 3: body in purple/magenta, eyes in green (hidden when blinking)
	row3 := copilotRow3Open
	if blink {
		row3 = copilotRow3Blink
	}
	col := 0
	for _, r := range row3 {
		if r != ' ' {
			st := purpleStyle
			if r == '▘' || r == '▝' {
				st = greenStyle
			}
			DrawRune(s, originX+col, originY+2, r, st)
		}
		col++
	}

	// Row 4: purple/magenta frame
	drawCopilotRow(s, originX, originY+3, copilotRow4, purpleStyle)
}

func drawCopilotRow(s tcell.Screen, x, y int, row string, st tcell.Style) {
	col := 0
	for _, r := range row {
		if r != ' ' {
			DrawRune(s, x+col, y, r, st)
		}
		col++
	}
}
