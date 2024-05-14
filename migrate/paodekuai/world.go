package paodekuai

import (
	"fmt"
	"gofishing-game/internal/cardutils"
	"gofishing-game/migrate/internal/cardrule"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
	"strings"

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

	cardutils.GetCardSystem().Init(cards)
}

func (w *PaodekuaiWorld) NewRoom(subId int) *roomutils.Room {
	r := &PaodekuaiRoom{
		helper: cardrule.NewPaodekuaiHelper(),
	}
	r.Room = roomutils.NewRoom(subId, r)
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

	r.SetPlay(fmt.Sprintf(roomutils.OptSeat, 3))
	r.SetNoPlay(fmt.Sprintf(roomutils.OptSeat, 2))

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
