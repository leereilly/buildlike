package world

import (
	"github.com/leereilly/commit-crawl/internal/rng"
)

// buildLogo is a 5-row, hand-drawn miniature of the "BUILD" wordmark.
// Each non-space character marks a colored floor pixel; letters are color-coded:
//
//	B=0(Red) U=1(Orange) I=2(Yellow) L=3(Green) D=4(Blue) (indices into palette.Cycle).
//
// A trailing magenta '*' (index 5) is rendered separately as the pressure plate.
var buildLogo = []string{
	"BBB. UUUU III LLLL DDD. ",
	"B..B U..U .I.. L... D..D",
	"BBB. U..U .I.. L... D..D",
	"B..B U..U .I.. L... D..D",
	"BBB. .UU. III LLLL DDD. ",
}

// PlaceBuildVault tries to carve a disconnected secret room on the given level
// that contains a colored BUILD logo and a magenta pressure plate.
// Returns true on success.
func PlaceBuildVault(l *Level, r *rng.RNG) bool {
	const vaultW = 26
	const vaultH = 7

	// Find a spot far from existing rooms.
	for attempt := 0; attempt < 60; attempt++ {
		x1 := r.IntRange(1, l.W-vaultW-2)
		y1 := r.IntRange(1, l.H-vaultH-2)
		x2 := x1 + vaultW - 1
		y2 := y1 + vaultH - 1
		if intersectsAny(Rect{x1, y1, x2, y2}, l.Rooms, 2) {
			continue
		}
		carveVault(l, x1, y1)
		// Place secret door: find a wall tile adjacent to existing corridor
		// that is also adjacent to the vault interior.
		if !placeSecretDoor(l, Rect{x1, y1, x2, y2}) {
			// Clear and retry
			for y := y1; y <= y2; y++ {
				for x := x1; x <= x2; x++ {
					l.Tiles[y][x] = TileWall
				}
			}
			l.VaultColors = nil
			continue
		}
		l.HasVault = true
		return true
	}
	return false
}

func carveVault(l *Level, x1, y1 int) {
	l.VaultColors = map[Point]int{}
	// Floor box
	for y := y1; y < y1+7; y++ {
		for x := x1; x < x1+26; x++ {
			l.Tiles[y][x] = TileFloor
		}
	}
	// Paint logo at +1 inset (centered vertically with 1-row top margin)
	for row, line := range buildLogo {
		for col, ch := range line {
			if ch == ' ' || ch == '.' {
				continue
			}
			var idx int
			switch ch {
			case 'B':
				idx = 0
			case 'U':
				idx = 1
			case 'I':
				idx = 2
			case 'L':
				idx = 3
			case 'D':
				idx = 4
			default:
				continue
			}
			p := Point{X: x1 + 1 + col, Y: y1 + 1 + row}
			l.VaultColors[p] = idx
		}
	}
	// Pressure plate: dead-center of the room.
	plate := Point{X: x1 + 13, Y: y1 + 3}
	// Clear any logo color there so the * is unambiguously magenta.
	delete(l.VaultColors, plate)
	l.Tiles[plate.Y][plate.X] = TilePlate
	l.VaultPlate = plate
}

func placeSecretDoor(l *Level, vault Rect) bool {
	// Walk the vault perimeter; for each wall tile, check if the outside neighbor
	// is a floor (corridor or room). Mark that perimeter tile a SecretDoor.
	candidates := []Point{}
	for x := vault.X1; x <= vault.X2; x++ {
		candidates = append(candidates, Point{x, vault.Y1 - 1}, Point{x, vault.Y2 + 1})
	}
	for y := vault.Y1; y <= vault.Y2; y++ {
		candidates = append(candidates, Point{vault.X1 - 1, y}, Point{vault.X2 + 1, y})
	}
	for _, c := range candidates {
		if !l.In(c) || l.Tiles[c.Y][c.X] != TileWall {
			continue
		}
		// One side floor (the existing dungeon), one side floor (the vault).
		hasOutside := false
		hasInside := false
		for _, d := range [4]Point{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}} {
			n := Point{c.X + d.X, c.Y + d.Y}
			if !l.In(n) {
				continue
			}
			inVault := n.X >= vault.X1 && n.X <= vault.X2 && n.Y >= vault.Y1 && n.Y <= vault.Y2
			if l.Tiles[n.Y][n.X] == TileFloor || l.Tiles[n.Y][n.X] == TileStairs {
				if inVault {
					hasInside = true
				} else {
					hasOutside = true
				}
			}
		}
		if hasOutside && hasInside {
			l.Tiles[c.Y][c.X] = TileSecretDoor
			return true
		}
	}
	return false
}

func intersectsAny(r Rect, rooms []Rect, pad int) bool {
	for _, o := range rooms {
		if r.X1-pad <= o.X2 && r.X2+pad >= o.X1 && r.Y1-pad <= o.Y2 && r.Y2+pad >= o.Y1 {
			return true
		}
	}
	return false
}
