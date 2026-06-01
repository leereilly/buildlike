// Package game (autopilot) implements `--copilot-plays`: a deterministic
// pathfinder that picks the next Action for the player so Copilot can drive
// a whole run end-to-end (great for demos, GIF captures, and benchmarking).
//
// The autopilot is intentionally simple — BFS, no learning, no model. Given
// the same `--seed` it must produce the exact same sequence of actions on
// every machine so demo recordings stay reproducible. Keep it that way: no
// time-based tie-breaks, no map iteration order, no float math.
package game

import (
	"github.com/leereilly/commit-crawl/internal/entity"
	"github.com/leereilly/commit-crawl/internal/world"
)

// AutopilotChoice is the next Action the autopilot wants to take, or
// ActWait when no useful move exists (e.g. the only path is blocked by a
// bug and we're at 1 HP). The returned action is safe to feed straight into
// (*Game).Step.
func AutopilotChoice(g *Game) Action {
	if g == nil || g.Floor == nil || g.Player == nil {
		return ActWait
	}
	l := g.Floor.Level
	from := g.Player.Pos

	if from == l.Stairs {
		return ActAscend
	}

	// Build the danger map: tiles adjacent to a *chasing* bug are
	// avoided when we're at low HP, so we don't autopilot ourselves
	// straight into a death.
	danger := map[world.Point]bool{}
	if g.Player.HP <= 2 && !g.Player.Invincible {
		for _, b := range g.Floor.Bugs {
			if !b.Alive {
				continue
			}
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					danger[world.Point{X: b.Pos.X + dx, Y: b.Pos.Y + dy}] = true
				}
			}
		}
	}

	// Targets, in priority order:
	//   1. nearest uneaten powerup, if we're missing HP
	//   2. nearest adjacent bug we can safely bump-attack
	//   3. the stairs
	//
	// Powerups beat bugs because they push HP up and unlock the
	// low-HP risk-aversion branch above.
	var targets []world.Point
	if g.Player.HP < g.Player.MaxHP {
		for _, pu := range g.Floor.Powerups {
			if !pu.Picked {
				targets = append(targets, pu.Pos)
			}
		}
	}
	if len(targets) == 0 {
		for _, b := range g.Floor.Bugs {
			if b.Alive {
				targets = append(targets, b.Pos)
			}
		}
	}
	if len(targets) == 0 {
		targets = []world.Point{l.Stairs}
	}

	step, ok := bfsNextStep(l, from, targets, danger, g.Floor.Bugs)
	if !ok {
		// Retry without the danger constraint — better to fight than
		// to stand still forever.
		step, ok = bfsNextStep(l, from, targets, nil, g.Floor.Bugs)
	}
	if !ok {
		return ActWait
	}
	return actionFor(step)
}

// bfsNextStep returns the first step (dx, dy) of the shortest path from
// `from` to any tile in `targets`, treating any tile in `danger` as
// impassable. Bugs occupy walkable tiles too — landing on a bug's cell
// performs a bump-attack, so the BFS allows stepping onto a bug cell only
// when that cell is itself a target.
func bfsNextStep(
	l *world.Level,
	from world.Point,
	targets []world.Point,
	danger map[world.Point]bool,
	bugs []*entity.Bug,
) (world.Point, bool) {
	if len(targets) == 0 {
		return world.Point{}, false
	}
	target := map[world.Point]bool{}
	for _, t := range targets {
		target[t] = true
	}
	bugAt := map[world.Point]bool{}
	for _, b := range bugs {
		if b.Alive {
			bugAt[b.Pos] = true
		}
	}

	// 8-connected neighborhood, listed in a fixed order so BFS
	// tie-breaks identically across runs. Cardinals first so the
	// path prefers straight-line motion when distances tie.
	deltas := [8]world.Point{
		{X: 0, Y: -1}, {X: 0, Y: 1}, {X: -1, Y: 0}, {X: 1, Y: 0},
		{X: -1, Y: -1}, {X: 1, Y: -1}, {X: -1, Y: 1}, {X: 1, Y: 1},
	}

	type node struct {
		p     world.Point
		first world.Point // the first step taken from `from`
	}

	visited := make([][]bool, l.H)
	for y := range visited {
		visited[y] = make([]bool, l.W)
	}
	visited[from.Y][from.X] = true

	queue := make([]node, 0, 64)
	for _, d := range deltas {
		n := world.Point{X: from.X + d.X, Y: from.Y + d.Y}
		if !l.In(n) || visited[n.Y][n.X] {
			continue
		}
		if !l.Walkable(n) {
			continue
		}
		if danger[n] {
			continue
		}
		if bugAt[n] && !target[n] {
			continue
		}
		visited[n.Y][n.X] = true
		queue = append(queue, node{p: n, first: d})
		if target[n] {
			return d, true
		}
	}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if target[cur.p] {
			return cur.first, true
		}
		for _, d := range deltas {
			n := world.Point{X: cur.p.X + d.X, Y: cur.p.Y + d.Y}
			if !l.In(n) || visited[n.Y][n.X] {
				continue
			}
			if !l.Walkable(n) {
				continue
			}
			if danger[n] {
				continue
			}
			if bugAt[n] && !target[n] {
				continue
			}
			visited[n.Y][n.X] = true
			queue = append(queue, node{p: n, first: cur.first})
		}
	}
	return world.Point{}, false
}

// actionFor maps a unit step (dx, dy in {-1, 0, 1}) to the matching move
// Action. ActWait is returned for the no-op step (0, 0).
func actionFor(step world.Point) Action {
	switch step {
	case world.Point{X: 0, Y: -1}:
		return ActMoveN
	case world.Point{X: 0, Y: 1}:
		return ActMoveS
	case world.Point{X: -1, Y: 0}:
		return ActMoveW
	case world.Point{X: 1, Y: 0}:
		return ActMoveE
	case world.Point{X: -1, Y: -1}:
		return ActMoveNW
	case world.Point{X: 1, Y: -1}:
		return ActMoveNE
	case world.Point{X: -1, Y: 1}:
		return ActMoveSW
	case world.Point{X: 1, Y: 1}:
		return ActMoveSE
	}
	return ActWait
}
