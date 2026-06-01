package ui

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/leereilly/commit-crawl/internal/contribgraph"
	"github.com/leereilly/commit-crawl/internal/ui/palette"
)

// EndStatus is the outcome of the post-level-5 network probes. The probes run
// in a background goroutine the moment the sequence begins, so by the time
// the user has watched the "You made it!" flash and the typed-out shell
// commands, the result is almost always already in.
type EndStatus int32

const (
	// EndStatusPending means the background probe is still running.
	EndStatusPending EndStatus = iota
	// EndStatusOK means we reached the internet and the GitHub handle (if
	// any) resolves to a real account.
	EndStatusOK
	// EndStatusOffline means we couldn't reach github.com at all.
	EndStatusOffline
	// EndStatusNoUser means we reached github.com but @username is 404.
	EndStatusNoUser
)

// Scene-time constants (one tick = 100ms in the main loop). All values are
// measured from the start of the end sequence.
const (
	endFlashTicks      = 30 // length of the "You made it!" flash
	endFlashBlink      = 6  // visible-on / hidden-off period during the flash
	endPostFlash       = 6  // brief pause before the prompt fades in
	endTypeRate        = 2  // ticks per typed character (~20 cps)
	endPostCdPause     = 8  // breath between typing cd and typing build
	endPostBuildPause  = 6  // breath between typing build and showing output
	endSpinnerTicks    = 22 // duration each spinner step "thinks" before ✓
	endPostOutputPause = 6  // breath between spinner ✓ and the git status prompt
	endPostGitPause    = 12 // breath after `git status` is typed before rickroll
	endFailPerLine     = 14 // ticks before the next "we can't continue" line appears
	endFailHold        = 28 // ticks the failure message lingers before rickroll
	endStatusTimeout   = 35 // tolerated wait (in scene ticks) for the probe
)

// endPromptPrefix is the shell prompt printed before each typed command.
// On macOS/Linux it's the familiar POSIX "$ "; on Windows we mimic a
// PowerShell prompt instead so the finale reads as native to whichever OS
// the player is actually on. The PowerShell form includes the canonical
// "PS <cwd>>" shape (with a placeholder C:\ cwd) so it reads as a real
// pwsh.exe prompt rather than a stylised shorthand. Both variants include
// the trailing space.
var endPromptPrefix = endPromptPrefixFor(runtime.GOOS)

func endPromptPrefixFor(goos string) string {
	if goos == "windows" {
		return `PS C:\> `
	}
	return "$ "
}

// endCdText is the deliberately-Ballmer prompt the player watches the game
// type at the post-level-5 shell. It dovetails with the rickroll's "developers,
// developers, developers..." subtitle. On Windows we swap to backslash path
// separators so the line looks at home in PowerShell.
var endCdText = endCdTextFor(runtime.GOOS)

func endCdTextFor(goos string) string {
	if goos == "windows" {
		return `cd developers\developers\developers`
	}
	return "cd developers/developers/developers"
}

// endCdSchedule[i] is the scene-relative tick at which character i of
// endCdText becomes visible. Generated once with a fixed seed so the typing
// cadence is jittery (bursts of fast keys, the occasional fingering pause)
// rather than a flat metronome, while still being deterministic across
// runs and replayable in tests. Average ~1.05 ticks/char — noticeably
// snappier than the 2 ticks/char used elsewhere.
var endCdSchedule = makeJitterSchedule(len(endCdText), 0xC0DECAB)

// makeJitterSchedule produces a cumulative per-character delay schedule that
// mimics human burst typing. Each character draws a delay from a weighted
// distribution: mostly fast (0–1 ticks), occasionally a thinking pause
// (2–3 ticks). Seeded so the cadence is repeatable.
func makeJitterSchedule(n int, seed int64) []int {
	sched := make([]int, n)
	r := rand.New(rand.NewSource(seed))
	t := 0
	for i := 0; i < n; i++ {
		// Weighted delay: 30% zero (chord with previous), 55% one tick,
		// 12% two ticks, 3% three ticks. The "zero" bucket is what makes
		// the line feel like a real typist rather than a CRT printout.
		var delay int
		switch p := r.Intn(100); {
		case p < 30:
			delay = 0
		case p < 85:
			delay = 1
		case p < 97:
			delay = 2
		default:
			delay = 3
		}
		// Don't let the first character land on tick 0 — the prompt
		// needs a frame to settle before the typewriter starts, or it
		// looks like the line was always there.
		if i == 0 && delay == 0 {
			delay = 1
		}
		t += delay
		sched[i] = t
	}
	return sched
}

// endCdTotalTicks is the tick at which the final character of endCdText
// appears (and therefore the line is fully typed).
func endCdTotalTicks() int {
	if len(endCdSchedule) == 0 {
		return 0
	}
	return endCdSchedule[len(endCdSchedule)-1]
}

// endBuildText is the build command typed on the next line. The spinner
// output below it pretends to be `build`'s log output. We deliberately use
// the local-binary invocation form on both platforms — "./build" on POSIX
// and ".\build.exe" on Windows — so the gag reads as "the cd actually
// landed us next to a binary we can run", instead of a bare "build" that a
// real POSIX shell would either fail to resolve or resolve to an unrelated
// tool on $PATH.
var endBuildText = endBuildTextFor(runtime.GOOS)

func endBuildTextFor(goos string) string {
	if goos == "windows" {
		return `.\build.exe`
	}
	return "./build"
}

// endGitText is the cherry on top: the player watches the game type out
// git status before the rick roll kicks in. The command is the same on
// both POSIX and PowerShell, so it doesn't need an OS-conditional form.
const endGitText = "git status"

// endSpinnerMessages is the build-output the spinners walk through when we
// have internet and a valid GitHub user. Order matters — the third line is
// the punchline that primes the rickroll's contribution-graph gag.
var endSpinnerMessages = []string{
	"Analyzing all available data...",
	"Chomping at the bits...",
	"Generating your special Build GitHub contribution graph",
}

// endSpinnerFrames cycles for the in-progress spinner glyph; matches the
// classic CLI braille spinner so it reads as "real" build output.
var endSpinnerFrames = []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}

// EndSequenceState carries the live state of the post-level-5 finale. It
// owns the network probe goroutine and is consumed by RenderEndSequence and
// EndSequenceDone.
type EndSequenceState struct {
	// StartTick is the global game tick at which the sequence began. All
	// scene timing is measured relative to this value.
	StartTick int

	// Username is the GitHub handle entered on the title screen, sans '@'.
	// Empty means "the player never entered one" — we skip the user check
	// in that case.
	Username string

	// GraphPath is where the Build-themed contribution graph SVG will be
	// written when the success branch reaches the spinner stage. The
	// matching animated GIF is written alongside with the .svg extension
	// swapped for .gif. Defaults to contribgraph.DefaultOutputPath (i.e.
	// "contribution-graph.svg" in the current working directory).
	GraphPath string

	// status is the EndStatus value, accessed atomically because the
	// network goroutine writes it from a different goroutine than the
	// renderer reads it.
	status int32

	// cancel stops the in-flight probe if the sequence is torn down early
	// (e.g. the player rage-quits before the result lands).
	cancel context.CancelFunc

	// graphOnce guarantees the contribution-graph fetch fires exactly
	// once per end sequence, no matter how many render or advance ticks
	// re-enter MaybeStartContribGraph.
	graphOnce sync.Once
}

// endStatusClient is overridable by tests so we can pin the probe to a
// deterministic outcome without doing real HTTP. Production callers leave
// it nil and we fall through to the default net/http implementation.
var endStatusClient func(ctx context.Context, username string) EndStatus

// endContribGenerator is the contribution-graph generator hook. Tests
// override this to avoid the network and to assert on the call. Production
// callers leave it nil and we fall through to contribgraph.Generate, which
// fetches https://github.com/<user>.contribs and writes both the SVG and
// the matching animated GIF.
var endContribGenerator func(ctx context.Context, username, outPath string) error

// MaybeStartContribGraph kicks off the one-shot Build-themed contribution
// graph fetch the first time the post-build spinners begin to roll on the
// success branch. The render side calls this every frame; the sync.Once
// guarantees the actual fetch + file write happens at most once per end
// sequence. Returns true on the call that actually scheduled the goroutine.
//
// Preconditions for firing (matching the spec — "once the spinners start,
// and the username is valid and there's internet"):
//   - A username was entered (an empty handle means there is no graph to
//     render).
//   - The probe has settled as EndStatusOK (i.e. github.com is reachable
//     and @username resolves to a real account).
//   - The scene has advanced past the decision point, which is the same
//     tick the first spinner line starts spinning in drawEndSuccess.
func (es *EndSequenceState) MaybeStartContribGraph(tick int) bool {
	if es == nil || es.Username == "" {
		return false
	}
	if es.Status() != EndStatusOK {
		return false
	}
	scene := tick - es.StartTick
	decisionScene, ok := endDecisionScene(scene)
	if !ok || scene < decisionScene {
		return false
	}
	started := false
	es.graphOnce.Do(func() {
		started = true
		out := es.GraphPath
		if out == "" {
			out = contribgraph.DefaultOutputPath
		}
		go runContribGraph(es.Username, out)
	})
	return started
}

// runContribGraph is the goroutine body for the one-shot generator. It
// uses a generous context timeout so the fetch can complete even on a
// slow link without ever blocking the main render loop. Errors are
// intentionally swallowed: the player has already been told via the
// probe that the internet is up, so failing the SVG write here doesn't
// need a second on-screen surface.
func runContribGraph(username, outPath string) {
	fn := endContribGenerator
	if fn == nil {
		fn = func(ctx context.Context, username, outPath string) error {
			_, err := contribgraph.Generate(ctx, nil, username, outPath)
			return err
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = fn(ctx, username, outPath)
}

// NewEndSequence allocates the state and kicks off the network probe in a
// background goroutine. The probe is best-effort: any failure (DNS, TLS,
// timeout, weird status code) is mapped to EndStatusOffline so the player
// always sees a clean fail path rather than a stuck spinner.
func NewEndSequence(startTick int, username string) *EndSequenceState {
	ctx, cancel := context.WithCancel(context.Background())
	es := &EndSequenceState{
		StartTick: startTick,
		Username:  username,
		GraphPath: contribgraph.DefaultOutputPath,
		cancel:    cancel,
	}
	atomic.StoreInt32(&es.status, int32(EndStatusPending))
	go es.runProbe(ctx)
	return es
}

// Status returns the current probe result, safe to call from the renderer.
func (es *EndSequenceState) Status() EndStatus {
	return EndStatus(atomic.LoadInt32(&es.status))
}

func (es *EndSequenceState) setStatus(s EndStatus) {
	atomic.StoreInt32(&es.status, int32(s))
}

// Cancel stops the background probe. Safe to call multiple times.
func (es *EndSequenceState) Cancel() {
	if es.cancel != nil {
		es.cancel()
	}
}

func (es *EndSequenceState) runProbe(ctx context.Context) {
	if endStatusClient != nil {
		es.setStatus(endStatusClient(ctx, es.Username))
		return
	}
	es.setStatus(defaultEndProbe(ctx, es.Username))
}

// defaultEndProbe is the real-world network check: a HEAD to api.github.com
// to confirm connectivity, then (if a username was entered) a GET to the
// users endpoint to confirm the handle exists.
func defaultEndProbe(ctx context.Context, username string) EndStatus {
	client := &http.Client{Timeout: 3 * time.Second}
	connectReq, err := http.NewRequestWithContext(ctx, http.MethodHead, "https://api.github.com/", nil)
	if err != nil {
		return EndStatusOffline
	}
	connectResp, err := client.Do(connectReq)
	if err != nil {
		return EndStatusOffline
	}
	connectResp.Body.Close()

	if username == "" {
		return EndStatusOK
	}

	userReq, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://api.github.com/users/"+username, nil)
	if err != nil {
		return EndStatusOK
	}
	userResp, err := client.Do(userReq)
	if err != nil {
		return EndStatusOffline
	}
	defer userResp.Body.Close()
	switch userResp.StatusCode {
	case http.StatusOK:
		return EndStatusOK
	case http.StatusNotFound:
		return EndStatusNoUser
	default:
		// Rate-limited, 5xx, etc. — be generous and pretend the user is
		// valid rather than punishing the player with the no-account
		// rickroll on the basis of an ambiguous server response.
		return EndStatusOK
	}
}

// EndSequenceDone reports whether the sequence has finished playing and the
// caller should transition to the rick roll. The main loop polls this each
// tick so the transition stays driven by the central ticker.
func EndSequenceDone(es *EndSequenceState, tick int) bool {
	if es == nil {
		return true
	}
	scene := tick - es.StartTick
	if scene < 0 {
		return false
	}

	// We've finished the typed "build" prompt and reached the decision
	// point. Anything before that means the sequence is still mid-flight.
	decisionScene, ok := endDecisionScene(scene)
	if !ok {
		return false
	}

	status := effectiveStatus(es.Status(), scene-decisionScene)
	switch status {
	case EndStatusPending:
		return false
	case EndStatusOK:
		return scene >= endSuccessTotalTicks(decisionScene)
	default:
		return scene-decisionScene >= endFailureTotalTicks()
	}
}

// endDecisionScene returns the scene tick at which the typed `build` line is
// fully drawn and we start branching on the probe status. The bool is false
// while the player is still watching the flash / cd / build animations,
// which lets the caller bail out of "done?" checks cheaply.
func endDecisionScene(scene int) (int, bool) {
	cdStart := endFlashTicks + endPostFlash
	cdDone := cdStart + endCdTotalTicks()
	buildStart := cdDone + endPostCdPause
	buildDone := buildStart + len(endBuildText)*endTypeRate
	decisionScene := buildDone + endPostBuildPause
	if scene < decisionScene {
		return 0, false
	}
	return decisionScene, true
}

// effectiveStatus folds the probe-timeout grace period into a single status
// value. If the probe is still pending after endStatusTimeout ticks of
// decision-scene wall time, we give up and treat the player as offline so
// the show goes on.
func effectiveStatus(s EndStatus, decisionRel int) EndStatus {
	if s == EndStatusPending && decisionRel >= endStatusTimeout {
		return EndStatusOffline
	}
	return s
}

func endSuccessTotalTicks(decisionScene int) int {
	spinners := len(endSpinnerMessages) * endSpinnerTicks
	gitStart := spinners + endPostOutputPause
	gitDone := gitStart + len(endGitText)*endTypeRate
	return decisionScene + gitDone + endPostGitPause
}

func endFailureTotalTicks() int {
	// Two lines stagger in, then the hold period before the rickroll.
	return endFailPerLine*2 + endFailHold
}

// RenderEndSequence draws one frame of the post-level-5 finale. All animation
// is a pure function of the scene tick (tick - StartTick) plus the latched
// probe status, so the main loop can simply call this on every ticker pulse.
func RenderEndSequence(s tcell.Screen, es *EndSequenceState, tick int) {
	Clear(s)
	w, h := s.Size()
	if es == nil {
		return
	}
	scene := tick - es.StartTick

	// --- Phase A: "You made it!" flash ---
	if scene < endFlashTicks {
		if (scene/endFlashBlink)%2 == 0 {
			msg := "You made it!"
			c := palette.Cycle[(tick/2)%len(palette.Cycle)]
			DrawString(s, (w-runeLen(msg))/2, h/2, msg, palette.FG(c).Bold(true))
		}
		return
	}

	// --- Phase B onwards: terminal-style output ---
	postFlash := scene - endFlashTicks
	// A brief blank pause so the flash has time to dissipate before the
	// prompt fades in.
	if postFlash < endPostFlash {
		return
	}

	// Anchor the fake terminal near the upper-third of the screen so the
	// spinner block can grow downward without colliding with the bottom
	// edge on standard 24-line terminals.
	const leftMargin = 4
	topY := h/3 - 1
	if topY < 1 {
		topY = 1
	}
	x := leftMargin
	promptW := runeLen(endPromptPrefix)
	if x+promptW+runeLen(endCdText)+2 > w {
		x = max0(w - promptW - runeLen(endCdText) - 2)
	}

	cdStart := endFlashTicks + endPostFlash
	drawTypedPrompt(s, x, topY, endCdText, scene-cdStart, tick, true, endCdSchedule)

	cdDone := cdStart + endCdTotalTicks()
	if scene < cdDone {
		return
	}
	if scene < cdDone+endPostCdPause {
		return
	}

	buildStart := cdDone + endPostCdPause
	buildRow := topY + 1
	drawTypedPrompt(s, x, buildRow, endBuildText, scene-buildStart, tick, true, nil)

	buildDone := buildStart + len(endBuildText)*endTypeRate
	if scene < buildDone {
		return
	}
	if scene < buildDone+endPostBuildPause {
		return
	}

	decisionScene := buildDone + endPostBuildPause
	decisionRel := scene - decisionScene
	status := effectiveStatus(es.Status(), decisionRel)

	outputRow := buildRow + 1

	switch status {
	case EndStatusPending:
		// Quiet ellipsis while we wait for the probe to land. Capped at
		// three dots so the line doesn't grow unbounded.
		dots := strings.Repeat(".", 1+(decisionRel/3)%3)
		DrawString(s, x, outputRow, dots, palette.FG(palette.DimGray))
	case EndStatusOK:
		drawEndSuccess(s, x, outputRow, decisionRel, tick)
	case EndStatusOffline:
		drawEndFailure(s, x, outputRow, decisionRel, []string{
			"No Internet connection...",
			"Gonna have to give you up...",
		})
	case EndStatusNoUser:
		account := fmt.Sprintf("No account found for @%s", es.Username)
		drawEndFailure(s, x, outputRow, decisionRel, []string{
			account,
			"Gonna have to give you up...",
		})
	}
}

// drawTypedPrompt renders a single "$ <text>" line with a typewriter reveal.
// rel is the number of scene ticks since the typewriter started for this line;
// negative values draw nothing. When schedule is non-nil it gives the per-
// character cumulative reveal ticks (for jittery, human-paced typing); when
// nil the function falls back to the uniform endTypeRate. While the line is
// still being typed we also blink a caret at the write head.
func drawTypedPrompt(s tcell.Screen, x, y int, text string, rel, tick int, blinkCaret bool, schedule []int) {
	if rel < 0 {
		return
	}
	DrawString(s, x, y, endPromptPrefix, palette.FG(palette.Green).Bold(true))
	promptW := runeLen(endPromptPrefix)
	var shown int
	if schedule != nil {
		// sort.SearchInts(schedule, rel+1) returns the smallest i with
		// schedule[i] > rel, which equals the count of characters whose
		// cumulative delay has already elapsed at scene-rel `rel`.
		shown = sort.SearchInts(schedule, rel+1)
	} else {
		shown = rel / endTypeRate
	}
	if shown > len(text) {
		shown = len(text)
	}
	DrawString(s, x+promptW, y, text[:shown], palette.FG(palette.White).Bold(true))
	if blinkCaret && shown < len(text) && (tick/3)%2 == 0 {
		DrawRune(s, x+promptW+shown, y, '▌', palette.FG(palette.Yellow))
	}
}

// drawEndSuccess renders the three-spinner build output followed by the
// typed `git status` prompt. Each spinner shows the braille spinner glyph
// while in progress and a green check + dimmed message once it settles.
func drawEndSuccess(s tcell.Screen, x, y, decisionRel, tick int) {
	for i, msg := range endSpinnerMessages {
		lineStart := i * endSpinnerTicks
		if decisionRel < lineStart {
			return
		}
		done := decisionRel >= lineStart+endSpinnerTicks
		row := y + i
		if done {
			DrawRune(s, x, row, '✓', palette.FG(palette.Green).Bold(true))
			DrawString(s, x+2, row, msg, palette.FG(palette.DimGray))
			continue
		}
		// In-progress: cycle the braille spinner and tint the message
		// white so the player's eye lands on the active line.
		spin := endSpinnerFrames[(tick/2)%len(endSpinnerFrames)]
		DrawRune(s, x, row, spin, palette.FG(palette.CopilotCyan).Bold(true))
		DrawString(s, x+2, row, msg, palette.FG(palette.White))
		return
	}
	spinnersDone := len(endSpinnerMessages) * endSpinnerTicks
	if decisionRel < spinnersDone+endPostOutputPause {
		return
	}
	gitRow := y + len(endSpinnerMessages) + 1
	drawTypedPrompt(s, x, gitRow, endGitText,
		decisionRel-spinnersDone-endPostOutputPause, tick, true, nil)
}

// drawEndFailure renders the no-internet / no-account branch: two staggered
// red lines that hold long enough for the player to read them before the
// rickroll takes over. lines[0] is the contextual diagnostic; lines[1] is
// always the Astley callback.
func drawEndFailure(s tcell.Screen, x, y, decisionRel int, lines []string) {
	for i, line := range lines {
		appearAt := i * endFailPerLine
		if decisionRel < appearAt {
			return
		}
		st := palette.FG(palette.Red).Bold(true)
		if i == len(lines)-1 {
			// The "Gonna have to give you up..." callback shimmers
			// through the rainbow so it visibly previews the rickroll
			// that's about to land.
			st = palette.FG(palette.Magenta).Bold(true)
		}
		DrawString(s, x, y+i, line, st)
	}
}
