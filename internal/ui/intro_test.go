package ui_test

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/leereilly/commit-crawl/internal/entity"
	"github.com/leereilly/commit-crawl/internal/ui"
	"github.com/leereilly/commit-crawl/internal/world"
)

// stubFloor satisfies ui.FloorView with a minimal generated level. Mirrors
// the pattern teleport_test.go uses to exercise renderers headlessly.
type stubFloor struct {
	level *world.Level
}

func (f *stubFloor) GetLevel() *world.Level         { return f.level }
func (f *stubFloor) GetBugs() []*entity.Bug         { return nil }
func (f *stubFloor) GetPowerups() []*entity.Powerup { return nil }
func (f *stubFloor) GetJester() *entity.Jester      { return nil }

// TestRenderIntroSmoke confirms RenderIntro doesn't panic across the full
// animation timeline, including the pre-hold beat, the bloom mid-frame, and
// the post-completion overrun the main loop tolerates before AdvanceIntro
// flips the phase.
func TestRenderIntroSmoke(t *testing.T) {
	s := tcell.NewSimulationScreen("")
	if err := s.Init(); err != nil {
		t.Fatalf("init sim screen: %v", err)
	}
	defer s.Fini()
	s.SetSize(80, 24)

	lvl := world.NewLevel(80, 22, 1)
	lvl.Mask = world.LetterFor(1)
	lvl.Spawn = world.Point{X: 10, Y: 10}
	for y := 0; y < lvl.H; y++ {
		for x := 0; x < lvl.W; x++ {
			lvl.Tiles[y][x] = world.TileFloor
		}
	}

	fs := &stubFloor{level: lvl}
	p := entity.NewPlayer()
	p.Pos = lvl.Spawn

	st := &ui.IntroState{SrcX: 40, SrcY: 12, StartTick: 0}

	for tick := 0; tick <= ui.IntroDurationTicks+5; tick++ {
		ui.RenderIntro(s, st, p, fs, tick)
	}
}

// TestIntroDoneTransitions covers the timeline gate used by AdvanceIntro:
// the intro must not report "done" while still animating, and must report
// "done" once tick - StartTick reaches IntroDurationTicks.
func TestIntroDoneTransitions(t *testing.T) {
	st := &ui.IntroState{SrcX: 5, SrcY: 5, StartTick: 100}
	if ui.IntroDone(st, 100) {
		t.Errorf("IntroDone at start tick should be false")
	}
	if ui.IntroDone(st, 100+ui.IntroDurationTicks-1) {
		t.Errorf("IntroDone one tick before completion should be false")
	}
	if !ui.IntroDone(st, 100+ui.IntroDurationTicks) {
		t.Errorf("IntroDone at completion tick should be true")
	}
	if !ui.IntroDone(nil, 0) {
		t.Errorf("nil IntroState should always be done")
	}
}

// TestUsernameAtPosMatchesLayout confirms the helper that the intro uses to
// compute the starting '@' position matches the actual screen cell where
// RenderUsername draws the magenta '@'.
func TestUsernameAtPosMatchesLayout(t *testing.T) {
	cases := []struct{ w, h int }{
		{80, 24},
		{120, 40},
		{60, 10}, // tiny screen exercises the min-height clamp
	}
	for _, c := range cases {
		x, y := ui.UsernameAtPos(c.w, c.h)
		s := tcell.NewSimulationScreen("")
		if err := s.Init(); err != nil {
			t.Fatalf("init sim screen: %v", err)
		}
		s.SetSize(c.w, c.h)
		ui.RenderUsername(s, "leereilly", 0, true)
		got, _, _, _ := s.GetContent(x, y)
		if got != '@' {
			t.Errorf("UsernameAtPos(%d,%d)=(%d,%d) but screen has %q, not '@'", c.w, c.h, x, y, got)
		}
		s.Fini()
	}
}
