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
	PhasePlaying
	PhaseHelp
	PhaseDead
	PhaseWon
	PhaseRickRoll
	PhaseQuit
)

type Game struct {
	Screen tcell.Screen
	RNG    *rng.RNG
	Player *entity.Player
	Floor  *FloorState
	Log    *ui.MessageLog
	Phase  Phase
	Tick   int // global tick counter for animation

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
}

func New(s tcell.Screen, r *rng.RNG) *Game {
	return &Game{
		Screen: s,
		RNG:    r,
		Player: entity.NewPlayer(),
		Log:    ui.NewLog(200),
		Phase:  PhaseTitle,
	}
}

// StartRun (re)initializes the player and generates level 1.
func (g *Game) StartRun() {
	g.Player = entity.NewPlayer()
	if g.KonamiArmed {
		g.Player.Invincible = true
	}
	g.ClippyPresses = 0
	g.CopilotBlinkUntil = 0
	g.Floor = BuildFloor(1, g.Player, g.RNG)
	g.Log = ui.NewLog(200)
	g.Log.Push("Welcome to Buildlike. Squash bugs, ship the build.", ui.LogInfo)
	if g.Player.Invincible {
		g.Log.Push("★ Konami code accepted: you are INVINCIBLE. ★", ui.LogSpecial)
	}
	g.Phase = PhasePlaying
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
}

func (g *Game) ascend() {
	if g.Floor.Level.Depth >= 5 {
		// End of the final level: cue the surprise before the victory screen.
		g.Phase = PhaseRickRoll
		return
	}
	g.Floor = BuildFloor(g.Floor.Level.Depth+1, g.Player, g.RNG)
	letters := []byte{'B', 'U', 'I', 'L', 'D'}
	letter := byte('?')
	if d := g.Floor.Level.Depth; d >= 1 && d <= len(letters) {
		letter = letters[d-1]
	}
	g.Log.Push(fmt.Sprintf("You ascend to level %d — %c.", g.Floor.Level.Depth, letter), ui.LogInfo)
}
