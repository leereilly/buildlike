package game

import (
	"testing"

	"github.com/leereilly/commit-crawl/internal/entity"
	"github.com/leereilly/commit-crawl/internal/world"
)

// newAutopilotTestGame builds a tiny hand-rolled level so the autopilot
// tests don't depend on the BSP generator's random output. The map is a
// single 7x3 corridor:
//
//	#########
//	#@.....>#
//	#########
//
// The player spawns on the left and the stairs are on the far right.
func newAutopilotTestGame(t *testing.T) *Game {
	t.Helper()
	l := world.NewLevel(9, 3, 1)
	for x := 1; x < 8; x++ {
		l.Set(world.Point{X: x, Y: 1}, world.TileFloor)
	}
	l.Spawn = world.Point{X: 1, Y: 1}
	l.Stairs = world.Point{X: 7, Y: 1}
	l.Set(l.Stairs, world.TileStairs)

	p := entity.NewPlayer()
	p.Pos = l.Spawn
	return &Game{
		Player: p,
		Floor:  &FloorState{Level: l},
		Phase:  PhasePlaying,
	}
}

func TestAutopilotWalksTowardStairs(t *testing.T) {
	g := newAutopilotTestGame(t)
	got := AutopilotChoice(g)
	if got != ActMoveE {
		t.Fatalf("expected ActMoveE toward stairs, got %v", got)
	}
}

func TestAutopilotAscendsOnStairs(t *testing.T) {
	g := newAutopilotTestGame(t)
	g.Player.Pos = g.Floor.Level.Stairs
	if got := AutopilotChoice(g); got != ActAscend {
		t.Fatalf("expected ActAscend on stairs, got %v", got)
	}
}

func TestAutopilotPrefersPowerupWhenInjured(t *testing.T) {
	g := newAutopilotTestGame(t)
	// Drop max HP up so we're injured, and stick a powerup next to us.
	g.Player.MaxHP = 12
	g.Player.HP = 8
	g.Floor.Powerups = []*entity.Powerup{
		{Pos: world.Point{X: 2, Y: 1}},
	}
	if got := AutopilotChoice(g); got != ActMoveE {
		t.Fatalf("expected ActMoveE toward powerup, got %v", got)
	}
}

func TestAutopilotBumpsAdjacentBug(t *testing.T) {
	g := newAutopilotTestGame(t)
	// Place a bug immediately east of the player; with no powerups
	// missing and the player at full HP, the autopilot should attack
	// the bug on the path to the stairs.
	g.Floor.Bugs = []*entity.Bug{
		entity.NewBug(world.Point{X: 2, Y: 1}),
	}
	if got := AutopilotChoice(g); got != ActMoveE {
		t.Fatalf("expected ActMoveE to bump bug, got %v", got)
	}
}

func TestAutopilotWaitsWhenNoPathExists(t *testing.T) {
	g := newAutopilotTestGame(t)
	// Wall off the corridor immediately east of the player.
	g.Floor.Level.Set(world.Point{X: 2, Y: 1}, world.TileWall)
	if got := AutopilotChoice(g); got != ActWait {
		t.Fatalf("expected ActWait when blocked, got %v", got)
	}
}

func TestAutopilotHandlesNilSafely(t *testing.T) {
	if got := AutopilotChoice(nil); got != ActWait {
		t.Fatalf("expected ActWait for nil game, got %v", got)
	}
	if got := AutopilotChoice(&Game{}); got != ActWait {
		t.Fatalf("expected ActWait for empty game, got %v", got)
	}
}
