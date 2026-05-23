package entity

import "github.com/leereilly/buildlike/internal/world"

type Player struct {
	Pos     world.Point
	HP      int
	MaxHP   int
	Vaulted bool // grabbed the BUILD vault bonus this run
}

func NewPlayer() *Player {
	return &Player{HP: 12, MaxHP: 12}
}

type Powerup struct {
	Pos    world.Point
	Picked bool
}
