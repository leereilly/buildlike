package world

// Point is a grid coordinate (x = col, y = row).
type Point struct{ X, Y int }

// Rect is a closed rectangle [X1..X2] x [Y1..Y2] (inclusive).
type Rect struct{ X1, Y1, X2, Y2 int }

func (r Rect) Center() Point { return Point{(r.X1 + r.X2) / 2, (r.Y1 + r.Y2) / 2} }
func (r Rect) W() int        { return r.X2 - r.X1 + 1 }
func (r Rect) H() int        { return r.Y2 - r.Y1 + 1 }

// Level holds the tile grid plus metadata for one dungeon floor.
type Level struct {
	W, H        int
	Tiles       [][]Tile
	Rooms       []Rect
	Spawn       Point
	Stairs      Point
	Depth       int         // 1..5 (B, U, I, L, D)
	Mask        *LetterMask // letter-shaped walkability mask; nil = full rectangle
	HasVault    bool        // legacy easter-egg flag (kept for code paths; not used in the 5-letter flow)
	VaultPlate  Point
	VaultColors map[Point]int
}

func NewLevel(w, h, depth int) *Level {
	l := &Level{W: w, H: h, Depth: depth}
	l.Tiles = make([][]Tile, h)
	for y := 0; y < h; y++ {
		l.Tiles[y] = make([]Tile, w)
		for x := 0; x < w; x++ {
			l.Tiles[y][x] = TileWall
		}
	}
	return l
}

func (l *Level) In(p Point) bool     { return p.X >= 0 && p.X < l.W && p.Y >= 0 && p.Y < l.H }
func (l *Level) At(p Point) Tile     { return l.Tiles[p.Y][p.X] }
func (l *Level) Set(p Point, t Tile) { l.Tiles[p.Y][p.X] = t }

func (l *Level) Walkable(p Point) bool {
	if !l.In(p) {
		return false
	}
	return l.Tiles[p.Y][p.X].Walkable()
}

// FloodFillReachable returns true if `to` is reachable from `from` via walkable tiles.
func (l *Level) FloodFillReachable(from, to Point) bool {
	if !l.In(from) || !l.In(to) {
		return false
	}
	seen := make([][]bool, l.H)
	for y := range seen {
		seen[y] = make([]bool, l.W)
	}
	stack := []Point{from}
	seen[from.Y][from.X] = true
	for len(stack) > 0 {
		p := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if p == to {
			return true
		}
		for _, d := range [4]Point{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}} {
			n := Point{p.X + d.X, p.Y + d.Y}
			if !l.In(n) || seen[n.Y][n.X] {
				continue
			}
			if !l.Tiles[n.Y][n.X].Walkable() {
				continue
			}
			seen[n.Y][n.X] = true
			stack = append(stack, n)
		}
	}
	return false
}
