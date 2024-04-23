package sangong

import (
	"gofishing-game/service"
	"third/cardutil"
	"time"
)

type SangongWorld struct{}

func init() {
	service.CreateWorld("三公", &SangongWorld{})

	var cards []int
	for color := 0; color < 4; color++ {
		for value := 2; value <= 14; value++ {
			c := (color << 4) | value
			cards = append(cards, c)
		}
	}

	cardutil.GetCardSystem().Init(cards)
}

func (w *SangongWorld) NewRoom(id, subId int) *service.Room {
	r := &SangongRoom{
		helper: cardutil.NewSangongHelper(),
	}
	r.Room = service.NewRoom(id, subId, r)
	r.SetRestartTime(18 * time.Second)

	r.SetPlay(OptFangzhudangzhuang)
	r.SetNoPlay(OptWuzhuang)
	r.SetNoPlay(OptZiyouqiangzhuang)

	r.SetPlay(OptChouma1)
	r.SetNoPlay(OptChouma2)
	r.SetNoPlay(OptChouma3)
	r.SetNoPlay(OptChouma5)
	r.SetNoPlay(OptChouma8)
	r.SetNoPlay(OptChouma10)
	r.SetNoPlay(OptChouma20)

	r.SetPlay(service.OptAutoPlay)                    // 自动代打
	r.SetNoPlay(service.OptForbidEnterAfterGameStart) // 游戏开始后禁止进入游戏
	return r.Room
}

func (w *SangongWorld) GetName() string {
	return "sg"
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
