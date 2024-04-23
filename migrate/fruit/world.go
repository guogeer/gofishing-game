package fruit

import (
	"gofishing-game/service"

	"github.com/guogeer/quasar/util"
)

type FruitWorld struct{}

func init() {
	service.CreateWorld(&FruitWorld{})
}

func (w *FruitWorld) NewRoom(id, subId int) *service.Room {
	r := &FruitRoom{}
	r.Room = service.NewRoom(id, subId, r)
	// 定时同步
	util.NewTimer(r.OnTime, syncTime)
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
