package texas

import (
	"gofishing-game/service"
	"third/cardutil"
	"time"
)

type TexasWorld struct{}

func init() {
	service.CreateWorld("德州扑克", &TexasWorld{})

	var cards []int
	for color := 0; color < 4; color++ {
		for value := 2; value <= 14; value++ {
			c := (color << 4) | value
			cards = append(cards, c)
		}
	}

	cardutil.GetCardSystem().Init(cards)
}

func (w *TexasWorld) NewRoom(id, subId int) *service.Room {
	helper := cardutil.NewTexasHelper()
	room := &TexasRoom{
		helper: helper,
		cards:  make([]int, 0, helper.Size()),

		tempDealerSeat: -1,
		dealerSeat:     -1,
		smallBlindSeat: -1,
		bigBlindSeat:   -1,
	}
	room.Room = service.NewRoom(id, subId, room)
	room.AutoStart()
	room.SetRestartTime(18 * time.Second)

	room.SetPlay(service.OptAutoPlay)                    // 自动代打
	room.SetNoPlay(service.OptForbidEnterAfterGameStart) // 游戏开始后禁止进入游戏
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
