package entity

import "github.com/leereilly/buildlike/internal/world"

type Player struct {
	Pos        world.Point
	HP         int
	MaxHP      int
	Vaulted    bool // grabbed the BUILD vault bonus this run
	Invincible bool // Konami-code cheat: bugs cannot deal damage
}

func NewPlayer() *Player {
	return &Player{HP: 12, MaxHP: 12}
}

type Powerup struct {
	Pos    world.Point
	Picked bool
}
