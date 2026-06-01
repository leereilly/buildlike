package entity

import (
	"github.com/leereilly/commit-crawl/internal/rng"
	"github.com/leereilly/commit-crawl/internal/world"
)

// Jester is a rare, single-floor easter-egg encounter (a nod to Limmy's
// "Adventure Call" Falconhoof sketch — "Kill jester!"). It appears as a white
// 'j' on exactly one randomly chosen floor per run. Behaviour is bug-like: it
// homes in on the player and bump-attacks for 1 HP.
type Jester struct {
	Pos     world.Point
	Alive   bool
	Chasing bool
}

func NewJester(p world.Point) *Jester { return &Jester{Pos: p, Alive: true} }

// Act runs the jester's turn. Mirrors Bug.Act: chases on sight within 6 tiles,
// gives up beyond 10, deals 1 damage when adjacent.
func (j *Jester) Act(l *world.Level, player world.Point, occ map[world.Point]bool, r *rng.RNG) int {
	if !j.Alive {
		return 0
	}
	dist := jesterChebyshev(j.Pos, player)
	if dist <= 6 && world.LineOfSight(l, j.Pos, player) {
		j.Chasing = true
	} else if dist > 10 {
		j.Chasing = false
	}

	if j.Chasing {
		if dist == 1 {
			return 1
		}
		step := jesterGreedyStep(j.Pos, player)
		next := world.Point{X: j.Pos.X + step.X, Y: j.Pos.Y + step.Y}
		if l.Walkable(next) && !occ[next] && next != player {
			delete(occ, j.Pos)
			j.Pos = next
			occ[j.Pos] = true
		}
		return 0
	}

	if r.Chance(0.7) {
		return 0
	}
	dirs := [4]world.Point{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}}
	d := dirs[r.Intn(4)]
	next := world.Point{X: j.Pos.X + d.X, Y: j.Pos.Y + d.Y}
	if l.Walkable(next) && !occ[next] && next != player {
		delete(occ, j.Pos)
		j.Pos = next
		occ[j.Pos] = true
	}
	return 0
}

func jesterChebyshev(a, b world.Point) int {
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

func jesterGreedyStep(from, to world.Point) world.Point {
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
