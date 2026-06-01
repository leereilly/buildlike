// Package rng wraps math/rand with a seedable source so the entire game can be
// made reproducible via the --seed CLI flag.
package rng

import (
	"math/rand"
	"time"
)

type RNG struct {
	r *rand.Rand
}

func New(seed int64) *RNG {
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	return &RNG{r: rand.New(rand.NewSource(seed))}
}

func (g *RNG) Intn(n int) int                     { return g.r.Intn(n) }
func (g *RNG) IntRange(lo, hi int) int            { return lo + g.r.Intn(hi-lo+1) }
func (g *RNG) Float64() float64                   { return g.r.Float64() }
func (g *RNG) Chance(p float64) bool              { return g.r.Float64() < p }
func (g *RNG) Pick(n int) int                     { return g.r.Intn(n) }
func (g *RNG) Shuffle(n int, swap func(i, j int)) { g.r.Shuffle(n, swap) }
