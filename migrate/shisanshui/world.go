package shisanshui

import (
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
	"time"
)

type ShisanshuiWorld struct{}

func init() {
	service.CreateWorld("十三水", &ShisanshuiWorld{})

	var cards = []int{0xf0, 0xf1}
	for color := 0; color < 4; color++ {
		for value := 0x02; value <= 0x0e; value++ {
			c := (color << 4) | value
			cards = append(cards, c)
		}
	}

	cardutils.GetCardSystem().Init(cards)
}

func (w *ShisanshuiWorld) NewRoom(subId int) *roomutils.Room {
	room := &ShisanshuiRoom{
		helper: cardutils.NewShisanshuiHelper(),
	}
	room.Room = roomutils.NewRoom(subId, room)

	room.SetFreeDuration(60 * time.Second)
	room.SetNoPlay(OptDaxiaowang)

	room.SetPlay(OptXianghubipai)
	room.SetNoPlay(OptFangzhudangzhuang)

	room.SetNoPlay(OptLipai50s)
	room.SetNoPlay(OptLipai80s)
	room.SetNoPlay(OptLipai70s)

	room.SetNoPlay(service.OptZeroSeat + 4)
	room.SetNoPlay(service.OptZeroSeat + 5)
	return room.Room
}

func (w *ShisanshuiWorld) GetName() string {
	return "13shui"
}

func (w *ShisanshuiWorld) NewPlayer() *service.Player {
	p := &ShisanshuiPlayer{}
	p.cards = make([]int, 13)
	p.splitCards = make([]int, 13)

	p.Player = service.NewPlayer(p)
	p.initGame()
	return p.Player
}
