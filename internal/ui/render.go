package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/leereilly/buildlike/internal/entity"
	"github.com/leereilly/buildlike/internal/ui/palette"
	"github.com/leereilly/buildlike/internal/world"
)

// DrawRune writes a single styled rune at (x, y).
func DrawRune(s tcell.Screen, x, y int, r rune, st tcell.Style) {
	s.SetContent(x, y, r, nil, st)
}

// DrawString writes a styled string starting at (x, y).
func DrawString(s tcell.Screen, x, y int, text string, st tcell.Style) {
	for i, r := range text {
		s.SetContent(x+i, y, r, nil, st)
	}
}

// Clear blanks the screen.
func Clear(s tcell.Screen) {
	w, h := s.Size()
	bg := palette.Style(palette.White, palette.Black)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			s.SetContent(x, y, ' ', nil, bg)
		}
	}
}

// RenderGame draws the full playing-state UI: HUD, map, log.
func RenderGame(s tcell.Screen, p *entity.Player, fs FloorView, log *MessageLog) {
	Clear(s)
	drawHUD(s, p, fs.GetLevel())
	drawMap(s, p, fs)
	drawLogTail(s, log)
}

// FloorView is the interface RenderGame needs. We avoid an import cycle with
// the game package by accepting an interface.
type FloorView interface {
	GetLevel() *world.Level
	GetBugs() []*entity.Bug
	GetPowerups() []*entity.Powerup
}

func drawHUD(s tcell.Screen, p *entity.Player, l *world.Level) {
	title := "Buildlike"
	DrawString(s, 1, 0, title, palette.FG(palette.Yellow).Bold(true))
	letter := byte('?')
	letters := []byte{'B', 'U', 'I', 'L', 'D'}
	if l.Depth >= 1 && l.Depth <= len(letters) {
		letter = letters[l.Depth-1]
	}
	depth := fmt.Sprintf("Lv %d — %c", l.Depth, letter)
	color := palette.Magenta
	if l.Mask != nil {
		color = palette.Cycle[l.Mask.ColorIx]
	}
	DrawString(s, 12, 0, depth, palette.FG(color).Bold(true))

	// HP bar
	const barW = 20
	filled := 0
	if p.MaxHP > 0 {
		filled = p.HP * barW / p.MaxHP
		if filled < 0 {
			filled = 0
		}
		if filled > barW {
			filled = barW
		}
	}
	frac := float64(p.HP) / float64(maxInt(p.MaxHP, 1))
	hpColor := palette.Green
	switch {
	case frac < 0.25:
		hpColor = palette.Red
	case frac < 0.5:
		hpColor = palette.Orange
	case frac < 0.75:
		hpColor = palette.Yellow
	}
	w, _ := s.Size()
	label := fmt.Sprintf("HP %2d/%-2d ", p.HP, p.MaxHP)
	startX := w - barW - len(label) - 2
	if startX < 25 {
		startX = 25
	}
	DrawString(s, startX, 0, label, palette.FG(palette.White))
	bx := startX + len(label)
	for i := 0; i < barW; i++ {
		if i < filled {
			DrawRune(s, bx+i, 0, '█', palette.FG(hpColor))
		} else {
			DrawRune(s, bx+i, 0, '░', palette.FG(palette.DimGray))
		}
	}
}

func drawMap(s tcell.Screen, p *entity.Player, fs FloorView) {
	l := fs.GetLevel()
	const offsetY = 1
	for y := 0; y < l.H; y++ {
		for x := 0; x < l.W; x++ {
			t := l.Tiles[y][x]
			r := t.Glyph()
			var st tcell.Style
			switch t {
			case world.TileWall, world.TileSecretDoor:
				// Tint mask-perimeter walls in the floor's letter color so the
				// silhouette of the letter pops.
				st = palette.FG(palette.Blue)
				if l.Mask != nil {
					p := world.Point{X: x, Y: y}
					if isLetterOutline(l, p) {
						st = palette.FG(palette.Cycle[l.Mask.ColorIx]).Bold(true)
					}
				}
			case world.TileFloor:
				// Vault coloring?
				if idx, ok := l.VaultColors[world.Point{X: x, Y: y}]; ok && idx >= 0 && idx < len(palette.Cycle) {
					// Render a colored floor pixel as a solid block.
					DrawRune(s, x, y+offsetY, '█', palette.FG(palette.Cycle[idx]))
					continue
				}
				st = palette.FG(palette.DimGray)
			case world.TileStairs:
				st = palette.FG(palette.Orange).Bold(true)
			case world.TilePlate:
				st = palette.FG(palette.Magenta).Bold(true)
			default:
				st = palette.FG(palette.DimGray)
			}
			DrawRune(s, x, y+offsetY, r, st)
		}
	}
	// Powerups
	for _, pu := range fs.GetPowerups() {
		if pu.Picked {
			continue
		}
		DrawRune(s, pu.Pos.X, pu.Pos.Y+offsetY, '+', palette.FG(palette.Green).Bold(true))
	}
	// Bugs
	for _, b := range fs.GetBugs() {
		if !b.Alive {
			continue
		}
		DrawRune(s, b.Pos.X, b.Pos.Y+offsetY, 'b', palette.FG(palette.Red).Bold(true))
	}
	// Player
	DrawRune(s, p.Pos.X, p.Pos.Y+offsetY, '@', palette.FG(palette.Yellow).Bold(true))
}

func drawLogTail(s tcell.Screen, log *MessageLog) {
	w, h := s.Size()
	// Log area: bottom 2 rows.
	y0 := h - 2
	tail := log.Tail(2)
	for i := 0; i < 2; i++ {
		// clear line
		for x := 0; x < w; x++ {
			s.SetContent(x, y0+i, ' ', nil, palette.Style(palette.White, palette.Black))
		}
		if i >= len(tail) {
			continue
		}
		e := tail[i]
		var c tcell.Color
		switch e.Kind {
		case LogGood:
			c = palette.Green
		case LogBad:
			c = palette.Red
		case LogWarn:
			c = palette.Orange
		case LogSpecial:
			c = palette.Magenta
		default:
			c = palette.White
		}
		DrawString(s, 1, y0+i, "> "+e.Text, palette.FG(c))
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// isLetterOutline returns true if p is a wall cell that sits on the boundary
// between inside-mask and outside-mask area — i.e. it is the silhouette of
// the letter, viewed from inside the dungeon.
func isLetterOutline(l *world.Level, p world.Point) bool {
	if l.Mask == nil || !l.Mask.Contains(p) {
		return false
	}
	for _, d := range [4]world.Point{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		n := world.Point{X: p.X + d.X, Y: p.Y + d.Y}
		if !l.Mask.Contains(n) {
			return true
		}
	}
	return false
}
