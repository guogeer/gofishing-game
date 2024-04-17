package internal

import (
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
)

type fingerGuessingWorld struct {
}

func GetWorld() *fingerGuessingWorld {
	return service.GetWorld().(*fingerGuessingWorld)
}

func init() {
	w := &fingerGuessingWorld{}
	service.CreateWorld(w)
	roomutils.LoadGames(w)
}

func (w *fingerGuessingWorld) GetName() string {
	return "fingerGuessing"
}

func (w *fingerGuessingWorld) NewPlayer() *service.Player {
	p := &fingerGuessingPlayer{}
	p.Player = service.NewPlayer(p)
	p.Player.DataObj().Push(p)

	return p.Player
}

func (w *fingerGuessingWorld) NewRoom(subId int) *roomutils.Room {
	room := &fingerGuessingRoom{}
	room.Room = roomutils.NewRoom(subId, room)
	return room.Room
}

func GetPlayer(uid int) *fingerGuessingPlayer {
	if p := service.GetPlayer(uid); p != nil {
		return p.GameAction.(*fingerGuessingPlayer)
	}
	return nil
}
