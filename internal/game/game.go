package game

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/leereilly/buildlike/internal/entity"
	"github.com/leereilly/buildlike/internal/rng"
	"github.com/leereilly/buildlike/internal/ui"
	"github.com/leereilly/buildlike/internal/world"
)

type Phase int

const (
	PhaseTitle Phase = iota
	PhaseUsername
	PhasePlaying
	PhaseHelp
	PhaseDead
	PhaseWon
	PhaseEndSequence
	PhaseRickRoll
	PhaseQuit
)

// MaxUsernameLen mirrors GitHub's 39-character cap on usernames.
const MaxUsernameLen = 39

type Game struct {
	Screen tcell.Screen
	RNG    *rng.RNG
	Player *entity.Player
	Floor  *FloorState
	Log    *ui.MessageLog
	Phase  Phase
	Tick   int // global tick counter for animation

	// Username is the GitHub handle the player entered on the opening
	// PhaseUsername screen. The leading '@' is never stored here; it is
	// rendered as a prefix in the UI so the player always knows it's
	// already included.
	Username string

	// Konami code state. KonamiProgress tracks how many keystrokes of the
	// classic ↑↑↓↓←→←→BA sequence have been entered on the title screen so
	// far. Once the full sequence is entered, KonamiArmed flips true and
	// every subsequent StartRun grants the player invincibility.
	KonamiProgress int
	KonamiArmed    bool

	// ClippyPresses counts player keystrokes on the second floor so the
	// Clippy easter-egg overlay can fade out after the initial grace
	// window. It resets to 0 in StartRun and is only incremented while
	// the player is on depth 2.
	ClippyPresses int

	// CopilotBlinkUntil is the global Tick value at and beyond which the
	// Copilot mascot's eyes return to the open state. Each PhasePlaying
	// keypress arms it to Tick+3 so the eyes stay shut for ~300ms.
	CopilotBlinkUntil int

	// JesterDepth is the BUILD floor (1..5) on which the white 'j' jester
	// easter egg appears for this run. Rolled in StartRun. 0 means "no
	// jester yet" (pre-run).
	JesterDepth int

	// Title triple-tap state. Pressing the same digit '1'..'5' three times
	// in a row on the title screen warps the player straight to that BUILD
	// floor. TitleDigit is the digit currently being repeated (0 when no
	// run is in progress); TitleDigitCount is how many consecutive matching
	// presses we've seen. A non-matching press resets both. Mirrors the
	// no-timeout style of the Konami code tracker above.
	TitleDigit      rune
	TitleDigitCount int

	// EndSeq holds the live state of the post-level-5 finale: the typed
	// shell commands, the spinner animation, and the latched result of
	// the background network probe. Non-nil only during PhaseEndSequence.
	EndSeq *ui.EndSequenceState
}

// TitleTripleTapCount is the number of consecutive identical-digit presses
// required on the title screen to trigger the level-skip warp.
const TitleTripleTapCount = 3

func New(s tcell.Screen, r *rng.RNG) *Game {
	return &Game{
		Screen: s,
		RNG:    r,
		Player: entity.NewPlayer(),
		Log:    ui.NewLog(200),
		Phase:  PhaseTitle,
	}
}

// AppendUsernameRune appends a single rune to Username if it is a valid
// GitHub-handle character (alphanumeric or hyphen) and the handle is not at
// the GitHub-imposed length cap. Returns true on accept, false on reject so
// the caller can decide whether to redraw.
func (g *Game) AppendUsernameRune(r rune) bool {
	if len(g.Username) >= MaxUsernameLen {
		return false
	}
	switch {
	case r >= 'a' && r <= 'z':
	case r >= 'A' && r <= 'Z':
	case r >= '0' && r <= '9':
	case r == '-':
		// GitHub disallows a leading hyphen and consecutive hyphens. We
		// silently reject those here so the on-screen value is always a
		// shape GitHub itself would accept.
		if len(g.Username) == 0 {
			return false
		}
		if g.Username[len(g.Username)-1] == '-' {
			return false
		}
	default:
		return false
	}
	g.Username += string(r)
	return true
}

// BackspaceUsername removes the last rune of Username. Returns true if a
// character was removed.
func (g *Game) BackspaceUsername() bool {
	if g.Username == "" {
		return false
	}
	// Strip trailing rune (handles ASCII only, which is all we accept).
	g.Username = g.Username[:len(g.Username)-1]
	return true
}

// UsernameReady reports whether the entered Username can be submitted. It
// must be non-empty and must not end with a hyphen (matching GitHub's rule).
func (g *Game) UsernameReady() bool {
	if g.Username == "" {
		return false
	}
	if g.Username[len(g.Username)-1] == '-' {
		return false
	}
	return true
}

// StartRun (re)initializes the player and generates level 1.
func (g *Game) StartRun() {
	g.StartRunAtDepth(1)
}

// StartRunAtDepth (re)initializes the player and drops them onto the
// requested BUILD floor (1..5). Depth is clamped into range. Used by the
// normal StartRun (depth=1) and by the title-screen triple-tap level-skip
// easter egg. When depth > 1 a log line announces the warp so the player
// always sees a confirmation that they landed where they aimed.
func (g *Game) StartRunAtDepth(depth int) {
	if depth < 1 {
		depth = 1
	}
	if depth > 5 {
		depth = 5
	}
	g.Player = entity.NewPlayer()
	if g.KonamiArmed {
		g.Player.Invincible = true
	}
	g.ClippyPresses = 0
	g.CopilotBlinkUntil = 0
	g.JesterDepth = g.RNG.IntRange(1, 5)
	g.Floor = BuildFloor(depth, g.Player, g.RNG, g.JesterDepth == depth)
	g.Log = ui.NewLog(200)
	g.Log.Push("Welcome to Buildlike. Squash bugs, ship the build.", ui.LogInfo)
	if g.Player.Invincible {
		g.Log.Push("★ Konami code accepted: you are INVINCIBLE. ★", ui.LogSpecial)
	}
	if depth > 1 {
		letters := []byte{'B', 'U', 'I', 'L', 'D'}
		g.Log.Push(fmt.Sprintf("Triple-tap warp engaged — touching down on floor %d (%c).", depth, letters[depth-1]), ui.LogSpecial)
	}
	g.Phase = PhasePlaying
}

// TitleDigitTap feeds one keystroke into the title-screen triple-tap
// detector. If r is a digit '1'..'5', the press is recorded and consumed
// (returns consumed=true). Three consecutive identical digits returns
// jumpTo=that depth so the caller can warp. Any non-matching key (different
// digit or non-digit) resets the counter; a non-digit additionally returns
// consumed=false so the caller can fall through to its normal
// "any key transitions to username entry" behaviour.
func (g *Game) TitleDigitTap(r rune) (consumed bool, jumpTo int) {
	if r < '1' || r > '5' {
		g.TitleDigit = 0
		g.TitleDigitCount = 0
		return false, 0
	}
	if r != g.TitleDigit {
		g.TitleDigit = r
		g.TitleDigitCount = 0
	}
	g.TitleDigitCount++
	if g.TitleDigitCount >= TitleTripleTapCount {
		depth := int(r - '0')
		g.TitleDigit = 0
		g.TitleDigitCount = 0
		return true, depth
	}
	return true, 0
}

// ResetTitleDigit clears the title-screen triple-tap counter. Called when
// the player engages an unrelated title-screen interaction (e.g. advancing
// the Konami code) so partial digit progress can't be carried across.
func (g *Game) ResetTitleDigit() {
	g.TitleDigit = 0
	g.TitleDigitCount = 0
}

// Step applies one player action and resolves the world's response.
func (g *Game) Step(a Action) {
	if g.Phase != PhasePlaying {
		return
	}
	advanced := false
	switch {
	case a == ActWait:
		advanced = true
	case a == ActAscend:
		if g.Player.Pos == g.Floor.Level.Stairs {
			g.ascend()
			return
		}
		g.Log.Push("There are no stairs here.", ui.LogWarn)
	case a == ActHelp:
		g.Phase = PhaseHelp
		return
	default:
		if dx, dy, ok := Delta(a); ok {
			g.tryMove(dx, dy)
			advanced = true
		}
	}
	if advanced {
		g.bugsAct()
	}
	if g.Player.HP <= 0 {
		g.Phase = PhaseDead
	}
	g.Tick++
}

func (g *Game) tryMove(dx, dy int) {
	dest := world.Point{X: g.Player.Pos.X + dx, Y: g.Player.Pos.Y + dy}
	// Secret door reveal: bumping a secret door turns it into floor.
	if g.Floor.Level.In(dest) && g.Floor.Level.At(dest) == world.TileSecretDoor {
		g.Floor.Level.Set(dest, world.TileFloor)
		g.Log.Push("You discover a hidden passage. The walls hum faintly.", ui.LogSpecial)
		return
	}
	// Bug at dest? Bump-attack.
	for _, b := range g.Floor.Bugs {
		if b.Alive && b.Pos == dest {
			b.Alive = false
			g.Log.Push(ui.PickKillFlavor(g.RNG), ui.LogGood)
			return
		}
	}
	// Jester at dest? Bump-attack (Falconhoof easter egg).
	if j := g.Floor.Jester; j != nil && j.Alive && j.Pos == dest {
		j.Alive = false
		g.Log.Push(`"Kill jester" </falconhoof>`, ui.LogSpecial)
		return
	}
	if !g.Floor.Level.Walkable(dest) {
		return
	}
	g.Player.Pos = dest
	// Pressure plate?
	if g.Floor.Level.HasVault && dest == g.Floor.Level.VaultPlate && !g.Player.Vaulted {
		g.Player.Vaulted = true
		g.Player.MaxHP++
		g.Player.HP++
		g.Log.Push("You found the BUILD vault.", ui.LogSpecial)
		g.Log.Push("The walls hum with the sound of a thousand keyboards.", ui.LogSpecial)
		g.Log.Push("+1 max HP. The build is green.", ui.LogGood)
	}
	// Powerup?
	for _, pu := range g.Floor.Powerups {
		if !pu.Picked && pu.Pos == dest {
			pu.Picked = true
			g.Player.MaxHP++
			g.Player.HP++
			g.Log.Push(ui.PickPowerupFlavor(g.RNG), ui.LogGood)
			return
		}
	}
	// On stairs?
	if dest == g.Floor.Level.Stairs {
		g.Log.Push("Stairs up. Press > to ascend.", ui.LogInfo)
	}
}

func (g *Game) bugsAct() {
	occ := map[world.Point]bool{}
	for _, b := range g.Floor.Bugs {
		if b.Alive {
			occ[b.Pos] = true
		}
	}
	if j := g.Floor.Jester; j != nil && j.Alive {
		occ[j.Pos] = true
	}
	for _, b := range g.Floor.Bugs {
		if dmg := b.Act(g.Floor.Level, g.Player.Pos, occ, g.RNG); dmg > 0 {
			if g.Player.Invincible {
				if g.RNG.Chance(0.15) {
					g.Log.Push("You shrug off the bite. (INVINCIBLE)", ui.LogSpecial)
				}
				continue
			}
			g.Player.HP -= dmg
			g.Log.Push(ui.PickHurtFlavor(g.RNG), ui.LogBad)
		}
	}
	if j := g.Floor.Jester; j != nil && j.Alive {
		if dmg := j.Act(g.Floor.Level, g.Player.Pos, occ, g.RNG); dmg > 0 {
			if g.Player.Invincible {
				if g.RNG.Chance(0.15) {
					g.Log.Push("The jester cackles, but you shrug it off. (INVINCIBLE)", ui.LogSpecial)
				}
			} else {
				g.Player.HP -= dmg
				if g.Player.HP <= 0 {
					g.Log.Push("Jester kills *you*", ui.LogBad)
				} else {
					g.Log.Push("The jester tells you a bad joke.", ui.LogBad)
				}
			}
		}
	}
}

// AdvanceEndSequence is called once per ticker pulse while the game is in
// PhaseEndSequence. When the sequence's pure-function timeline has played
// out (typed prompts done, spinners settled or failure message held), it
// tears down the EndSeq state and transitions to PhaseRickRoll. Calling this
// in any other phase is a no-op so the main loop can dispatch it
// unconditionally. As a side-effect it also nudges the contribution-graph
// generator: once the success-branch spinners begin, the EndSeq fires a
// one-shot goroutine that writes contribution-graph.svg.
func (g *Game) AdvanceEndSequence() {
	if g.Phase != PhaseEndSequence || g.EndSeq == nil {
		return
	}
	g.EndSeq.MaybeStartContribGraph(g.Tick)
	if ui.EndSequenceDone(g.EndSeq, g.Tick) {
		g.EndSeq.Cancel()
		g.EndSeq = nil
		g.Phase = PhaseRickRoll
	}
}

func (g *Game) ascend() {
	if g.Floor.Level.Depth >= 5 {
		// End of the final level: kick off the celebration sequence.
		// PhaseEndSequence handles the "You made it!" flash, the typed
		// `cd developers/developers/developers` / `build` / `git status`
		// shell, and the spinner output before the rick roll takes over.
		g.EndSeq = ui.NewEndSequence(g.Tick, g.Username)
		g.Phase = PhaseEndSequence
		return
	}
	nextDepth := g.Floor.Level.Depth + 1
	g.Floor = BuildFloor(nextDepth, g.Player, g.RNG, g.JesterDepth == nextDepth)
	letters := []byte{'B', 'U', 'I', 'L', 'D'}
	letter := byte('?')
	if d := g.Floor.Level.Depth; d >= 1 && d <= len(letters) {
		letter = letters[d-1]
	}
	g.Log.Push(fmt.Sprintf("You ascend to level %d — %c.", g.Floor.Level.Depth, letter), ui.LogInfo)
}
