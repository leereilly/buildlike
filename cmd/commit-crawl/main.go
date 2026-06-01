// Command commit-crawl is a charming Go roguelike themed on Microsoft Build.
// You play @, you squash bugs (b), you eat green +s, you climb five letter-
// shaped dungeons spelling BUILD, and the credits roll with a... surprise.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/leereilly/commit-crawl/internal/contribgraph"
	"github.com/leereilly/commit-crawl/internal/game"
	"github.com/leereilly/commit-crawl/internal/rng"
	"github.com/leereilly/commit-crawl/internal/ui"
	"github.com/leereilly/commit-crawl/internal/ui/palette"
)

// version is the module version printed by --version. It's overwritten at
// build time by `go build -ldflags "-X main.version=…"` and, failing that,
// derived from runtime/debug.ReadBuildInfo so `go install` builds still
// show something sensible.
var version = ""

const usageHeader = `commit-crawl — a tiny terminal roguelike themed on Microsoft Build.

Usage:
  gh commit-crawl [flags]
  gh commit-crawl --user HANDLE [--output PATH] [--format svg,gif]
  gh commit-crawl --copilot-plays [--as HANDLE] [--seed N]
  gh commit-crawl --user HANDLE --pr
  gh commit-crawl completion {bash|zsh|fish|powershell}
  gh commit-crawl --version

Examples:
  gh commit-crawl                       # play the game interactively
  gh commit-crawl --seed 42             # deterministic run (reproducible)
  gh commit-crawl --copilot-plays --as octocat
                                        # let Copilot drive a whole run
  gh commit-crawl --user octocat        # render octocat's Build-themed graph
                                        # to ./contribution-graph.{svg,gif}
  gh commit-crawl --user octocat --format svg --output me.svg
  gh commit-crawl --user octocat --pr   # print the gh commands to PR the
                                        # graph onto your profile README

Keybindings (in-game):
  arrows / hjkl / wasd / numpad   move
  y u b n                         diagonal moves (vi-keys)
  .  / space / 5                  wait one turn
  >                               ascend stairs
  ?                               help
  q / Esc / Ctrl-C                quit

Flags:`

const usageFooter = `
Easter eggs:
  ↑↑↓↓←→←→BA on the title screen          → invincibility
  triple-tap 1..5 on the title screen     → warp to that BUILD floor
  Clippy on floor 2, jester (j) on a random floor, Copilot blinks on every
  keypress, and there's a surprise after you ascend floor 5.

See https://github.com/leereilly/gh-commit-crawl for full docs.
`

func main() {
	// Subcommand: completion. Dispatched before flag parsing because it
	// has its own positional arg form (no leading dash) and shouldn't be
	// misread by the flag package.
	if len(os.Args) > 1 && os.Args[1] == "completion" {
		os.Exit(runCompletion(os.Args[2:]))
	}

	seed := flag.Int64("seed", 0, "RNG seed (0 = time-based; pin a value for reproducible runs and recordings)")
	noColor := flag.Bool("no-color", false, "disable colors for monochrome terminals")
	user := flag.String("user", "", "GitHub handle: render its Build-themed contribution graph and exit (skips the game)")
	output := flag.String("output", contribgraph.DefaultOutputPath, "output path for --user (e.g. me.svg). The matching .gif lands next to it.")
	format := flag.String("format", "svg,gif", "comma list of artifacts to write with --user: svg,gif")
	copilotPlays := flag.Bool("copilot-plays", false, "let Copilot drive the whole run (autopilot mode) — pair with --seed for reproducible demos")
	as := flag.String("as", "", "GitHub handle to pre-fill on the username screen (used by --copilot-plays so a demo is unattended)")
	pr := flag.Bool("pr", false, "with --user: print the gh CLI commands you'd run to PR the generated graph onto your profile README, then exit")
	showVersion := flag.Bool("version", false, "print the commit-crawl version and exit")

	flag.Usage = func() { printUsage(os.Stderr, flag.CommandLine) }
	flag.Parse()

	if *showVersion {
		fmt.Println(resolveVersion())
		return
	}

	palette.NoColor = *noColor

	if handle := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(*user), "@")); handle != "" {
		if *pr {
			if err := printPRCommands(handle, *output); err != nil {
				fail(err)
			}
			return
		}
		if err := generateContribGraph(handle, *output, *format); err != nil {
			fail(err)
		}
		return
	}
	if *pr {
		fail(fmt.Errorf("--pr requires --user HANDLE"))
	}

	screen, err := tcell.NewScreen()
	if err != nil {
		fail(fmt.Errorf("cannot create screen: %w", err))
	}
	if err := screen.Init(); err != nil {
		fail(fmt.Errorf("cannot init screen: %w", err))
	}
	defer screen.Fini()
	screen.SetStyle(palette.Style(palette.White, palette.Black))
	screen.HideCursor()
	screen.Clear()

	r := rng.New(*seed)
	g := game.New(screen, r)

	auto := newAutopilot(*copilotPlays, *as)
	run(screen, g, auto)
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "commit-crawl: %v\n", err)
	os.Exit(1)
}

func printUsage(w *os.File, fs *flag.FlagSet) {
	fmt.Fprint(w, usageHeader+"\n")
	fs.SetOutput(w)
	fs.PrintDefaults()
	fmt.Fprint(w, usageFooter)
}

// resolveVersion returns the package-level `version` if it was set by
// ldflags, otherwise the module version from runtime/debug.ReadBuildInfo.
// Falls back to "dev" for unstamped local builds (e.g. `go run .`).
func resolveVersion() string {
	if version != "" {
		return version
	}
	if bi, ok := debug.ReadBuildInfo(); ok {
		if v := bi.Main.Version; v != "" && v != "(devel)" {
			return v
		}
	}
	return "dev"
}

// autopilot drives every phase of the game when --copilot-plays is set.
// It exists so the main loop can ignore the keyboard entirely (no human in
// the loop) while still threading through the same Game state machine that
// a real player would.
type autopilot struct {
	enabled bool
	handle  string // pre-filled GitHub handle for the username screen
	typed   int    // chars of `handle` typed onto the username screen so far
}

func newAutopilot(enabled bool, handle string) *autopilot {
	h := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(handle), "@"))
	if enabled && h == "" {
		// Sensible default so `--copilot-plays` is one flag, not two.
		h = "copilot"
	}
	return &autopilot{enabled: enabled, handle: h}
}

// step is called every autopilot ticker pulse. It mutates g in-place by
// synthesizing the next sensible user input for the current phase. Returns
// true when the program should quit (autopilot has nothing else to do).
func (a *autopilot) step(g *game.Game) bool {
	if !a.enabled || g == nil {
		return false
	}
	switch g.Phase {
	case game.PhaseTitle:
		g.Phase = game.PhaseUsername
	case game.PhaseUsername:
		if a.typed < len(a.handle) {
			g.AppendUsernameRune(rune(a.handle[a.typed]))
			a.typed++
			return false
		}
		if !g.UsernameReady() {
			return false
		}
		w, h := g.Screen.Size()
		sx, sy := ui.UsernameAtPos(w, h)
		g.BeginIntro(sx, sy)
	case game.PhaseIntro:
		g.SkipIntro()
	case game.PhasePlaying:
		g.Step(game.AutopilotChoice(g))
	case game.PhaseDead:
		// Autopilot took a loss; bow out rather than infinitely retry.
		return true
	case game.PhaseEndSequence:
		// EndSequence advances itself; nothing to synthesize.
	case game.PhaseRickRoll:
		g.Phase = game.PhaseWon
	case game.PhaseWon:
		return true
	}
	return false
}

func run(screen tcell.Screen, g *game.Game, auto *autopilot) {
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

	// Autopilot fires on its own slower cadence so the on-screen action is
	// readable. Disabled cleanly when --copilot-plays is off by leaving
	// the channel nil.
	var autoTickC <-chan time.Time
	if auto != nil && auto.enabled {
		autoTicker := time.NewTicker(150 * time.Millisecond)
		defer autoTicker.Stop()
		autoTickC = autoTicker.C
	}

	render(screen, g, tipIdx)

	for {
		select {
		case ev := <-events:
			switch ev := ev.(type) {
			case *tcell.EventResize:
				screen.Sync()
			case *tcell.EventKey:
				// In autopilot mode, the keyboard is reserved as a manual
				// kill-switch — q/Esc/Ctrl-C always quits, everything else
				// is ignored so the human can't accidentally fight the bot.
				if auto != nil && auto.enabled {
					if game.MapKey(ev) == game.ActQuit {
						return
					}
					break
				}
				if handleKey(g, ev) {
					return
				}
			}
			render(screen, g, tipIdx)
		case <-autoTickC:
			if auto.step(g) {
				return
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
// the Build-themed artifacts requested by `format` to `outPath` (SVG) and
// the matching `.gif` (when GIF output is selected). The renderer never
// touches tcell so the caller can use --user from a non-interactive shell
// (CI, docs builds, etc.).
func generateContribGraph(handle, outPath, format string) error {
	wantSVG, wantGIF, err := parseFormats(format)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	data, err := contribgraph.Fetch(ctx, nil, handle)
	if err != nil {
		return err
	}
	if outPath == "" {
		outPath = contribgraph.DefaultOutputPath
	}
	var wrote []string
	if wantSVG {
		svg := contribgraph.Render(data, handle, nil)
		if err := os.WriteFile(outPath, svg, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", outPath, err)
		}
		wrote = append(wrote, outPath)
	}
	if wantGIF {
		gifBytes, err := contribgraph.RenderGIF(data, handle, nil)
		if err != nil {
			return fmt.Errorf("render gif: %w", err)
		}
		gifPath := gifPathFor(outPath)
		if err := os.WriteFile(gifPath, gifBytes, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", gifPath, err)
		}
		wrote = append(wrote, gifPath)
	}
	fmt.Fprintf(os.Stdout, "Wrote @%s's Build-themed contribution graph to %s\n", handle, strings.Join(wrote, " and "))
	return nil
}

// parseFormats accepts a comma-separated list of "svg" and "gif" tokens.
// Empty or whitespace tokens are ignored; unknown tokens are an error.
func parseFormats(s string) (svg, gif bool, err error) {
	if strings.TrimSpace(s) == "" {
		return true, true, nil
	}
	for _, raw := range strings.Split(s, ",") {
		tok := strings.ToLower(strings.TrimSpace(raw))
		switch tok {
		case "":
			// permissive
		case "svg":
			svg = true
		case "gif":
			gif = true
		default:
			return false, false, fmt.Errorf("unknown --format value %q (want svg, gif, or both)", tok)
		}
	}
	if !svg && !gif {
		return false, false, fmt.Errorf("--format must select at least one of svg, gif")
	}
	return svg, gif, nil
}

// gifPathFor derives the GIF output path that pairs with an SVG output
// path. A `.svg` (case-insensitive) suffix is swapped for `.gif`; anything
// else gets `.gif` appended. Mirrors the helper in internal/contribgraph
// so the standalone main can render only the GIF when asked.
func gifPathFor(svgPath string) string {
	if len(svgPath) >= 4 && strings.EqualFold(svgPath[len(svgPath)-4:], ".svg") {
		return svgPath[:len(svgPath)-4] + ".gif"
	}
	return svgPath + ".gif"
}

// printPRCommands emits the exact shell+gh commands a user would run to
// open a pull request adding (or refreshing) the Build-themed contribution
// graph on their GitHub profile README — without ever executing them. The
// script is intentionally idempotent: it uses a marked block in README.md
// (`<!-- commit-crawl:start --> … <!-- commit-crawl:end -->`) so re-running
// it cleanly replaces the previous insert.
func printPRCommands(handle, outPath string) error {
	if strings.TrimSpace(handle) == "" {
		return fmt.Errorf("--pr requires --user HANDLE")
	}
	if outPath == "" {
		outPath = contribgraph.DefaultOutputPath
	}
	gifPath := gifPathFor(outPath)
	fmt.Printf(`# Open a PR adding @%[1]s's Build-themed contribution graph to their
# profile README. Review and run these one at a time — nothing was executed.

set -euo pipefail
gh commit-crawl --user %[1]s --output %[2]s

repo=%[1]s/%[1]s
tmp=$(mktemp -d)
gh repo clone "$repo" "$tmp" -- --depth=1
cp %[2]s "$tmp/commit-crawl.svg"
cp %[3]s "$tmp/commit-crawl.gif"

# Drop a marked block into README.md (idempotent — re-running replaces it).
cd "$tmp"
awk 'BEGIN{skip=0}
  /<!-- commit-crawl:start -->/{skip=1; next}
  /<!-- commit-crawl:end -->/{skip=0; next}
  !skip {print}' README.md > README.tmp && mv README.tmp README.md
cat >> README.md <<MD

<!-- commit-crawl:start -->
## My Build-themed contribution graph

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="commit-crawl.gif">
  <img alt="Build-themed contribution graph for @%[1]s" src="commit-crawl.svg">
</picture>

Generated by [gh-commit-crawl](https://github.com/leereilly/gh-commit-crawl).
<!-- commit-crawl:end -->
MD

git checkout -b commit-crawl-graph
git add commit-crawl.svg commit-crawl.gif README.md
git commit -m "Add Build-themed contribution graph"
git push -u origin commit-crawl-graph
gh pr create --fill --title "Add Build-themed contribution graph"
`, handle, outPath, gifPath)
	return nil
}

// runCompletion writes a minimal shell completion script for `commit-crawl`
// (matching the top-level flags). Returns the process exit code.
func runCompletion(args []string) int {
	shell := ""
	if len(args) > 0 {
		shell = strings.ToLower(strings.TrimSpace(args[0]))
	}
	const flags = "--seed --no-color --user --output --format --copilot-plays --as --pr --version --help"
	switch shell {
	case "bash":
		fmt.Printf(`# bash completion for commit-crawl. Source it from your ~/.bashrc, e.g.:
#   source <(gh commit-crawl completion bash)
_commit_crawl() {
  local cur prev
  COMPREPLY=()
  cur="${COMP_WORDS[COMP_CWORD]}"
  if [[ ${cur} == -* ]]; then
    COMPREPLY=( $(compgen -W "%s" -- ${cur}) )
    return 0
  fi
  case "${COMP_WORDS[1]:-}" in
    completion) COMPREPLY=( $(compgen -W "bash zsh fish powershell" -- ${cur}) );;
  esac
}
complete -F _commit_crawl commit-crawl gh-commit-crawl
`, flags)
	case "zsh":
		fmt.Printf(`# zsh completion for commit-crawl. Source it from your ~/.zshrc, e.g.:
#   source <(gh commit-crawl completion zsh)
_commit_crawl() {
  local -a flags
  flags=(%s)
  if [[ ${words[CURRENT]} == -* ]]; then
    compadd -- $flags
  fi
}
compdef _commit_crawl commit-crawl gh-commit-crawl
`, flags)
	case "fish":
		fmt.Println(`# fish completion for commit-crawl. Save to ~/.config/fish/completions/commit-crawl.fish:
#   gh commit-crawl completion fish > ~/.config/fish/completions/commit-crawl.fish`)
		for _, f := range strings.Fields(flags) {
			fmt.Printf("complete -c commit-crawl -l %s\n", strings.TrimPrefix(f, "--"))
			fmt.Printf("complete -c gh-commit-crawl -l %s\n", strings.TrimPrefix(f, "--"))
		}
	case "powershell":
		fmt.Printf(`# PowerShell completion for commit-crawl. Add to $PROFILE:
#   gh commit-crawl completion powershell | Out-String | Invoke-Expression
Register-ArgumentCompleter -Native -CommandName commit-crawl,gh-commit-crawl -ScriptBlock {
    param($wordToComplete, $commandAst, $cursorPosition)
    '%s'.Split(' ') | Where-Object { $_ -like "$wordToComplete*" } |
        ForEach-Object { [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterName', $_) }
}
`, flags)
	default:
		fmt.Fprintln(os.Stderr, "usage: gh commit-crawl completion {bash|zsh|fish|powershell}")
		return 2
	}
	return 0
}
