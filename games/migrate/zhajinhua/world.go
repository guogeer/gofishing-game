package zhajinhua

import (
	"gofishing-game/games/migrate/internal/cardrule"
	"gofishing-game/internal/cardutils"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
	"time"
)

type ZhajinhuaWorld struct{}

func init() {
	service.AddWorld(&ZhajinhuaWorld{})

	var cards []int
	for color := 0; color < 4; color++ {
		for value := 2; value <= 14; value++ {
			c := (color << 4) | value
			cards = append(cards, c)
		}
	}

	cardutils.GetCardSystem().Init(cards)
}

func (w *ZhajinhuaWorld) NewRoom(subId int) *roomutils.Room {
	helper := cardrule.NewZhajinhuaHelper()
	room := &ZhajinhuaRoom{
		helper: helper,

		dealerSeatIndex: -1,
	}
	room.Room = roomutils.NewRoom(subId, room)
	room.AutoStart()
	room.SetFreeDuration(8 * time.Second)

	room.SetPlay(OptLunshu10)
	room.SetNoPlay(OptLunshu20)

	room.SetNoPlay(OptMengpailunshu1)
	room.SetNoPlay(OptMengpailunshu2)
	room.SetNoPlay(OptMengpailunshu3)

	room.SetPlay(roomutils.OptAutoPlay)                    // 自动代打
	room.SetNoPlay(roomutils.OptForbidEnterAfterGameStart) // 游戏开始后禁止进入游戏
	return room.Room
}

func (w *ZhajinhuaWorld) GetName() string {
	return "zjh"
}

func (w *ZhajinhuaWorld) NewPlayer() *service.Player {
	p := &ZhajinhuaPlayer{}

	p.Player = service.NewPlayer(p)
	p.initGame()
	return p.Player
}

func GetPlayer(id int) *ZhajinhuaPlayer {
	if p := service.GetPlayer(id); p != nil {
		return p.GameAction.(*ZhajinhuaPlayer)
	}
	return nil
}
