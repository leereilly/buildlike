package world

import (
	"github.com/leereilly/commit-crawl/internal/rng"
)

const (
	MaxRooms       = 9
	MinPartSize    = 8 // smallest partition we'll split further
	MinRoomW       = 4
	MinRoomH       = 3
)

type bspNode struct {
	rect        Rect
	left, right *bspNode
	room        *Rect
}

// Generate builds a BSP dungeon on level l using rng r. If l.Mask is non-nil,
// rooms are only placed where they fit entirely inside the mask, and corridors
// are routed (via BFS) only through mask cells.
//
// Guarantees on success:
//   - len(l.Rooms) >= 2
//   - spawn (l.Spawn) is reachable from stairs (l.Stairs) via floodfill
//   - no floor tile lies outside the mask (when a mask is provided)
func Generate(l *Level, r *rng.RNG) bool {
	for y := 0; y < l.H; y++ {
		for x := 0; x < l.W; x++ {
			l.Tiles[y][x] = TileWall
		}
	}
	l.Rooms = nil

	// Use the mask's bounding box (if available) as the BSP root, otherwise
	// the full level interior.
	x1, y1, x2, y2 := 1, 1, l.W-2, l.H-2
	if l.Mask != nil {
		mx1, my1, mx2, my2, ok := maskBBox(l.Mask)
		if ok {
			x1, y1, x2, y2 = mx1, my1, mx2, my2
		}
	}
	root := &bspNode{rect: Rect{x1, y1, x2, y2}}
	rooms := 0
	split(root, r, &rooms, l.Mask)
	carveRooms(l, root)
	carveCorridors(l, root, r)

	// Defensive clip: anything outside the mask must remain wall.
	if l.Mask != nil {
		for y := 0; y < l.H; y++ {
			for x := 0; x < l.W; x++ {
				if l.Tiles[y][x] != TileWall && !l.Mask.Contains(Point{x, y}) {
					l.Tiles[y][x] = TileWall
				}
			}
		}
	}

	if len(l.Rooms) < 2 {
		return false
	}
	l.Spawn = l.Rooms[0].Center()
	last := l.Rooms[len(l.Rooms)-1]
	l.Stairs = last.Center()
	l.Set(l.Stairs, TileStairs)

	return l.FloodFillReachable(l.Spawn, l.Stairs)
}

func maskBBox(m *LetterMask) (x1, y1, x2, y2 int, ok bool) {
	x1, y1 = m.W, m.H
	x2, y2 = -1, -1
	for y := 0; y < m.H; y++ {
		for x := 0; x < m.W; x++ {
			if !m.cells[y][x] {
				continue
			}
			if x < x1 {
				x1 = x
			}
			if y < y1 {
				y1 = y
			}
			if x > x2 {
				x2 = x
			}
			if y > y2 {
				y2 = y
			}
		}
	}
	if x2 < 0 {
		return 0, 0, 0, 0, false
	}
	return x1, y1, x2, y2, true
}

func split(n *bspNode, r *rng.RNG, rooms *int, mask *LetterMask) {
	// If we've already capped rooms, stop subdividing this branch (just make it a leaf room).
	if *rooms >= MaxRooms {
		makeRoom(n, r, rooms, mask)
		return
	}
	w, h := n.rect.W(), n.rect.H()
	// Need at least 2*MinPartSize+1 in some dimension to safely pick a cut.
	if w < 2*MinPartSize+1 && h < 2*MinPartSize+1 {
		makeRoom(n, r, rooms, mask)
		return
	}
	// Choose split direction biased by aspect ratio.
	splitH := r.Chance(0.5)
	if w > h && float64(w)/float64(h) >= 1.25 {
		splitH = false // split vertically (left/right)
	} else if h > w && float64(h)/float64(w) >= 1.25 {
		splitH = true
	}

	if splitH {
		if h < 2*MinPartSize+1 {
			makeRoom(n, r, rooms, mask)
			return
		}
		cut := r.IntRange(n.rect.Y1+MinPartSize, n.rect.Y2-MinPartSize)
		n.left = &bspNode{rect: Rect{n.rect.X1, n.rect.Y1, n.rect.X2, cut}}
		n.right = &bspNode{rect: Rect{n.rect.X1, cut + 1, n.rect.X2, n.rect.Y2}}
	} else {
		if w < 2*MinPartSize+1 {
			makeRoom(n, r, rooms, mask)
			return
		}
		cut := r.IntRange(n.rect.X1+MinPartSize, n.rect.X2-MinPartSize)
		n.left = &bspNode{rect: Rect{n.rect.X1, n.rect.Y1, cut, n.rect.Y2}}
		n.right = &bspNode{rect: Rect{cut + 1, n.rect.Y1, n.rect.X2, n.rect.Y2}}
	}
	split(n.left, r, rooms, mask)
	split(n.right, r, rooms, mask)
}

func makeRoom(n *bspNode, r *rng.RNG, rooms *int, mask *LetterMask) {
	if *rooms >= MaxRooms {
		return
	}
	// Try up to 8 random rectangles inside the partition; accept the first one
	// that lies entirely inside the mask (if any).
	for attempt := 0; attempt < 8; attempt++ {
		rw := r.IntRange(MinRoomW, max(MinRoomW, n.rect.W()-2))
		rh := r.IntRange(MinRoomH, max(MinRoomH, n.rect.H()-2))
		if rw > n.rect.W()-2 {
			rw = n.rect.W() - 2
		}
		if rh > n.rect.H()-2 {
			rh = n.rect.H() - 2
		}
		if rw < MinRoomW || rh < MinRoomH {
			return
		}
		rx := r.IntRange(n.rect.X1+1, n.rect.X2-rw)
		ry := r.IntRange(n.rect.Y1+1, n.rect.Y2-rh)
		room := Rect{rx, ry, rx + rw - 1, ry + rh - 1}
		if !roomInMask(room, mask) {
			continue
		}
		n.room = &room
		*rooms++
		return
	}
}

// roomInMask returns true if every cell in `room` is contained in `mask`
// (or mask is nil).
func roomInMask(room Rect, mask *LetterMask) bool {
	if mask == nil {
		return true
	}
	for y := room.Y1; y <= room.Y2; y++ {
		for x := room.X1; x <= room.X2; x++ {
			if !mask.Contains(Point{x, y}) {
				return false
			}
		}
	}
	return true
}

func carveRooms(l *Level, n *bspNode) {
	if n == nil {
		return
	}
	if n.room != nil {
		for y := n.room.Y1; y <= n.room.Y2; y++ {
			for x := n.room.X1; x <= n.room.X2; x++ {
				l.Tiles[y][x] = TileFloor
			}
		}
		l.Rooms = append(l.Rooms, *n.room)
	}
	carveRooms(l, n.left)
	carveRooms(l, n.right)
}

// carveCorridors walks the BSP tree post-order, joining the representative
// point of each child subtree with a path. When the level has a mask, the
// path is found via BFS confined to mask cells; otherwise a classic L-shape
// is used. Returns a point inside this subtree to use for further connections.
func carveCorridors(l *Level, n *bspNode, r *rng.RNG) Point {
	if n == nil {
		return Point{}
	}
	if n.room != nil {
		return n.room.Center()
	}
	a := carveCorridors(l, n.left, r)
	b := carveCorridors(l, n.right, r)
	if a == (Point{}) {
		return b
	}
	if b == (Point{}) {
		return a
	}
	if l.Mask != nil {
		carvePath(l, a, b, l.Mask)
	} else {
		carveL(l, a, b, r)
	}
	if r.Chance(0.5) {
		return a
	}
	return b
}

// carvePath finds a shortest path from a to b through mask cells using BFS
// and carves wall tiles along the path into TileFloor.
func carvePath(l *Level, a, b Point, mask *LetterMask) bool {
	if !l.In(a) || !l.In(b) {
		return false
	}
	if !mask.Contains(a) || !mask.Contains(b) {
		return false
	}
	prev := make(map[Point]Point, 64)
	visited := make(map[Point]bool, 64)
	visited[a] = true
	q := []Point{a}
	found := false
	for len(q) > 0 {
		p := q[0]
		q = q[1:]
		if p == b {
			found = true
			break
		}
		for _, d := range [4]Point{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			n := Point{p.X + d.X, p.Y + d.Y}
			if !l.In(n) || visited[n] || !mask.Contains(n) {
				continue
			}
			visited[n] = true
			prev[n] = p
			q = append(q, n)
		}
	}
	if !found {
		return false
	}
	cur := b
	for {
		if l.Tiles[cur.Y][cur.X] == TileWall {
			l.Tiles[cur.Y][cur.X] = TileFloor
		}
		if cur == a {
			break
		}
		cur = prev[cur]
	}
	return true
}

func carveL(l *Level, a, b Point, r *rng.RNG) {
	if r.Chance(0.5) {
		carveH(l, a.X, b.X, a.Y)
		carveV(l, a.Y, b.Y, b.X)
	} else {
		carveV(l, a.Y, b.Y, a.X)
		carveH(l, a.X, b.X, b.Y)
	}
}

func carveH(l *Level, x1, x2, y int) {
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	for x := x1; x <= x2; x++ {
		if l.In(Point{x, y}) && l.Tiles[y][x] == TileWall {
			l.Tiles[y][x] = TileFloor
		}
	}
}

func carveV(l *Level, y1, y2, x int) {
	if y1 > y2 {
		y1, y2 = y2, y1
	}
	for y := y1; y <= y2; y++ {
		if l.In(Point{x, y}) && l.Tiles[y][x] == TileWall {
			l.Tiles[y][x] = TileFloor
		}
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
