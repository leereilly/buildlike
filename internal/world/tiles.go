// Package world models the dungeon: tiles, levels, BSP generation, and LOS.
package world

// Tile is one cell of the dungeon grid.
type Tile uint8

const (
	TileWall Tile = iota
	TileFloor
	TileStairs
	TileSecretDoor // renders as wall until revealed
	TilePlate      // pressure plate (BUILD vault easter egg)
)

func (t Tile) Walkable() bool {
	switch t {
	case TileFloor, TileStairs, TilePlate:
		return true
	}
	return false
}

// Glyph returns the rune for rendering this tile.
func (t Tile) Glyph() rune {
	switch t {
	case TileFloor:
		return '.'
	case TileStairs:
		return '>'
	case TilePlate:
		return '*'
	case TileSecretDoor:
		return '#' // disguised
	}
	return '#'
}
