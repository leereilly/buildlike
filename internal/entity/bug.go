package entity

import (
	"github.com/leereilly/commit-crawl/internal/rng"
	"github.com/leereilly/commit-crawl/internal/world"
)

type Bug struct {
	Pos     world.Point
	Alive   bool
	Chasing bool
}

func NewBug(p world.Point) *Bug { return &Bug{Pos: p, Alive: true} }

// Act runs the bug's turn. If the bug is adjacent to the player after sight
// check, returns 1 (damage to player). Otherwise returns 0.
func (b *Bug) Act(l *world.Level, player world.Point, occ map[world.Point]bool, r *rng.RNG) int {
	if !b.Alive {
		return 0
	}
	dist := chebyshev(b.Pos, player)
	if dist <= 6 && world.LineOfSight(l, b.Pos, player) {
		b.Chasing = true
	} else if dist > 10 {
		b.Chasing = false
	}

	if b.Chasing {
		// Adjacent? Attack.
		if dist == 1 {
			return 1
		}
		step := greedyStep(b.Pos, player)
		next := world.Point{X: b.Pos.X + step.X, Y: b.Pos.Y + step.Y}
		if l.Walkable(next) && !occ[next] && next != player {
			delete(occ, b.Pos)
			b.Pos = next
			occ[b.Pos] = true
		}
		return 0
	}

	// Wander: 70% idle, 30% random cardinal step.
	if r.Chance(0.7) {
		return 0
	}
	dirs := [4]world.Point{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}}
	d := dirs[r.Intn(4)]
	next := world.Point{X: b.Pos.X + d.X, Y: b.Pos.Y + d.Y}
	if l.Walkable(next) && !occ[next] && next != player {
		delete(occ, b.Pos)
		b.Pos = next
		occ[b.Pos] = true
	}
	return 0
}

func chebyshev(a, b world.Point) int {
	dx := a.X - b.X
	if dx < 0 {
		dx = -dx
	}
	dy := a.Y - b.Y
	if dy < 0 {
		dy = -dy
	}
	if dx > dy {
		return dx
	}
	return dy
}

// greedyStep picks an axis (or diagonal) that reduces distance most.
func greedyStep(from, to world.Point) world.Point {
	sx, sy := 0, 0
	if to.X > from.X {
		sx = 1
	} else if to.X < from.X {
		sx = -1
	}
	if to.Y > from.Y {
		sy = 1
	} else if to.Y < from.Y {
		sy = -1
	}
	return world.Point{X: sx, Y: sy}
}
