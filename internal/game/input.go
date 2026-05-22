package game

import "github.com/gdamore/tcell/v2"

// Action is what the player wants to do this turn.
type Action int

const (
	ActNone Action = iota
	ActMoveN
	ActMoveS
	ActMoveW
	ActMoveE
	ActMoveNW
	ActMoveNE
	ActMoveSW
	ActMoveSE
	ActWait
	ActAscend
	ActHelp
	ActQuit
	ActConfirm // any key on title/death/rickroll screens
)

// MapKey translates a tcell key event into a game Action. All control schemes
// (arrows, WASD, vi-keys, numpad) are active simultaneously.
func MapKey(ev *tcell.EventKey) Action {
	switch ev.Key() {
	case tcell.KeyUp:
		return ActMoveN
	case tcell.KeyDown:
		return ActMoveS
	case tcell.KeyLeft:
		return ActMoveW
	case tcell.KeyRight:
		return ActMoveE
	case tcell.KeyEnter:
		return ActConfirm
	case tcell.KeyEscape, tcell.KeyCtrlC:
		return ActQuit
	}
	switch ev.Rune() {
	// vi-keys
	case 'h':
		return ActMoveW
	case 'j':
		return ActMoveS
	case 'k':
		return ActMoveN
	case 'l':
		return ActMoveE
	case 'y':
		return ActMoveNW
	case 'u':
		return ActMoveNE
	case 'b':
		return ActMoveSW
	case 'n':
		return ActMoveSE
	// WASD
	case 'w', 'W':
		return ActMoveN
	case 'a', 'A':
		return ActMoveW
	case 's', 'S':
		return ActMoveS
	case 'd', 'D':
		return ActMoveE
	// numpad
	case '8':
		return ActMoveN
	case '2':
		return ActMoveS
	case '4':
		return ActMoveW
	case '6':
		return ActMoveE
	case '7':
		return ActMoveNW
	case '9':
		return ActMoveNE
	case '1':
		return ActMoveSW
	case '3':
		return ActMoveSE
	case '5', '.', ' ':
		return ActWait
	case '>':
		return ActAscend
	case '?':
		return ActHelp
	case 'q', 'Q':
		return ActQuit
	}
	return ActNone
}

// Delta returns the (dx, dy) for a movement action, or (0,0,false) if not a move.
func Delta(a Action) (int, int, bool) {
	switch a {
	case ActMoveN:
		return 0, -1, true
	case ActMoveS:
		return 0, 1, true
	case ActMoveW:
		return -1, 0, true
	case ActMoveE:
		return 1, 0, true
	case ActMoveNW:
		return -1, -1, true
	case ActMoveNE:
		return 1, -1, true
	case ActMoveSW:
		return -1, 1, true
	case ActMoveSE:
		return 1, 1, true
	}
	return 0, 0, false
}
