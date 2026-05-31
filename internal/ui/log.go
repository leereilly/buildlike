package ui

import (
	"github.com/leereilly/commit-crawl/internal/rng"
)

type LogKind int

const (
	LogInfo LogKind = iota
	LogGood
	LogBad
	LogWarn
	LogSpecial
)

type LogEntry struct {
	Text string
	Kind LogKind
}

type MessageLog struct {
	Entries []LogEntry
	Cap     int
}

func NewLog(cap int) *MessageLog { return &MessageLog{Cap: cap} }

func (m *MessageLog) Push(text string, kind LogKind) {
	m.Entries = append(m.Entries, LogEntry{text, kind})
	if len(m.Entries) > m.Cap {
		m.Entries = m.Entries[len(m.Entries)-m.Cap:]
	}
}

// Tail returns the last n entries.
func (m *MessageLog) Tail(n int) []LogEntry {
	if n <= 0 || len(m.Entries) == 0 {
		return nil
	}
	if n > len(m.Entries) {
		n = len(m.Entries)
	}
	return m.Entries[len(m.Entries)-n:]
}

var killFlavors = []string{
	"You squash a bug. It crunches satisfyingly.",
	"Segfault avoided.",
	"Code review: rejected.",
	"It crashes. Literally.",
	"That bug is now a feature. Of the floor.",
	"Build status: green. Bug status: red.",
	"You file an issue. With your boot.",
	"Patched in place.",
}

var hurtFlavors = []string{
	"A bug bites you. Rude.",
	"Production incident! -1 HP.",
	"Stack trace: ow.",
	"That's a regression on your morale.",
	"The bug crashes into you.",
	"Off-by-one ouchie.",
}

var powerupFlavors = []string{
	"Caffeinated. +1 HP.",
	"+1 to morale.",
	"Stack Overflow approved. +1 HP.",
	"You feel slightly healthier. (+1 HP)",
	"Hot fix. +1 HP.",
	"What a diff that makes. +1 HP.",
}

// Title-screen tip rotation.
var Tips = []string{
	"Tip: bump a bug to squash it.",
	"Tip: every level has stairs. They look like >.",
	"Tip: green + is health. Eat it.",
	"Tip: there are rumors of a vault on level 2.",
	"Tip: hjkl, arrows, WASD, and the numpad all work.",
	"Tip: press ? at any time for help.",
	"Tip: press . or space to wait a turn.",
}

func PickKillFlavor(r *rng.RNG) string    { return killFlavors[r.Intn(len(killFlavors))] }
func PickHurtFlavor(r *rng.RNG) string    { return hurtFlavors[r.Intn(len(hurtFlavors))] }
func PickPowerupFlavor(r *rng.RNG) string { return powerupFlavors[r.Intn(len(powerupFlavors))] }
