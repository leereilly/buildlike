// Command buildlike is a charming Go roguelike themed on Microsoft Build.
// You play @, you squash bugs (b), you eat green +s, you climb five letter-
// shaped dungeons spelling BUILD, and the credits roll with a... surprise.
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/leereilly/buildlike/internal/game"
	"github.com/leereilly/buildlike/internal/rng"
	"github.com/leereilly/buildlike/internal/ui"
	"github.com/leereilly/buildlike/internal/ui/palette"
)

func main() {
	seed := flag.Int64("seed", 0, "RNG seed (0 = time-based)")
	noColor := flag.Bool("no-color", false, "disable colors for monochrome terminals")
	flag.Parse()

	palette.NoColor = *noColor

	screen, err := tcell.NewScreen()
	if err != nil {
		fmt.Fprintf(os.Stderr, "buildlike: cannot create screen: %v\n", err)
		os.Exit(1)
	}
	if err := screen.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "buildlike: cannot init screen: %v\n", err)
		os.Exit(1)
	}
	defer screen.Fini()
	screen.SetStyle(palette.Style(palette.White, palette.Black))
	screen.HideCursor()
	screen.Clear()

	r := rng.New(*seed)
	g := game.New(screen, r)

	run(screen, g)
}

func run(screen tcell.Screen, g *game.Game) {
	// Event pump goroutine — converts blocking PollEvent into a channel
	// so we can multiplex with the animation ticker.
	events := make(chan tcell.Event, 8)
	quit := make(chan struct{})
	go func() {
		for {
			ev := screen.PollEvent()
			if ev == nil {
				return
			}
			select {
			case events <- ev:
			case <-quit:
				return
			}
		}
	}()
	defer close(quit)

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	tipIdx := 0
	tipTicker := time.NewTicker(3 * time.Second)
	defer tipTicker.Stop()

	render(screen, g, tipIdx)

	for {
		select {
		case ev := <-events:
			switch ev := ev.(type) {
			case *tcell.EventResize:
				screen.Sync()
			case *tcell.EventKey:
				if handleKey(g, ev) {
					return
				}
			}
			render(screen, g, tipIdx)
		case <-ticker.C:
			g.Tick++
			// Re-render phases that animate.
			if g.Phase == game.PhaseTitle || g.Phase == game.PhaseRickRoll {
				render(screen, g, tipIdx)
			} else if g.Phase == game.PhasePlaying && g.Tick <= g.CopilotBlinkUntil {
				// Copilot blink is active: re-render so the eyes can reopen
				// once the blink expires.
				render(screen, g, tipIdx)
			}
		case <-tipTicker.C:
			tipIdx++
			if g.Phase == game.PhaseTitle {
				render(screen, g, tipIdx)
			}
		}
	}
}

// handleKey routes a key press to the right phase. Returns true when the
// program should exit.
func handleKey(g *game.Game, ev *tcell.EventKey) bool {
	a := game.MapKey(ev)
	// Trigger blink on every keypress regardless of phase.
	g.CopilotBlinkUntil = g.Tick + 3
	switch g.Phase {
	case game.PhaseTitle:
		if a == game.ActQuit {
			return true
		}
		// Konami-code easter egg: while the player is entering a valid
		// prefix of ↑↑↓↓←→←→BA we consume the keystroke and stay on the
		// title screen. Any non-matching key falls through and starts the
		// run normally (so "press any key to descend" still works for
		// non-arrow keys). Once the full sequence is entered, KonamiArmed
		// is latched for the rest of the session and StartRun grants
		// invincibility.
		if !g.KonamiArmed && game.KonamiMatchesSlot(g.KonamiProgress, ev) {
			g.KonamiProgress++
			if g.KonamiProgress >= game.KonamiLen {
				g.KonamiArmed = true
			}
			return false
		}
		g.KonamiProgress = 0
		g.StartRun()
	case game.PhasePlaying:
		if a == game.ActQuit {
			return true
		}
		if g.Floor != nil && g.Floor.Level.Depth == 2 {
			g.ClippyPresses++
		}
		g.Step(a)
	case game.PhaseHelp:
		g.Phase = game.PhasePlaying
	case game.PhaseDead:
		if a == game.ActQuit {
			return true
		}
		if ev.Rune() == 'r' || ev.Rune() == 'R' || a == game.ActConfirm {
			g.StartRun()
		}
	case game.PhaseRickRoll:
		// Any key dismisses the easter egg and shows the victory screen.
		g.Phase = game.PhaseWon
	case game.PhaseWon:
		if a == game.ActQuit {
			return true
		}
		if ev.Rune() == 'r' || ev.Rune() == 'R' || a == game.ActConfirm {
			g.StartRun()
		}
	}
	return false
}

func render(screen tcell.Screen, g *game.Game, tipIdx int) {
	switch g.Phase {
	case game.PhaseTitle:
		tip := ui.Tips[tipIdx%len(ui.Tips)]
		ui.RenderTitle(screen, tip, g.Tick, g.KonamiArmed)
	case game.PhasePlaying:
		ui.RenderGame(screen, g.Player, g.Floor, g.Log)
		levelH := 0
		if g.Floor != nil {
			levelH = g.Floor.Level.H
		}
		ui.RenderCopilot(screen, g.Tick < g.CopilotBlinkUntil, levelH)
		if g.Floor != nil && g.Floor.Level.Depth == 2 {
			ui.RenderClippy(screen, g.ClippyPresses)
		}

	case game.PhaseHelp:
		ui.RenderHelp(screen)
	case game.PhaseDead:
		ui.RenderDeath(screen, g.Floor.Level.Depth, g.Player.MaxHP)
	case game.PhaseRickRoll:
		ui.RenderRickRoll(screen, g.Tick)
	case game.PhaseWon:
		ui.RenderWon(screen, g.Player.MaxHP, g.Player.Vaulted)
	}
	screen.Show()
}
