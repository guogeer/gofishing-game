package shengsidu

import (
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
)

type ShengsiduWorld struct{}

func init() {
	service.CreateWorld("生死堵", &ShengsiduWorld{})

	var cards = []int{}
	for color := 0; color < 4; color++ {
		for value := 0x02; value <= 0x0e; value++ {
			c := (color << 4) | value
			cards = append(cards, c)
		}
	}

	cardutils.GetCardSystem().Init(cards)
}

func (w *ShengsiduWorld) NewRoom(subId int) *roomutils.Room {
	r := &ShengsiduRoom{
		helper: cardutils.NewShengsiduHelper(),
	}
	r.Room = roomutils.NewRoom(subId, r)

	r.SetPlay(OptXianshipai)
	r.SetNoPlay(OptBuxianshipai)

	r.SetPlay(OptMeilunfangpian3xianchu)
	r.SetNoPlay(OptMeilunyingjiaxianchu)

	r.SetPlay(service.OptZeroSeat + 4)
	r.SetNoPlay(service.OptZeroSeat + 3)
	r.SetNoPlay(service.OptZeroSeat + 2)

	r.SetMainPlay(OptBixuguan)
	return r.Room
}

func (w *ShengsiduWorld) GetName() string {
	return "shengsidu"
}

func (w *ShengsiduWorld) NewPlayer() *service.Player {
	p := &ShengsiduPlayer{
		addition: make(map[string]int),
	}
	p.cards = make([]int, 1024)

	p.Player = service.NewPlayer(p)
	p.initGame()
	return p.Player
}

func GetPlayer(id int) *ShengsiduPlayer {
	if p := service.GetPlayer(id); p != nil {
		return p.GameAction.(*ShengsiduPlayer)
	}
	return nil
}
