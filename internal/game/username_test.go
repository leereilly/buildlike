package game_test

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/leereilly/buildlike/internal/game"
	"github.com/leereilly/buildlike/internal/rng"
)

func newTestGame(t *testing.T) *game.Game {
	t.Helper()
	scr := tcell.NewSimulationScreen("")
	if err := scr.Init(); err != nil {
		t.Fatalf("init sim screen: %v", err)
	}
	t.Cleanup(scr.Fini)
	return game.New(scr, rng.New(1))
}

func TestInitialPhaseIsTitle(t *testing.T) {
	g := newTestGame(t)
	if g.Phase != game.PhaseTitle {
		t.Fatalf("expected initial phase PhaseTitle, got %v", g.Phase)
	}
	if g.Username != "" {
		t.Fatalf("expected empty initial Username, got %q", g.Username)
	}
}

func TestAppendUsernameAcceptsValidChars(t *testing.T) {
	g := newTestGame(t)
	for _, r := range "leereilly-42" {
		if !g.AppendUsernameRune(r) {
			t.Fatalf("expected %q to be accepted, was rejected (Username=%q)", r, g.Username)
		}
	}
	if g.Username != "leereilly-42" {
		t.Fatalf("expected Username=%q, got %q", "leereilly-42", g.Username)
	}
}

func TestAppendUsernameRejectsInvalidChars(t *testing.T) {
	g := newTestGame(t)
	for _, r := range []rune{' ', '@', '!', '.', '/', '\n'} {
		if g.AppendUsernameRune(r) {
			t.Errorf("expected %q to be rejected, was accepted", r)
		}
	}
	if g.Username != "" {
		t.Fatalf("expected Username to stay empty, got %q", g.Username)
	}
}

func TestAppendUsernameRejectsLeadingHyphen(t *testing.T) {
	g := newTestGame(t)
	if g.AppendUsernameRune('-') {
		t.Fatalf("leading hyphen should be rejected")
	}
}

func TestAppendUsernameRejectsConsecutiveHyphens(t *testing.T) {
	g := newTestGame(t)
	for _, r := range "a-" {
		if !g.AppendUsernameRune(r) {
			t.Fatalf("expected %q to be accepted", r)
		}
	}
	if g.AppendUsernameRune('-') {
		t.Fatalf("consecutive hyphen should be rejected")
	}
}

func TestAppendUsernameEnforcesLengthCap(t *testing.T) {
	g := newTestGame(t)
	for i := 0; i < game.MaxUsernameLen; i++ {
		if !g.AppendUsernameRune('a') {
			t.Fatalf("char %d should fit under the cap", i)
		}
	}
	if g.AppendUsernameRune('a') {
		t.Fatalf("appending past MaxUsernameLen should be rejected")
	}
	if len(g.Username) != game.MaxUsernameLen {
		t.Fatalf("expected Username length=%d, got %d", game.MaxUsernameLen, len(g.Username))
	}
}

func TestBackspaceUsername(t *testing.T) {
	g := newTestGame(t)
	if g.BackspaceUsername() {
		t.Fatalf("backspace on empty Username should return false")
	}
	for _, r := range "abc" {
		g.AppendUsernameRune(r)
	}
	if !g.BackspaceUsername() {
		t.Fatalf("backspace on non-empty Username should return true")
	}
	if g.Username != "ab" {
		t.Fatalf("expected Username=%q after backspace, got %q", "ab", g.Username)
	}
}

func TestUsernameReady(t *testing.T) {
	g := newTestGame(t)
	if g.UsernameReady() {
		t.Fatalf("empty username should not be ready")
	}
	for _, r := range "abc" {
		g.AppendUsernameRune(r)
	}
	if !g.UsernameReady() {
		t.Fatalf("plain handle should be ready")
	}
	g.AppendUsernameRune('-')
	if g.UsernameReady() {
		t.Fatalf("handle ending in hyphen should not be ready")
	}
}

// TestUsernameOnlyAcceptsGitHubCharset spot-checks that the validator
// matches GitHub's documented charset: ASCII alphanumerics and single
// hyphens.
func TestUsernameOnlyAcceptsGitHubCharset(t *testing.T) {
	g := newTestGame(t)
	const allowed = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	for _, r := range allowed {
		g := newTestGame(t)
		if !g.AppendUsernameRune(r) {
			t.Errorf("expected %q (allowed) to be accepted", r)
		}
		_ = g
	}
	const rejected = " \t!@#$%^&*()+=[]{}|\\:;\"'<>,.?/~`"
	for _, r := range rejected {
		if g.AppendUsernameRune(r) {
			t.Errorf("expected %q (disallowed) to be rejected", r)
		}
	}
	// Sanity: nothing in the rejected set leaked into Username.
	if strings.ContainsAny(g.Username, rejected) {
		t.Fatalf("Username %q contains disallowed chars", g.Username)
	}
}
