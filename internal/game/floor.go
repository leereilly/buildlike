package game

import (
	"fmt"

	"github.com/leereilly/buildlike/internal/entity"
	"github.com/leereilly/buildlike/internal/rng"
	"github.com/leereilly/buildlike/internal/world"
)

// FloorState holds everything for the currently-active dungeon floor.
type FloorState struct {
	Level    *world.Level
	Bugs     []*entity.Bug
	Powerups []*entity.Powerup
}

// BuildFloor generates a floor for the given depth (1-3) with bugs and powerups.
func BuildFloor(depth int, p *entity.Player, r *rng.RNG) *FloorState {
	const W, H = 80, 22
	var l *world.Level
	for attempt := 0; attempt < 12; attempt++ {
		l = world.NewLevel(W, H, depth)
		if world.Generate(l, r) {
			break
		}
	}
	if depth == 2 {
		world.PlaceBuildVault(l, r)
	}

	p.Pos = l.Spawn
	fs := &FloorState{Level: l}

	// Bug counts per depth.
	bugCounts := map[int]int{1: 4, 2: 7, 3: 10}
	occ := map[world.Point]bool{p.Pos: true}
	for i := 0; i < bugCounts[depth]; i++ {
		pos, ok := randomFloor(l, r, occ, true)
		if !ok {
			break
		}
		fs.Bugs = append(fs.Bugs, entity.NewBug(pos))
		occ[pos] = true
	}
	// Powerups: 3 per level.
	for i := 0; i < 3; i++ {
		pos, ok := randomFloor(l, r, occ, true)
		if !ok {
			break
		}
		fs.Powerups = append(fs.Powerups, &entity.Powerup{Pos: pos})
		occ[pos] = true
	}
	return fs
}

// randomFloor picks a random floor tile not in `occ`. If avoidVault is true,
// the BUILD vault (disconnected from main dungeon) is excluded by reachability check.
func randomFloor(l *world.Level, r *rng.RNG, occ map[world.Point]bool, avoidVault bool) (world.Point, bool) {
	candidates := []world.Point{}
	for y := 0; y < l.H; y++ {
		for x := 0; x < l.W; x++ {
			p := world.Point{X: x, Y: y}
			if l.Tiles[y][x] != world.TileFloor {
				continue
			}
			if occ[p] {
				continue
			}
			if avoidVault && l.HasVault && !l.FloodFillReachable(l.Spawn, p) {
				continue
			}
			candidates = append(candidates, p)
		}
	}
	if len(candidates) == 0 {
		return world.Point{}, false
	}
	return candidates[r.Intn(len(candidates))], true
}

// String for debugging.
func (fs *FloorState) String() string {
	return fmt.Sprintf("Floor depth=%d rooms=%d bugs=%d powerups=%d", fs.Level.Depth, len(fs.Level.Rooms), len(fs.Bugs), len(fs.Powerups))
}

func (fs *FloorState) GetLevel() *world.Level         { return fs.Level }
func (fs *FloorState) GetBugs() []*entity.Bug         { return fs.Bugs }
func (fs *FloorState) GetPowerups() []*entity.Powerup { return fs.Powerups }
