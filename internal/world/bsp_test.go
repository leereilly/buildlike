package world_test

import (
	"fmt"
	"testing"

	"github.com/leereilly/buildlike/internal/rng"
	"github.com/leereilly/buildlike/internal/world"
)

// TestSpawnTopLeft confirms that the spawn point picked by Generate is the
// room whose center has the minimum X+Y (top-left preference) and that the
// stairs are placed in a different room reachable from spawn.
func TestSpawnTopLeft(t *testing.T) {
	for depth := 1; depth <= 5; depth++ {
		for _, seed := range []int64{1, 7, 42, 100, 2024} {
			r := rng.New(seed)
			var l *world.Level
			ok := false
			for attempt := 0; attempt < 24; attempt++ {
				l = world.NewLevel(80, 22, depth)
				l.Mask = world.LetterFor(depth)
				if world.Generate(l, r) {
					ok = true
					break
				}
			}
			if !ok {
				t.Errorf("depth=%d seed=%d: generation failed", depth, seed)
				continue
			}
			minScore := 1 << 30
			for _, room := range l.Rooms {
				c := room.Center()
				if c.X+c.Y < minScore {
					minScore = c.X + c.Y
				}
			}
			if got := l.Spawn.X + l.Spawn.Y; got != minScore {
				t.Errorf("depth=%d seed=%d: spawn (%d,%d) score=%d, want minRoomScore=%d",
					depth, seed, l.Spawn.X, l.Spawn.Y, got, minScore)
			}
			if l.Spawn == l.Stairs {
				t.Errorf("depth=%d seed=%d: spawn equals stairs (%v)", depth, seed, l.Spawn)
			}
			if !l.FloodFillReachable(l.Spawn, l.Stairs) {
				t.Errorf("depth=%d seed=%d: stairs unreachable from spawn", depth, seed)
			}
		}
	}
}

// TestStairsFar checks that the stairs are placed at the BFS-farthest walkable
// tile from spawn. The exact BFS distance must equal the maximum returned by
// bfsDistances — i.e. there is no walkable tile farther than the stairs.
func TestStairsFar(t *testing.T) {
	letters := []byte{'B', 'U', 'I', 'L', 'D'}
	for depth := 1; depth <= 5; depth++ {
		for _, seed := range []int64{1, 7, 42, 100, 2024} {
			r := rng.New(seed)
			var l *world.Level
			for attempt := 0; attempt < 24; attempt++ {
				l = world.NewLevel(80, 22, depth)
				l.Mask = world.LetterFor(depth)
				if world.Generate(l, r) {
					break
				}
			}
			// Compute spawn->everywhere BFS distance in the test (replicates
			// the generator's logic) and confirm that no walkable tile is
			// farther than the stairs.
			dist := bfsAll(l, l.Spawn)
			stairsDist, hasStairs := dist[l.Stairs]
			if !hasStairs {
				t.Errorf("depth=%d seed=%d: stairs unreachable from spawn", depth, seed)
				continue
			}
			for p, d := range dist {
				if d > stairsDist {
					t.Errorf("depth=%d seed=%d: tile %v is %d cells from spawn but stairs (%v) only %d",
						depth, seed, p, d, l.Stairs, stairsDist)
					break
				}
			}
			t.Log(fmt.Sprintf("depth=%d (%c) seed=%d rooms=%d spawn=(%d,%d) stairs=(%d,%d) bfsDist=%d",
				depth, letters[depth-1], seed, len(l.Rooms),
				l.Spawn.X, l.Spawn.Y, l.Stairs.X, l.Stairs.Y, stairsDist))
		}
	}
}

// bfsAll mirrors the generator's BFS over walkable tiles.
func bfsAll(l *world.Level, from world.Point) map[world.Point]int {
	dist := map[world.Point]int{from: 0}
	q := []world.Point{from}
	for len(q) > 0 {
		p := q[0]
		q = q[1:]
		for _, d := range [4]world.Point{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}} {
			n := world.Point{X: p.X + d.X, Y: p.Y + d.Y}
			if !l.In(n) || !l.Walkable(n) {
				continue
			}
			if _, ok := dist[n]; ok {
				continue
			}
			dist[n] = dist[p] + 1
			q = append(q, n)
		}
	}
	return dist
}
