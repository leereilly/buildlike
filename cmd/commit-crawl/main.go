// Command commit-crawl is a charming Go roguelike themed on Microsoft Build.
// You play @, you squash bugs (b), you eat green +s, you climb five letter-
// shaped dungeons spelling BUILD, and the credits roll with a... surprise.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/leereilly/commit-crawl/internal/contribgraph"
	"github.com/leereilly/commit-crawl/internal/game"
	"github.com/leereilly/commit-crawl/internal/rng"
	"github.com/leereilly/commit-crawl/internal/ui"
	"github.com/leereilly/commit-crawl/internal/ui/palette"
)

func main() {
	seed := flag.Int64("seed", 0, "RNG seed (0 = time-based)")
	noColor := flag.Bool("no-color", false, "disable colors for monochrome terminals")
	user := flag.String("user", "", "GitHub handle: render its Build-themed contribution graph (SVG + GIF) and exit (skips the game)")
	flag.Parse()

	palette.NoColor = *noColor

	if handle := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(*user), "@")); handle != "" {
		if err := generateContribGraph(handle); err != nil {
			fmt.Fprintf(os.Stderr, "commit-crawl: %v\n", err)
			os.Exit(1)
		}
		return
	}

	screen, err := tcell.NewScreen()
	if err != nil {
		fmt.Fprintf(os.Stderr, "commit-crawl: cannot create screen: %v\n", err)
		os.Exit(1)
	}
	if err := screen.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "commit-crawl: cannot init screen: %v\n", err)
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
			switch {
			case g.Phase == game.PhaseTitle, g.Phase == game.PhaseRickRoll, g.Phase == game.PhaseUsername:
				render(screen, g, tipIdx)
			case g.Phase == game.PhaseIntro:
				// PhaseIntro is fully tick-driven: each pulse advances the
				// bloom + slide animation, and AdvanceIntro flips the
				// phase to PhasePlaying once the timeline finishes.
				g.AdvanceIntro()
				render(screen, g, tipIdx)
			case g.Phase == game.PhaseEndSequence:
				// PhaseEndSequence is fully tick-driven: each pulse
				// advances the typed-prompt/spinner animation and may
				// auto-transition to the rick roll when the timeline
				// finishes.
				g.AdvanceEndSequence()
				render(screen, g, tipIdx)
			case g.Phase == game.PhasePlaying && g.Tick <= g.CopilotBlinkUntil:
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
		// title screen. Any non-matching key falls through and routes to
		// the username entry screen (so "press any key to begin" still
		// works for non-arrow keys). Once the full sequence is entered,
		// KonamiArmed is latched for the rest of the session and
		// StartRun grants invincibility.
		if !g.KonamiArmed && game.KonamiMatchesSlot(g.KonamiProgress, ev) {
			g.KonamiProgress++
			if g.KonamiProgress >= game.KonamiLen {
				g.KonamiArmed = true
			}
			g.ResetTitleDigit()
			return false
		}
		g.KonamiProgress = 0
		// Triple-tap a digit '1'..'5' on the title screen to warp the
		// freshly-spawned player straight to that BUILD floor. The digit
		// presses are silently consumed (the player stays on the title
		// screen) until a third matching press fires the warp. Any
		// non-digit key resets the counter and falls through to the normal
		// "any key starts username entry" path below, so the title hint
		// still works for SPACE / ENTER / letters.
		if consumed, depth := g.TitleDigitTap(ev.Rune()); consumed {
			if depth > 0 {
				g.StartRunAtDepth(depth)
			}
			return false
		}
		g.Phase = game.PhaseUsername
	case game.PhaseUsername:
		// Esc/Ctrl-C exits before the run even starts.
		if ev.Key() == tcell.KeyEscape || ev.Key() == tcell.KeyCtrlC {
			return true
		}
		// Backspace deletes the last typed character of the handle. We do
		// NOT let the user delete the implicit '@' — it's a fixed addon.
		if ev.Key() == tcell.KeyBackspace || ev.Key() == tcell.KeyBackspace2 {
			g.BackspaceUsername()
			return false
		}
		// Enter submits and starts the run, but only once we have a
		// valid handle. Silently no-op otherwise. Instead of jumping
		// straight to PhasePlaying, route through PhaseIntro so the
		// username screen melts away into the first floor with the
		// avatar gliding into the spawn cell.
		if ev.Key() == tcell.KeyEnter {
			if g.UsernameReady() {
				w, h := g.Screen.Size()
				sx, sy := ui.UsernameAtPos(w, h)
				g.BeginIntro(sx, sy)
			}
			return false
		}
		// Any other printable rune is fed through the validator.
		if r := ev.Rune(); r != 0 {
			g.AppendUsernameRune(r)
		}
	case game.PhaseIntro:
		// Any key (other than quit) skips the bloom animation and drops
		// the player straight into PhasePlaying. The intro's auto-advance
		// in the tick loop will get them there in ~2.6s if they wait.
		if a == game.ActQuit {
			return true
		}
		g.SkipIntro()
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
	case game.PhaseEndSequence:
		// The end sequence is auto-advancing and is mostly hands-off, but
		// the player can still bail out with q/Ctrl-C/Esc rather than be
		// held hostage by the typewriter pacing.
		if a == game.ActQuit {
			if g.EndSeq != nil {
				g.EndSeq.Cancel()
				g.EndSeq = nil
			}
			return true
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
	case game.PhaseUsername:
		ui.RenderUsername(screen, g.Username, g.Tick, g.UsernameReady())
	case game.PhaseIntro:
		ui.RenderIntro(screen, g.Intro, g.Player, g.Floor, g.Tick)
	case game.PhaseTitle:
		tip := ui.Tips[tipIdx%len(ui.Tips)]
		ui.RenderTitle(screen, tip, g.Tick, g.KonamiArmed)
	case game.PhasePlaying:
		ui.RenderGame(screen, g.Player, g.Floor, g.Log, g.Username)
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
	case game.PhaseEndSequence:
		ui.RenderEndSequence(screen, g.EndSeq, g.Tick)
	case game.PhaseRickRoll:
		ui.RenderRickRoll(screen, g.Tick)
	case game.PhaseWon:
		ui.RenderWon(screen, g.Player.MaxHP, g.Player.Vaulted)
	}
	screen.Show()
}

// generateContribGraph fetches handle's GitHub contribution data and writes
// the Build-themed SVG (plus a matching animated GIF) to the current
// working directory. It runs without ever touching tcell so the caller can
// use --user from a non-interactive shell (CI, docs builds, etc.).
func generateContribGraph(handle string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out := contribgraph.DefaultOutputPath
	if _, err := contribgraph.Generate(ctx, nil, handle, out); err != nil {
		return err
	}
	gifOut := strings.TrimSuffix(out, ".svg") + ".gif"
	fmt.Fprintf(os.Stdout, "Wrote @%s's Build-themed contribution graph to %s and %s\n", handle, out, gifOut)
	return nil
}
