package fruit

import (
	"gofishing-game/service"
	"gofishing-game/service/roomutils"

	"github.com/guogeer/quasar/utils"
)

type FruitWorld struct{}

func init() {
	service.CreateWorld(&FruitWorld{})
}

func (w *FruitWorld) NewRoom(subId int) *roomutils.Room {
	r := &FruitRoom{}
	r.Room = roomutils.NewRoom(subId, r)
	// 定时同步
	utils.NewTimer(r.OnTime, syncTime)
	// r.StartGame()
	return r.Room
}

func (w *FruitWorld) GetName() string {
	return "fruit"
}

func (w *FruitWorld) NewPlayer() *service.Player {
	p := &FruitPlayer{}
	p.fruitObj = NewFruitObj(p)
	p.Player = service.NewPlayer(p)
	return p.Player
}
