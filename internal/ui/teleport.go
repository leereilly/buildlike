package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/leereilly/buildlike/internal/ui/palette"
	"github.com/leereilly/buildlike/internal/world"
)

// TeleportDurationTicks is how many 100ms ticks the teleport animation runs
// before automatically returning to play. The animation can also be skipped
// with any key.
const TeleportDurationTicks = 30

// sparkleChars cycle through during cell materialization.
var sparkleChars = []rune{'·', '*', '+', '✦', '◆', '▓'}

// RenderTeleport draws the animated transition between two dungeon letters.
// The previous letter is shown small and dim on the left (so the player can
// see where they came from); the new letter materializes via colored
// sparkles that resolve into a solid silhouette on the right; an animated
// beam joins them. A "B U I L D" progress strip sits at the top.
func RenderTeleport(s tcell.Screen, prev, next *world.LetterMask, tick int) {
	Clear(s)
	w, h := s.Size()

	// Tiny-terminal fallback: just a centered shimmer. The thicker BUILD
	// letters produce a 48×20 destination stamp at nsx=6/nsy=2, so we need
	// a standard 80×24 terminal to give the header + bottom hint breathing
	// room. Anything smaller drops to the fallback.
	if w < 70 || h < 24 {
		msg := "TELEPORTING..."
		c := palette.Cycle[(tick/2)%len(palette.Cycle)]
		DrawString(s, max0((w-len(msg))/2), h/2, msg, palette.FG(c).Bold(true))
		return
	}

	drawTeleportHeader(s, prev, next, w, tick)

	centerY := h / 2

	// --- Previous letter (small, dim) on the left ---
	const psx, psy = 2, 1
	pStampH, pStampW := 10*psy, 8*psx
	pOriginX, pOriginY := 3, centerY-pStampH/2
	if prev != nil {
		drawStampOutline(s, world.Stamp(prev.Letter), pOriginX, pOriginY, psx, psy, palette.DimGray)
		// Letter glyph + a left arrow as a "you came from here" label.
		labelY := pOriginY + pStampH + 1
		DrawRune(s, pOriginX+pStampW/2-1, labelY, '←', palette.FG(palette.DimGray))
		DrawRune(s, pOriginX+pStampW/2+1, labelY, rune(prev.Letter),
			palette.FG(palette.Cycle[prev.ColorIx%len(palette.Cycle)]).Bold(true))
	}

	// --- New letter (large, materializing) on the right ---
	const nsx, nsy = 6, 2
	nStampH, nStampW := 10*nsy, 8*nsx
	nOriginX, nOriginY := w-nStampW-3, centerY-nStampH/2
	if next != nil {
		drawMaterializingStamp(s, world.Stamp(next.Letter), nOriginX, nOriginY, nsx, nsy, next.ColorIx, tick)
		// No glyph label here — the giant materialised letter is the label.
	}

	// --- Animated beam between them ---
	beamLeft := pOriginX + pStampW + 2
	beamRight := nOriginX - 2
	if beamRight > beamLeft+4 {
		drawTeleportBeam(s, beamLeft, beamRight, centerY, tick)
	}

	// --- Bottom hint ---
	hint := "Teleporting...  press any key to continue"
	DrawString(s, max0((w-len(hint))/2), h-2, hint, palette.FG(palette.White))
}

// drawTeleportHeader draws the "[B] [U] [I] [L] [D]" progress row at the top
// of the screen. The destination letter pulses in its color; the source
// letter sits in steady color; remaining letters are dim.
func drawTeleportHeader(s tcell.Screen, prev, next *world.LetterMask, w, tick int) {
	letters := []byte{'B', 'U', 'I', 'L', 'D'}
	const cell = 4 // "[X] "
	total := cell*len(letters) - 1
	x := max0((w - total) / 2)
	y := 1
	for i, ch := range letters {
		var st tcell.Style
		switch {
		case next != nil && byte(ch) == next.Letter:
			pulse := palette.Cycle[(i+tick/2)%len(palette.Cycle)]
			st = palette.FG(pulse).Bold(true)
		case prev != nil && byte(ch) == prev.Letter:
			st = palette.FG(palette.Cycle[i%len(palette.Cycle)])
		default:
			st = palette.FG(palette.DimGray)
		}
		DrawRune(s, x, y, '[', palette.FG(palette.DimGray))
		DrawRune(s, x+1, y, rune(ch), st)
		DrawRune(s, x+2, y, ']', palette.FG(palette.DimGray))
		x += cell
	}
}

// drawTeleportBeam animates a horizontal rainbow beam with travelling dashes
// pointing from the source letter to the destination letter.
func drawTeleportBeam(s tcell.Screen, x1, x2, y, tick int) {
	for x := x1; x <= x2; x++ {
		offs := mod6(x + tick)
		var ch rune
		switch offs {
		case 0:
			ch = '>'
		case 1, 5:
			ch = '═'
		case 3:
			ch = '·'
		default:
			ch = '─'
		}
		c := palette.Cycle[mod(x*7+tick, len(palette.Cycle))]
		DrawRune(s, x, y, ch, palette.FG(c).Bold(true))
	}
	// Bright arrowhead at the destination side.
	DrawRune(s, x2, y, '►', palette.FG(palette.Cycle[(tick/2)%len(palette.Cycle)]).Bold(true))
}

// drawStampOutline renders only the silhouette edge of `stamp` (each inside
// cell that has at least one outside neighbour) at the given scale.
func drawStampOutline(s tcell.Screen, stamp []string, originX, originY, sx, sy int, c tcell.Color) {
	if stamp == nil {
		return
	}
	sh, sw := stampDims(stamp)
	inside := func(lx, ly int) bool {
		return lx >= 0 && ly >= 0 && lx < sw && ly < sh && stamp[ly][lx] == '#'
	}
	style := palette.FG(c)
	for ly := 0; ly < sh; ly++ {
		for lx := 0; lx < sw; lx++ {
			if !inside(lx, ly) {
				continue
			}
			edge := !inside(lx-1, ly) || !inside(lx+1, ly) || !inside(lx, ly-1) || !inside(lx, ly+1)
			if !edge {
				continue
			}
			for dy := 0; dy < sy; dy++ {
				for dx := 0; dx < sx; dx++ {
					DrawRune(s, originX+lx*sx+dx, originY+ly*sy+dy, '█', style)
				}
			}
		}
	}
}

// drawMaterializingStamp renders `stamp` at the given scale with a tick-
// driven materialization effect: each pixel of the scaled stamp has a
// deterministic "reveal" tick; before reveal it shimmers with a cycling
// sparkle character in a random palette colour; once revealed it settles
// to a solid block in the letter's signature colour.
func drawMaterializingStamp(s tcell.Screen, stamp []string, originX, originY, sx, sy, colorIx, tick int) {
	if stamp == nil {
		return
	}
	sh, sw := stampDims(stamp)
	inside := func(lx, ly int) bool {
		return lx >= 0 && ly >= 0 && lx < sw && ly < sh && stamp[ly][lx] == '#'
	}
	baseColor := palette.Cycle[colorIx%len(palette.Cycle)]
	const maxReveal = 22
	const settleDelay = 4
	for ly := 0; ly < sh; ly++ {
		for lx := 0; lx < sw; lx++ {
			if !inside(lx, ly) {
				continue
			}
			for dy := 0; dy < sy; dy++ {
				for dx := 0; dx < sx; dx++ {
					px := originX + lx*sx + dx
					py := originY + ly*sy + dy
					rev := mod(simpleHash(px, py), maxReveal)
					if tick >= rev+settleDelay {
						DrawRune(s, px, py, '█', palette.FG(baseColor))
					} else {
						chIdx := mod(simpleHash(px, py)+tick, len(sparkleChars))
						cIdx := mod(simpleHash(px, py)/3+tick, len(palette.Cycle))
						DrawRune(s, px, py, sparkleChars[chIdx], palette.FG(palette.Cycle[cIdx]).Bold(true))
					}
				}
			}
		}
	}
}

// FormatFloorTitle returns a short "Floor N — X" label used in the teleport
// header bottom row. Exported in case other phases want the same string.
func FormatFloorTitle(letter byte, depth int) string {
	return fmt.Sprintf("Floor %d — %c", depth, letter)
}

func stampDims(stamp []string) (h, w int) {
	h = len(stamp)
	if h > 0 {
		w = len(stamp[0])
	}
	return
}

func simpleHash(x, y int) int {
	h := x*73856093 ^ y*19349663
	if h < 0 {
		h = -h
	}
	return h
}

func mod(a, n int) int {
	r := a % n
	if r < 0 {
		r += n
	}
	return r
}

func mod6(x int) int { return mod(x, 6) }

func max0(x int) int {
	if x < 0 {
		return 0
	}
	return x
}
