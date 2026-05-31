package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/leereilly/commit-crawl/internal/ui/palette"
)

// UsernameAtPos returns the screen cell where the brand-magenta '@' glyph is
// drawn during PhaseUsername for the given terminal size. The post-username
// intro transition uses this to glide the player avatar from its on-screen
// resting position into the floor spawn without snapping.
func UsernameAtPos(w, h int) (int, int) {
	const fieldWidth = 42
	cy := h / 2
	if cy < 6 {
		cy = 6
	}
	boxX := (w - fieldWidth) / 2
	boxY := cy - 1
	return boxX + 2, boxY + 1
}

// RenderUsername draws the opening "enter your GitHub handle" screen.
//
// TUI design notes:
//   - The '@' sigil is rendered to the left of the input field in the brand
//     magenta, separated from the editable area by a vertical bar. This is a
//     standard "fixed prefix / addon" pattern (think Bootstrap input groups,
//     or the way `gh` prints handles): the prefix is visually distinct from
//     the field so the user can see at a glance that they should NOT retype
//     the '@'.
//   - A second hint line under the box restates the same idea in words for
//     screen readers and players new to the convention.
//   - A blinking block cursor on `tick` parity provides a clear caret without
//     needing tcell's real cursor (which we keep hidden for the rest of the
//     game).
//   - The "Press Enter" affordance is dimmed until the handle is valid, so
//     submission is discoverable but never ambiguous.
func RenderUsername(s tcell.Screen, username string, tick int, ready bool) {
	Clear(s)
	w, h := s.Size()

	const (
		fieldWidth = 42
		prefixCols = 4 // "│ @ │" inside the box (left border + space + @ + space + bar)
	)

	atX, atY := UsernameAtPos(w, h)
	cy := atY + 1 // UsernameAtPos returns (boxX+2, boxY+1) where boxY = cy-1
	boxX := atX - 2
	boxY := atY - 1

	welcome := "Welcome to Commit Crawl!"
	DrawString(s, (w-runeLen(welcome))/2, cy-5,
		welcome, palette.FG(palette.Yellow).Bold(true))

	prompt := "Type your GitHub username to begin:"
	DrawString(s, (w-runeLen(prompt))/2, cy-3,
		prompt, palette.FG(palette.White))

	// Input frame.
	drawBox(s, boxX, boxY, fieldWidth, 3, palette.FG(palette.DimGray))

	// Prefix cell: "│ @ │"  — the literal '@' sits inside the field, in the
	// brand magenta, with a divider after it so it reads as a non-editable
	// addon.
	DrawRune(s, atX, atY, '@', palette.FG(palette.Magenta).Bold(true))
	DrawRune(s, boxX+4, boxY+1, '│', palette.FG(palette.DimGray))

	// Editable area starts after the divider.
	editX := boxX + prefixCols + 2 // box-left + "│ @ │" + space
	editWidth := fieldWidth - (prefixCols + 3)

	// Render the typed username, truncating from the left if it overflows
	// the field so the cursor stays visible.
	display := username
	if len(display) > editWidth-1 {
		display = display[len(display)-(editWidth-1):]
	}
	DrawString(s, editX, boxY+1, display, palette.FG(palette.White).Bold(true))

	// Blinking cursor (block on even ticks, hollow on odd).
	cursorX := editX + len(display)
	if cursorX < editX+editWidth {
		cursor := '▮'
		if (tick/5)%2 == 1 {
			cursor = '▯'
		}
		DrawRune(s, cursorX, boxY+1, cursor, palette.FG(palette.Yellow).Bold(true))
	}

	// Hint line directly under the field reinforces that '@' is pre-filled.
	hint := "↑ the @ is already on us — just type your handle"
	DrawString(s, (w-runeLen(hint))/2, boxY+4,
		hint, palette.FG(palette.DimGray))

	// Affordance row.
	enterColor := palette.DimGray
	enterText := "[Enter] start  ·  [Esc] quit"
	if ready {
		enterColor = palette.Green
		enterText = "[Enter] start your run  ·  [Esc] quit"
	}
	DrawString(s, (w-runeLen(enterText))/2, boxY+6,
		enterText, palette.FG(enterColor).Bold(ready))
}

// drawBox renders a single-line box at (x, y) of the given width and height
// using the rounded box-drawing characters.
func drawBox(s tcell.Screen, x, y, w, h int, st tcell.Style) {
	if w < 2 || h < 2 {
		return
	}
	DrawRune(s, x, y, '╭', st)
	DrawRune(s, x+w-1, y, '╮', st)
	DrawRune(s, x, y+h-1, '╰', st)
	DrawRune(s, x+w-1, y+h-1, '╯', st)
	for i := 1; i < w-1; i++ {
		DrawRune(s, x+i, y, '─', st)
		DrawRune(s, x+i, y+h-1, '─', st)
	}
	for i := 1; i < h-1; i++ {
		DrawRune(s, x, y+i, '│', st)
		DrawRune(s, x+w-1, y+i, '│', st)
	}
}
