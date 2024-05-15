package threedice

import (
	"container/list"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
	"time"

	"github.com/guogeer/quasar/util"
)

func init() {
	service.CreateWorld("夺宝王", &ThreeDiceWorld{})
}

type ThreeDiceWorld struct{}

func (w *ThreeDiceWorld) NewRoom(subId int) *roomutils.Room {
	room := &ThreeDiceRoom{
		helper:        cardutils.NewThreeDiceHelper(),
		robDealerList: list.New(),
	}
	room.Room = roomutils.NewRoom(subId, room)
	room.SetFreeDuration(18 * time.Second)

	room.syncTimer = util.NewPeriodTimer(room.Sync, "2017-11-28", 2*time.Second)
	// room.StartGame()
	return room.Room
}

func (w *ThreeDiceWorld) GetName() string {
	return "dice3"
}

func (w *ThreeDiceWorld) NewPlayer() *service.Player {
	p := &ThreeDicePlayer{}

	p.Player = service.NewPlayer(p)
	p.initGame()
	return p.Player
}

func GetPlayer(id int) *ThreeDicePlayer {
	if p := service.GetPlayer(id); p != nil {
		return p.GameAction.(*ThreeDicePlayer)
	}
	return nil
}
