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

// DrawString writes a styled string starting at (x, y), advancing one screen
// column per rune (not per byte) so multi-byte runes like ★ render correctly.
func DrawString(s tcell.Screen, x, y int, text string, st tcell.Style) {
	col := 0
	for _, r := range text {
		s.SetContent(x+col, y, r, nil, st)
		col++
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
func RenderGame(s tcell.Screen, p *entity.Player, fs FloorView, log *MessageLog, username string) {
	Clear(s)
	drawHUD(s, p, fs.GetLevel(), username)
	drawMap(s, p, fs)
	drawLogTail(s, log, fs.GetLevel().H)
}

// FloorView is the interface RenderGame needs. We avoid an import cycle with
// the game package by accepting an interface.
type FloorView interface {
	GetLevel() *world.Level
	GetBugs() []*entity.Bug
	GetPowerups() []*entity.Powerup
	GetJester() *entity.Jester
}

func drawHUD(s tcell.Screen, p *entity.Player, l *world.Level, username string) {
	// Top-left: the floor's "Lv N — L" depth label. The letter cycles
	// through B-U-I-L-D and is tinted by the level's brand color so the
	// player sees at a glance which letter of the wordmark they're on.
	x := 1
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
	DrawString(s, x, 0, depth, palette.FG(color).Bold(true))

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
	// Identity badge sits to the LEFT of the HP label: the player's GitHub
	// handle, rendered with the '@' in brand magenta and the username in
	// yellow so it matches the in-game '@' avatar glyph. Falls back to the
	// game title when no username is set (e.g. in tests that bypass
	// PhaseUsername).
	identity := "Buildlike"
	if username != "" {
		identity = "@" + username
	}
	identityW := runeLen(identity)
	// "<identity> HP NN/MM ████████████████████" right-aligned, with a
	// single space between the identity and the HP label.
	startX := w - barW - len(label) - identityW - 1 - 1
	// Keep the right-hand cluster clear of the depth label. depth uses an
	// em-dash so we measure in runes, not bytes.
	minStart := x + runeLen(depth) + 2
	if startX < minStart {
		startX = minStart
	}
	if username != "" {
		DrawRune(s, startX, 0, '@', palette.FG(palette.Magenta).Bold(true))
		DrawString(s, startX+1, 0, username, palette.FG(palette.Yellow).Bold(true))
	} else {
		DrawString(s, startX, 0, identity, palette.FG(palette.Yellow).Bold(true))
	}
	labelX := startX + identityW + 1
	DrawString(s, labelX, 0, label, palette.FG(palette.White))
	bx := labelX + len(label)
	for i := 0; i < barW; i++ {
		if i < filled {
			DrawRune(s, bx+i, 0, '█', palette.FG(hpColor))
		} else {
			DrawRune(s, bx+i, 0, '░', palette.FG(palette.DimGray))
		}
	}
	if p.Invincible {
		// Pulse a rainbow star just past the HP bar so the player has a
		// persistent reminder that the Konami cheat is active.
		c := palette.Cycle[((w*7)%len(palette.Cycle))]
		DrawRune(s, bx+barW+1, 0, '★', palette.FG(c).Bold(true))
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
	// Jester (easter egg)
	if j := fs.GetJester(); j != nil && j.Alive {
		DrawRune(s, j.Pos.X, j.Pos.Y+offsetY, 'j', palette.FG(palette.White).Bold(true))
	}
	// Player
	DrawRune(s, p.Pos.X, p.Pos.Y+offsetY, '@', palette.FG(palette.Yellow).Bold(true))
}

func drawLogTail(s tcell.Screen, log *MessageLog, levelH int) {
	w, h := s.Size()
	// Position the log to the right of the Copilot glyph so it looks like
	// Copilot is speaking. Glyph sits at x=0, y=levelH+2..levelH+5 (4 rows).
	// Place the 2 log lines on the middle two rows of the glyph.
	const mapOriginY = 1
	glyphTop := mapOriginY + levelH + 1
	y0 := glyphTop + 1
	x0 := CopilotArtW + 2 // 2-column gap after the glyph
	if y0+2 > h {
		// Fallback: bottom of the screen if it doesn't fit beside the glyph.
		y0 = h - 2
		x0 = 1
	}
	tail := log.Tail(2)
	bg := palette.Style(palette.White, palette.Black)
	for i := 0; i < 2; i++ {
		for x := x0; x < w; x++ {
			s.SetContent(x, y0+i, ' ', nil, bg)
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
		DrawString(s, x0, y0+i, "> "+e.Text, palette.FG(c))
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
