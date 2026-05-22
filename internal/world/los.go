package world

// Bresenham line of sight from a to b: returns true iff no wall tile lies
// strictly between them. The endpoints themselves do not block.
func LineOfSight(l *Level, a, b Point) bool {
	dx := abs(b.X - a.X)
	dy := abs(b.Y - a.Y)
	sx, sy := 1, 1
	if a.X >= b.X {
		sx = -1
	}
	if a.Y >= b.Y {
		sy = -1
	}
	err := dx - dy
	x, y := a.X, a.Y
	for {
		if x == b.X && y == b.Y {
			return true
		}
		if !(x == a.X && y == a.Y) {
			if !l.In(Point{x, y}) {
				return false
			}
			t := l.Tiles[y][x]
			// Secret doors block sight even though they sit between rooms.
			if t == TileWall || t == TileSecretDoor {
				return false
			}
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x += sx
		}
		if e2 < dx {
			err += dx
			y += sy
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
