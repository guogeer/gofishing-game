package sangong

import (
	"gofishing-game/internal/cardutils"
	"gofishing-game/migrate/internal/cardrule"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
	"time"
)

type SangongWorld struct{}

func init() {
	service.CreateWorld(&SangongWorld{})

	var cards []int
	for color := 0; color < 4; color++ {
		for value := 2; value <= 14; value++ {
			c := (color << 4) | value
			cards = append(cards, c)
		}
	}

	cardutils.GetCardSystem().Init(cards)
}

func (w *SangongWorld) NewRoom(subId int) *roomutils.Room {
	r := &SangongRoom{
		helper: cardrule.NewSangongHelper(),
	}
	r.Room = roomutils.NewRoom(subId, r)
	r.SetFreeDuration(18 * time.Second)

	r.SetPlay(OptFangzhudangzhuang)
	r.SetNoPlay(OptWuzhuang)
	r.SetNoPlay(OptZiyouqiangzhuang)

	r.SetPlay(OptChouma, 1)
	r.SetNoPlay(OptChouma, 2)
	r.SetNoPlay(OptChouma, 3)
	r.SetNoPlay(OptChouma, 5)
	r.SetNoPlay(OptChouma, 8)
	r.SetNoPlay(OptChouma, 10)
	r.SetNoPlay(OptChouma, 20)

	r.SetPlay(roomutils.OptAutoPlay)                    // 自动代打
	r.SetNoPlay(roomutils.OptForbidEnterAfterGameStart) // 游戏开始后禁止进入游戏
	return r.Room
}

func (w *SangongWorld) GetName() string {
	return "sangong"
}

func (w *SangongWorld) NewPlayer() *service.Player {
	p := &SangongPlayer{}
	p.cards = make([]int, 3)

	p.Player = service.NewPlayer(p)
	p.initGame()
	return p.Player
}

func GetPlayer(id int) *SangongPlayer {
	if p := service.GetPlayer(id); p != nil {
		return p.GameAction.(*SangongPlayer)
	}
	return nil
}
