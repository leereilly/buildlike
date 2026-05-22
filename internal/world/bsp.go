package world

import (
	"github.com/leereilly/buildlike/internal/rng"
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

// Generate builds a BSP dungeon on level l using rng r. It guarantees:
//   - room count <= MaxRooms
//   - spawn (l.Spawn) is reachable from stairs (l.Stairs) via floodfill
// On the rare event the connectivity check fails, the caller is expected to
// regenerate.
func Generate(l *Level, r *rng.RNG) bool {
	// Reset
	for y := 0; y < l.H; y++ {
		for x := 0; x < l.W; x++ {
			l.Tiles[y][x] = TileWall
		}
	}
	l.Rooms = nil

	root := &bspNode{rect: Rect{1, 1, l.W - 2, l.H - 2}}
	rooms := 0
	split(root, r, &rooms)
	carveRooms(l, root)
	carveCorridors(l, root, r)

	if len(l.Rooms) == 0 {
		return false
	}
	l.Spawn = l.Rooms[0].Center()
	last := l.Rooms[len(l.Rooms)-1]
	l.Stairs = last.Center()
	l.Set(l.Stairs, TileStairs)

	return l.FloodFillReachable(l.Spawn, l.Stairs)
}

func split(n *bspNode, r *rng.RNG, rooms *int) {
	// If we've already capped rooms, stop subdividing this branch (just make it a leaf room).
	if *rooms >= MaxRooms {
		makeRoom(n, r, rooms)
		return
	}
	w, h := n.rect.W(), n.rect.H()
	// Too small to split — make a room here.
	if w < 2*MinPartSize && h < 2*MinPartSize {
		makeRoom(n, r, rooms)
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
		if h < 2*MinPartSize {
			makeRoom(n, r, rooms)
			return
		}
		cut := r.IntRange(n.rect.Y1+MinPartSize, n.rect.Y2-MinPartSize)
		n.left = &bspNode{rect: Rect{n.rect.X1, n.rect.Y1, n.rect.X2, cut}}
		n.right = &bspNode{rect: Rect{n.rect.X1, cut + 1, n.rect.X2, n.rect.Y2}}
	} else {
		if w < 2*MinPartSize {
			makeRoom(n, r, rooms)
			return
		}
		cut := r.IntRange(n.rect.X1+MinPartSize, n.rect.X2-MinPartSize)
		n.left = &bspNode{rect: Rect{n.rect.X1, n.rect.Y1, cut, n.rect.Y2}}
		n.right = &bspNode{rect: Rect{cut + 1, n.rect.Y1, n.rect.X2, n.rect.Y2}}
	}
	split(n.left, r, rooms)
	split(n.right, r, rooms)
}

func makeRoom(n *bspNode, r *rng.RNG, rooms *int) {
	if *rooms >= MaxRooms {
		return
	}
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
	n.room = &room
	*rooms++
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
// point of each child subtree with an L-shaped corridor. Returns a point
// inside this subtree to use for further connections.
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
	carveL(l, a, b, r)
	if r.Chance(0.5) {
		return a
	}
	return b
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
