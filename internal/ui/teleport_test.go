package ui_test

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/leereilly/buildlike/internal/ui"
	"github.com/leereilly/buildlike/internal/world"
)

// TestRenderTeleportSmoke confirms RenderTeleport doesn't panic for any of
// the BUILD letter pairings (B→U, U→I, I→L, L→D) and across all animation
// ticks, including the edge case of a nil "previous" mask. It uses tcell's
// simulation screen so it runs headless in CI.
func TestRenderTeleportSmoke(t *testing.T) {
	s := tcell.NewSimulationScreen("")
	if err := s.Init(); err != nil {
		t.Fatalf("init sim screen: %v", err)
	}
	defer s.Fini()
	s.SetSize(80, 24)

	cases := []struct {
		prevDepth, nextDepth int
	}{
		{0, 1}, // first teleport (nil prev)
		{1, 2}, // B → U
		{2, 3}, // U → I
		{3, 4}, // I → L
		{4, 5}, // L → D
	}

	for _, c := range cases {
		var prev *world.LetterMask
		if c.prevDepth >= 1 {
			prev = world.LetterFor(c.prevDepth)
		}
		next := world.LetterFor(c.nextDepth)
		for tick := 0; tick <= ui.TeleportDurationTicks+5; tick++ {
			ui.RenderTeleport(s, prev, next, tick)
		}
	}
}

// TestRenderTeleportTinyScreen exercises the small-terminal fallback path.
func TestRenderTeleportTinyScreen(t *testing.T) {
	s := tcell.NewSimulationScreen("")
	if err := s.Init(); err != nil {
		t.Fatalf("init sim screen: %v", err)
	}
	defer s.Fini()
	s.SetSize(40, 10)
	ui.RenderTeleport(s, world.LetterFor(1), world.LetterFor(2), 5)
}
