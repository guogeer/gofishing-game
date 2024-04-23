package paohuzi

import (
	"gofishing-game/service"
	"third/cardutil"
)

type PaohuziWorld struct{}

func init() {
	service.CreateWorld("跑胡子", &PaohuziWorld{})

	cards := make([]int, 0, 128)
	for value := 0x01; value <= 0x0a; value++ {
		// 一二三四五六七八九十
		c := value | 0x00
		cards = append(cards, c, c, c, c)
		// 壹贰叁肆伍陆柒捌玖拾
		c = value | 0x10
		cards = append(cards, c, c, c, c)
	}
	cardutil.GetCardSystem().Init(cards)
}

func (w *PaohuziWorld) NewRoom(id, subId int) *service.Room {
	r := &PaohuziRoom{
		helper: cardutil.NewPaohuziHelper(),

		expectChowPlayers: make(map[int]*PaohuziPlayer),
		expectWinPlayers:  make(map[int]*PaohuziPlayer),
	}
	r.Room = service.NewRoom(id, subId, r)
	r.SetPlay(service.OptZeroSeat + 3)
	r.SetNoPlay(service.OptZeroSeat + 2)

	return r.Room
}

func (w *PaohuziWorld) GetName() string {
	return "phz"
}

func (w *PaohuziWorld) NewPlayer() *service.Player {
	p := &PaohuziPlayer{}
	p.cards = make([]int, 1024)

	p.Player = service.NewPlayer(p)
	p.initGame()
	return p.Player
}

func GetPlayer(id int) *PaohuziPlayer {
	if p := service.GetPlayer(id); p != nil {
		return p.GameAction.(*PaohuziPlayer)
	}
	return nil
}
