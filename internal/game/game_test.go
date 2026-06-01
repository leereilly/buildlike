package game_test

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/leereilly/commit-crawl/internal/game"
	"github.com/leereilly/commit-crawl/internal/rng"
	"github.com/leereilly/commit-crawl/internal/world"
)

// TestAscendWalksAllFloors walks every dungeon (B → U → I → L → D) by
// directly invoking the ascend path. Confirms that:
//   - Each non-final ascent advances Floor.Level.Depth by one and keeps the
//     game in PhasePlaying on the new letter-shaped floor.
//   - Ascending off the final D floor enters PhaseEndSequence (the
//     post-game spinner / typed-shell finale), which the main loop later
//     hands off to PhaseRickRoll and then PhaseWon.
func TestAscendWalksAllFloors(t *testing.T) {
	scr := tcell.NewSimulationScreen("")
	if err := scr.Init(); err != nil {
		t.Fatalf("init sim screen: %v", err)
	}
	defer scr.Fini()

	g := game.New(scr, rng.New(42))
	g.StartRun()
	if g.Floor.Level.Depth != 1 {
		t.Fatalf("StartRun: expected depth 1, got %d", g.Floor.Level.Depth)
	}

	letters := []byte{'B', 'U', 'I', 'L', 'D'}

	for from := 1; from <= 5; from++ {
		// Force a stairs ascent by teleporting the player onto the stairs
		// and invoking the same code path the keyboard handler uses.
		g.Player.Pos = g.Floor.Level.Stairs
		g.Step(game.ActAscend)

		if from == 5 {
			if g.Phase != game.PhaseEndSequence {
				t.Errorf("ascend from D: expected PhaseEndSequence, got %v", g.Phase)
			}
			return
		}

		if g.Phase != game.PhasePlaying {
			t.Fatalf("ascend from %c: expected PhasePlaying on new floor, got %v", letters[from-1], g.Phase)
		}
		if g.Floor.Level.Depth != from+1 {
			t.Errorf("ascend from %c: expected new depth %d, got %d", letters[from-1], from+1, g.Floor.Level.Depth)
		}
		if g.Floor.Level.Mask == nil || g.Floor.Level.Mask.Letter != letters[from] {
			gotLetter := byte('?')
			if g.Floor.Level.Mask != nil {
				gotLetter = g.Floor.Level.Mask.Letter
			}
			t.Errorf("ascend from %c: expected new mask letter %c, got %c", letters[from-1], letters[from], gotLetter)
		}
	}
}

// TestStartRunAtDepthClamps spot-checks the title-screen triple-tap warp
// path: StartRunAtDepth must drop the player onto exactly the requested
// BUILD floor and clamp out-of-range depths.
func TestStartRunAtDepthClamps(t *testing.T) {
	scr := tcell.NewSimulationScreen("")
	if err := scr.Init(); err != nil {
		t.Fatalf("init sim screen: %v", err)
	}
	defer scr.Fini()

	g := game.New(scr, rng.New(7))
	for _, tc := range []struct {
		give, want int
	}{{0, 1}, {1, 1}, {3, 3}, {5, 5}, {99, 5}} {
		g.StartRunAtDepth(tc.give)
		if g.Floor.Level.Depth != tc.want {
			t.Errorf("StartRunAtDepth(%d): got depth %d, want %d", tc.give, g.Floor.Level.Depth, tc.want)
		}
		if g.Phase != game.PhasePlaying {
			t.Errorf("StartRunAtDepth(%d): expected PhasePlaying, got %v", tc.give, g.Phase)
		}
	}
}

// Compile-time check: world.LetterMask is exposed so the UI can read it.
var _ *world.LetterMask = (*world.LetterMask)(nil)
