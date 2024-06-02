package texas

import (
	"gofishing-game/games/migrate/internal/cardrule"
	"gofishing-game/internal/cardutils"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
	"time"
)

type TexasWorld struct{}

func init() {
	service.AddWorld(&TexasWorld{})

	var cards []int
	for color := 0; color < 4; color++ {
		for value := 2; value <= 14; value++ {
			c := (color << 4) | value
			cards = append(cards, c)
		}
	}

	cardutils.GetCardSystem().Init(cards)
}

func (w *TexasWorld) NewRoom(subId int) *roomutils.Room {
	helper := cardrule.NewTexasHelper()
	room := &TexasRoom{
		helper: helper,
		cards:  make([]int, 0, helper.Size()),

		tempDealerSeat: -1,
		dealerSeat:     -1,
		smallBlindSeat: -1,
		bigBlindSeat:   -1,
	}
	room.Room = roomutils.NewRoom(subId, room)
	room.AutoStart()
	room.SetFreeDuration(18 * time.Second)

	room.SetPlay(roomutils.OptAutoPlay)                    // 自动代打
	room.SetNoPlay(roomutils.OptForbidEnterAfterGameStart) // 游戏开始后禁止进入游戏
	return room.Room
}

func (w *TexasWorld) GetName() string {
	return "texas"
}

func (w *TexasWorld) NewPlayer() *service.Player {
	p := &TexasPlayer{}

	p.Player = service.NewPlayer(p)
	p.initGame()
	return p.Player
}
