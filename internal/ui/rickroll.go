package ui

import (
	_ "embed"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/leereilly/buildlike/internal/ui/palette"
)

// rickFramesRaw is the embedded ASCII animation. The file is git-ignored; drop
// frames into internal/ui/rickroll_frames.txt locally to enable the animation.
// Format options (auto-detected):
//   - Frames separated by a line containing only "===FRAME===" (recommended).
//   - Frames separated by form-feed (\f).
//   - Otherwise: fixed chunks of rickFrameHeight lines per frame.
//
// To use the upstream johnsoupir/ASCII_Rickroll BASH frames, strip the first
// 25 lines of header and save the rest as rickroll_frames.txt (no separator,
// 36 lines per frame).
//
//go:embed rickroll_frames.txt
var rickFramesRaw string

// rickFrameHeight is the line count per frame when no separator is present.
// Matches the upstream BASH script's 130x36 resolution.
const rickFrameHeight = 36

// rickFrameDelayTicks paces the animation. The main loop ticks every 100ms,
// so 1 tick/frame is ~10fps. The upstream uses ~8fps (0.12s sleep).
const rickFrameDelayTicks = 1

var rickFrames = parseRickFrames(rickFramesRaw)

func parseRickFrames(raw string) [][]string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	// Normalize newlines.
	raw = strings.ReplaceAll(raw, "\r\n", "\n")

	var chunks []string
	switch {
	case strings.Contains(raw, "===FRAME==="):
		chunks = strings.Split(raw, "===FRAME===")
	case strings.Contains(raw, "\f"):
		chunks = strings.Split(raw, "\f")
	default:
		lines := strings.Split(strings.TrimRight(raw, "\n"), "\n")
		for i := 0; i < len(lines); i += rickFrameHeight {
			end := i + rickFrameHeight
			if end > len(lines) {
				end = len(lines)
			}
			chunks = append(chunks, strings.Join(lines[i:end], "\n"))
		}
	}

	frames := make([][]string, 0, len(chunks))
	for _, c := range chunks {
		c = strings.Trim(c, "\n")
		if c == "" {
			continue
		}
		frames = append(frames, strings.Split(c, "\n"))
	}
	return frames
}

// RenderRickRoll draws one frame of the embedded ASCII animation, cycling
// through frames as the global tick advances. Falls back to a colorful banner
// when no frames are embedded.
func RenderRickRoll(s tcell.Screen, tick int) {
	Clear(s)
	w, h := s.Size()

	if len(rickFrames) == 0 {
		msg := "NEVER GONNA GIVE YOU UP"
		sub := "(drop frames into internal/ui/rickroll_frames.txt to animate)"
		c := palette.Cycle[(tick/2)%len(palette.Cycle)]
		DrawString(s, (w-len(msg))/2, h/2-1, msg, palette.FG(c).Bold(true))
		DrawString(s, (w-len(sub))/2, h/2+1, sub, palette.FG(palette.DimGray))
		return
	}

	frame := rickFrames[(tick/rickFrameDelayTicks)%len(rickFrames)]

	// Frame dimensions.
	fh := len(frame)
	fw := 0
	for _, line := range frame {
		if len(line) > fw {
			fw = len(line)
		}
	}

	// Center the frame in the full available height (the bottom hint that
	// used to sit on the last row has been removed).
	originX := (w - fw) / 2
	if originX < 0 {
		originX = 0
	}
	originY := (h - fh) / 2
	if originY < 0 {
		originY = 0
	}

	c := palette.Cycle[(tick/4)%len(palette.Cycle)]
	st := palette.FG(c)

	for i, line := range frame {
		y := originY + i
		if y < 0 || y >= h {
			continue
		}
		// Trim lines that would overflow the screen width.
		if originX+len(line) > w {
			cut := w - originX
			if cut < 0 {
				cut = 0
			}
			if cut < len(line) {
				line = line[:cut]
			}
		}
		DrawString(s, originX, y, line, st)
	}
}
