package dice

import (
	"gofishing-game/service"
	"gofishing-game/service/roomutils"

	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/utils"
)

func init() {
	service.AddWorld(&DiceWorld{})
}

type DiceWorld struct{}

func (w *DiceWorld) NewRoom(subId int) *roomutils.Room {
	r := &DiceRoom{}
	r.Room = roomutils.NewRoom(subId, r)

	var chips []int64
	config.Scan("config", "diceChips", "value", config.JSON(&chips))
	// 定时同步
	utils.NewTimer(r.OnTime, syncTime)
	r.StartGame()
	return r.Room
}

func (w *DiceWorld) GetName() string {
	return "dice"
}

func (w *DiceWorld) NewPlayer() *service.Player {
	p := &DicePlayer{}
	p.Player = service.NewPlayer(p)
	return p.Player
}
