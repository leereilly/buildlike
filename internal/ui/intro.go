package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/leereilly/commit-crawl/internal/entity"
	"github.com/leereilly/commit-crawl/internal/ui/palette"
	"github.com/leereilly/commit-crawl/internal/world"
)

// IntroHoldTicks is the number of 100ms ticks the intro pauses on a cleared
// screen showing only the player's '@' before the world begins to bloom.
const IntroHoldTicks = 4

// IntroRevealTicks is the number of 100ms ticks spent revealing the first
// floor and gliding the player avatar from the username field to the spawn.
const IntroRevealTicks = 22

// IntroDurationTicks is the total length of the post-username intro.
const IntroDurationTicks = IntroHoldTicks + IntroRevealTicks

// introSparkles cycle in front of cells that have not yet been revealed by
// the bloom wavefront. Matches the spirit of the BUILD teleport sparkles so
// the two transitions feel like the same dialect.
var introSparkles = []rune{'·', '*', '+', '✦', '◆'}

// IntroState carries the per-frame data the intro transition needs to glide
// the '@' from the username field into the spawn cell while the first floor
// blooms into view.
type IntroState struct {
	// SrcX, SrcY is the screen cell where the magenta '@' glyph sat at the
	// end of PhaseUsername. The avatar starts the slide here.
	SrcX, SrcY int
	// StartTick is the global Tick at which BeginIntro was called. The
	// renderer derives every animation frame from (Tick - StartTick) so the
	// caller only has to keep ticking and call RenderIntro.
	StartTick int
}

// IntroDone reports whether the intro animation has finished and the game
// should advance to PhasePlaying.
func IntroDone(st *IntroState, tick int) bool {
	return st == nil || tick-st.StartTick >= IntroDurationTicks
}

// RenderIntro draws frame `tick` of the intro transition.
//
// The animation has two halves:
//   - For the first IntroHoldTicks ticks the screen is cleared except for a
//     single yellow '@' at the username field position. This is the "the rest
//     of the world has fallen away" beat.
//   - Then a manhattan-distance bloom spreads out from the spawn cell while
//     the '@' slides on an eased path from the field to the spawn. Cells
//     ahead of the wavefront flicker with rainbow sparkles; cells behind it
//     settle into their normal map colours. The avatar leaves a brief rainbow
//     trail along its glide path.
func RenderIntro(s tcell.Screen, st *IntroState, p *entity.Player, fs FloorView, tick int) {
	Clear(s)
	if st == nil || p == nil || fs == nil {
		return
	}
	elapsed := tick - st.StartTick
	if elapsed < 0 {
		elapsed = 0
	}

	if elapsed < IntroHoldTicks {
		DrawRune(s, st.SrcX, st.SrcY, '@', palette.FG(palette.Yellow).Bold(true))
		return
	}

	reveal := elapsed - IntroHoldTicks
	if reveal > IntroRevealTicks {
		reveal = IntroRevealTicks
	}

	const offsetY = 1
	l := fs.GetLevel()
	dstX, dstY := p.Pos.X, p.Pos.Y+offsetY

	drawMapBloom(s, l, fs, dstX, dstY-offsetY, reveal, IntroRevealTicks)

	// Eased slide of the avatar from the username position to the spawn.
	prog := float64(reveal) / float64(IntroRevealTicks)
	if prog > 1.0 {
		prog = 1.0
	}
	eased := easeInOutCubic(prog)
	atX := lerpInt(st.SrcX, dstX, eased)
	atY := lerpInt(st.SrcY, dstY, eased)

	drawIntroTrail(s, st.SrcX, st.SrcY, dstX, dstY, eased, tick)
	DrawRune(s, atX, atY, '@', palette.FG(palette.Yellow).Bold(true))
}

// drawMapBloom renders cells whose manhattan distance to (spawnMapX,
// spawnMapY) — coordinates in *map* space — is within the current bloom
// radius. Cells just inside the wavefront pulse with sparkles for one or
// two ticks so the bloom edge reads as a wavefront, not a hard line.
func drawMapBloom(s tcell.Screen, l *world.Level, fs FloorView, spawnMapX, spawnMapY, reveal, total int) {
	const offsetY = 1
	maxRadius := l.W + l.H
	radius := reveal * maxRadius / total
	for y := 0; y < l.H; y++ {
		for x := 0; x < l.W; x++ {
			dist := absInt(x-spawnMapX) + absInt(y-spawnMapY)
			if dist > radius {
				continue
			}
			// Edge-of-wave shimmer: cells revealed in the last couple of
			// ticks render as a sparkle before settling to their final glyph.
			ticksSinceReveal := radius - dist
			if ticksSinceReveal < 2 {
				ch := introSparkles[(x*7+y*13+reveal)%len(introSparkles)]
				c := palette.Cycle[(x+y+reveal)%len(palette.Cycle)]
				DrawRune(s, x, y+offsetY, ch, palette.FG(c).Bold(true))
				continue
			}
			drawTile(s, l, x, y, offsetY)
		}
	}
	for _, pu := range fs.GetPowerups() {
		if pu.Picked {
			continue
		}
		dist := absInt(pu.Pos.X-spawnMapX) + absInt(pu.Pos.Y-spawnMapY)
		if dist > radius {
			continue
		}
		DrawRune(s, pu.Pos.X, pu.Pos.Y+offsetY, '+', palette.FG(palette.Green).Bold(true))
	}
	// Bugs and the jester are deliberately hidden during the intro so the
	// first thing the player sees is a quiet, hostile-free overview of the
	// floor they just landed on. They appear naturally on the first
	// PhasePlaying render.
}

// drawIntroTrail draws a short rainbow streak between (srcX, srcY) and the
// current eased position of the avatar so the slide has visual weight.
func drawIntroTrail(s tcell.Screen, srcX, srcY, dstX, dstY int, eased float64, tick int) {
	const samples = 8
	for i := 1; i <= samples; i++ {
		t := eased * float64(i) / float64(samples+1)
		x := lerpInt(srcX, dstX, t)
		y := lerpInt(srcY, dstY, t)
		c := palette.Cycle[(i+tick)%len(palette.Cycle)]
		DrawRune(s, x, y, '·', palette.FG(c).Bold(true))
	}
}

func lerpInt(a, b int, t float64) int {
	v := float64(a) + (float64(b)-float64(a))*t
	if v >= 0 {
		return int(v + 0.5)
	}
	return int(v - 0.5)
}

func easeInOutCubic(t float64) float64 {
	if t < 0.5 {
		return 4 * t * t * t
	}
	p := -2*t + 2
	return 1 - p*p*p/2
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
