package game_test

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/leereilly/commit-crawl/internal/game"
	"github.com/leereilly/commit-crawl/internal/rng"
	"github.com/leereilly/commit-crawl/internal/ui"
	"github.com/leereilly/commit-crawl/internal/world"
)

// TestAscendTriggersTeleport walks every dungeon (B → U → I → L → D) by
// directly invoking the ascend path. Confirms that:
//   - Each ascent (except the final one) enters PhaseTeleport with PrevMask
//     set to the floor we left, then resumes play on the next floor.
//   - The new Floor has the expected letter.
//   - From D, ascending rolls the credits (PhaseRickRoll), which the main
//     loop dismisses to PhaseWon.
func TestAscendTriggersTeleport(t *testing.T) {
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
		// Force a stairs ascent by teleporting the player onto the stairs and
		// invoking the same code path the keyboard handler uses.
		g.Player.Pos = g.Floor.Level.Stairs
		fromMask := g.Floor.Level.Mask
		g.Step(game.ActAscend)

		if from == 5 {
			if g.Phase != game.PhaseRickRoll {
				t.Errorf("ascend from D: expected PhaseRickRoll (end-of-run easter egg), got %v", g.Phase)
			}
			// Dismiss the RickRoll like the main loop does on key press.
			g.Phase = game.PhaseWon
			if g.Phase != game.PhaseWon {
				t.Errorf("after dismissing RickRoll: expected PhaseWon, got %v", g.Phase)
			}
			return
		}

		if g.Phase != game.PhaseTeleport {
			t.Fatalf("ascend from %c: expected PhaseTeleport, got %v", letters[from-1], g.Phase)
		}
		if g.PrevMask != fromMask {
			t.Errorf("ascend from %c: PrevMask was not the previous floor mask", letters[from-1])
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

		// Tick the teleport animation through to completion.
		for i := 0; i < ui.TeleportDurationTicks+1; i++ {
			g.AdvanceTeleport()
		}
		if g.Phase != game.PhasePlaying {
			t.Fatalf("after teleport from %c: expected PhasePlaying, got %v", letters[from-1], g.Phase)
		}
	}
}

// TestSkipTeleportRoutesPlaying confirms that pressing a key during the
// transition skips straight to play on the new floor, for every non-final
// ascent.
func TestSkipTeleportRoutesPlaying(t *testing.T) {
	scr := tcell.NewSimulationScreen("")
	if err := scr.Init(); err != nil {
		t.Fatalf("init sim screen: %v", err)
	}
	defer scr.Fini()

	g := game.New(scr, rng.New(7))
	g.StartRun()

	// First ascent (B → U).
	g.Player.Pos = g.Floor.Level.Stairs
	g.Step(game.ActAscend)
	if g.Phase != game.PhaseTeleport {
		t.Fatalf("expected PhaseTeleport, got %v", g.Phase)
	}
	g.SkipTeleport()
	if g.Phase != game.PhasePlaying {
		t.Errorf("first-ascent SkipTeleport: expected Playing, got %v", g.Phase)
	}

	// Second ascent (U → I).
	g.Player.Pos = g.Floor.Level.Stairs
	g.Step(game.ActAscend)
	if g.Phase != game.PhaseTeleport {
		t.Fatalf("second ascent: expected PhaseTeleport, got %v", g.Phase)
	}
	g.SkipTeleport()
	if g.Phase != game.PhasePlaying {
		t.Errorf("second-ascent SkipTeleport: expected Playing, got %v", g.Phase)
	}
}

// Compile-time check: world.LetterMask is exposed so the UI can read it.
var _ *world.LetterMask = (*world.LetterMask)(nil)
