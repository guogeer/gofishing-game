package xiaojiu

import (
	"gofishing-game/internal/cardutils"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
	"time"
)

func init() {
	service.CreateWorld(&XiaojiuWorld{})

	var cards []int
	for color := 0; color < 4; color++ {
		for value := 2; value <= 0xa; value++ {
			c := (color << 4) | value
			cards = append(cards, c)
		}
	}
	cards = append(cards, 0x0e, 0x1e, 0x2e, 0x3e)
	cardutils.GetCardSystem().Init(cards)
}

type XiaojiuWorld struct{}

func (w *XiaojiuWorld) NewRoom(subId int) *roomutils.Room {
	room := &XiaojiuRoom{}
	room.Room = roomutils.NewRoom(subId, room)
	room.SetFreeDuration(18 * time.Second)

	room.SetNoPlay(roomOptMingjiu) // 明九
	room.SetPlay(roomOptAnjiu)     // 暗九

	room.SetNoPlay(roomOptLunzhuang)   // 轮庄
	room.SetNoPlay(roomOptSuijizhuang) // 随机庄
	room.SetPlay(roomOptFangzhuzhuang) // 房主庄

	room.SetPlay(roomOptDanrenxianzhu10)   // 单人限注10
	room.SetNoPlay(roomOptDanrenxianzhu20) // 单人限注20
	room.SetNoPlay(roomOptDanrenxianzhu30) // 单人限注30
	room.SetNoPlay(roomOptDanrenxianzhu50) // 单人限注50

	room.SetNoPlay(roomOptZhuangjiabie10) // 蹩十
	room.SetNoPlay("必压_1")                // 必压1
	room.SetNoPlay("必压_2")                // 必压2
	room.SetNoPlay("必压_5")                // 必压5
	room.SetNoPlay("必压_10")               // 必压10
	room.SetNoPlay("必压_20")               // 必压20

	return room.Room
}

func (w *XiaojiuWorld) GetName() string {
	return "xiao9"
}

func (w *XiaojiuWorld) NewPlayer() *service.Player {
	p := &XiaojiuPlayer{}

	p.Player = service.NewPlayer(p)
	p.initGame()
	return p.Player
}

func GetPlayer(id int) *XiaojiuPlayer {
	if p := service.GetPlayer(id); p != nil {
		return p.GameAction.(*XiaojiuPlayer)
	}
	return nil
}
