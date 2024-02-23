package internal

import (
	ddzutils "gofishing-game/demo/doudizhu/utils"
	"gofishing-game/internal/cardutils"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
)

type DoudizhuWorld struct{}

func init() {
	service.CreateWorld(&DoudizhuWorld{})

	var cards = []int{0xf0, 0xf1}
	for color := 0; color < 4; color++ {
		for value := 0x02; value <= 0x0e; value++ {
			c := (color << 4) | value
			cards = append(cards, c)
		}
	}

	cardutils.GetCardSystem().Init(cards)
}

func (w *DoudizhuWorld) NewRoom(subId int) *roomutils.Room {
	r := &DoudizhuRoom{
		helper:       ddzutils.NewDoudizhuHelper(),
		currentTimes: 1,
	}
	r.Room = roomutils.NewRoom(subId, r)
	r.SetPlay(OptZhadan3)
	r.SetNoPlay(OptZhadan4)
	r.SetNoPlay(OptZhadan5)

	r.SetPlay(OptQiangdizhu)
	r.SetNoPlay(OptJiaodizhu)
	r.SetNoPlay(OptJiaofen)

	return r.Room
}

func (w *DoudizhuWorld) GetName() string {
	return "ddz"
}

func (w *DoudizhuWorld) NewPlayer() *service.Player {
	p := &DoudizhuPlayer{}
	p.cards = make([]int, 1024)

	p.Player = service.NewPlayer(p)
	p.initGame()
	return p.Player
}
