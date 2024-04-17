package paodekuai

import (
	"gofishing-game/service"
	"strings"
	"third/cardutil"

	"github.com/guogeer/quasar/config"
)

type PaodekuaiWorld struct{}

func init() {
	service.CreateWorld(&PaodekuaiWorld{})

	var cards = []int{0xf0, 0xf1}
	for color := 0; color < 4; color++ {
		for value := 0x02; value <= 0x0e; value++ {
			c := (color << 4) | value
			cards = append(cards, c)
		}
	}

	cardutil.GetCardSystem().Init(cards)
}

func (w *PaodekuaiWorld) NewRoom(id, subId int) *service.Room {
	r := &PaodekuaiRoom{
		helper: cardutil.NewPaodekuaiHelper(),
	}
	r.Room = service.NewRoom(id, subId, r)
	r.SetPlay(OptCard16)
	r.SetNoPlay(OptCard15)

	r.SetPlay(OptXianshipai)
	r.SetNoPlay(OptBuxianshipai)
	// r.SetNoPlay(OptZhuangJiaXianChu)
	// r.SetPlay(OptHeiTaoSanXianChu)
	// r.SetPlay(OptHeiTaoSanBiChu)
	r.SetPlay(OptBixuguan)
	r.SetNoPlay(OptKebuguan)
	r.SetNoPlay(OptHongtaoshizhaniao)
	r.SetPlay(OptShoulunheitaosanxianchu)
	r.SetNoPlay(OptSidaisan)

	roomName, _ := config.String("Room", subId, "RoomName")
	if strings.Contains(roomName, "郑州") {
		r.SetMainPlay(OptSandaidui)
		r.SetNoPlay(OptMeilunheitaosanbichu)
	}

	r.SetPlay(service.OptZeroSeat + 3)
	r.SetNoPlay(service.OptZeroSeat + 2)

	return r.Room
}

func (w *PaodekuaiWorld) GetName() string {
	return "pdk"
}

func (w *PaodekuaiWorld) NewPlayer() *service.Player {
	p := &PaodekuaiPlayer{}
	p.cards = make([]int, 1024)

	p.Player = service.NewPlayer(p)
	p.initGame()
	return p.Player
}

func GetPlayer(id int) *PaodekuaiPlayer {
	if p := service.GetPlayer(id); p != nil {
		return p.GameAction.(*PaodekuaiPlayer)
	}
	return nil
}
