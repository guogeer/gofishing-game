package lottery

import (
	"gofishing-game/service"
	"third/cardutil"
	"time"

	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/utils"
)

type lotteryWorld struct{}

func init() {
	service.CreateWorld(&lotteryWorld{})

	var cards []int
	for color := 0; color < 4; color++ {
		for value := 2; value <= 14; value++ {
			c := (color << 4) | value
			cards = append(cards, c)
		}
	}

	cardutil.GetCardSystem().Init(cards)
}

func (w *lotteryWorld) NewRoom(id, subId int) *service.Room {
	helper := cardutil.NewZhajinhuaHelper()
	helper.SetOption("AAA")
	room := &lotteryRoom{
		helper: helper,
	}
	room.Room = service.NewRoom(id, subId, room)
	room.SetRestartTime(18 * time.Second)
	util.NewPeriodTimer(room.Sync, "2001-01-01", time.Second)

	// 代号时时乐捕鱼
	tags, _ := config.String("Room", subId, "Tags")
	if config.IsPart(tags, "fish") {
		room.isFishing = true
	}
	return room.Room
}

func (w *lotteryWorld) GetName() string {
	return "ssl"
}

func (w *lotteryWorld) NewPlayer() *service.Player {
	p := &lotteryPlayer{}

	p.Player = service.NewPlayer(p)
	p.initGame()
	return p.Player
}

func GetPlayer(id int) *lotteryPlayer {
	if p := service.GetPlayer(id); p != nil {
		return p.GameAction.(*lotteryPlayer)
	}
	return nil
}
