package niuniu

import (
	"gofishing-game/service"
	"strings"
	"third/cardutil"
	"time"

	"github.com/guogeer/quasar/config"
)

type NiuNiuWorld struct{}

func init() {
	service.CreateWorld("牛牛", &NiuNiuWorld{})

	var cards []int
	for color := 0; color < 4; color++ {
		for value := 2; value <= 14; value++ {
			c := (color << 4) | value
			cards = append(cards, c)
		}
	}

	cardutil.GetCardSystem().Init(cards)
}

func (w *NiuNiuWorld) NewRoom(id, subId int) *service.Room {
	r := &NiuNiuRoom{
		helper: cardutil.NewNiuNiuHelper(),
	}
	r.Room = service.NewRoom(id, subId, r)

	r.SetRestartTime(8 * time.Second)

	r.SetPlay(OptNiuNiuShangZhuang)
	r.SetNoPlay(OptGuDingShangZhuang)
	r.SetNoPlay(OptZiYouShangZhuang)
	r.SetNoPlay(OptMingPaiShangZhuang)
	r.SetNoPlay(OptTongBiNiuNiu)

	r.SetNoPlay(OptWuXiaoNiu)
	r.SetNoPlay(OptZhaDanNiu)
	r.SetNoPlay(OptWuHuaNiu)

	r.SetPlay(OptFanBeiGuiZe1)
	r.SetNoPlay(OptFanBeiGuiZe2)

	r.SetNoPlay(OptDiZhu1_2)
	r.SetNoPlay(OptDiZhu2_4)
	r.SetNoPlay(OptDiZhu4_8)
	r.SetPlay(OptDiZhu1_2_3_4_5)

	r.SetPlay(OptDiZhu1)
	r.SetNoPlay(OptDiZhu2)
	r.SetNoPlay(OptDiZhu4)

	r.SetPlay(OptZuiDaQiangZhuang1)
	r.SetNoPlay(OptZuiDaQiangZhuang2)
	r.SetNoPlay(OptZuiDaQiangZhuang3)
	r.SetNoPlay(OptZuiDaQiangZhuang4)

	r.SetPlay(service.OptAutoPlay)                    // 自动代打
	r.SetNoPlay(service.OptForbidEnterAfterGameStart) // 游戏开始后禁止进入游戏

	roomName, _ := config.String("Room", subId, "RoomName")
	if strings.Contains(roomName, "耒阳") {
		r.SetNoPlay(OptSiHuaNiu)
		r.SetMainPlay(OptFanBeiGuiZe3)
	}
	// 明牌场
	if strings.Contains(roomName, "明牌") {
		r.SetMainPlay(OptMingPaiShangZhuang)
	}

	return r.Room
}

func (w *NiuNiuWorld) GetName() string {
	return "dmnn"
}

func (w *NiuNiuWorld) NewPlayer() *service.Player {
	p := &NiuNiuPlayer{}
	p.cards = make([]int, 5)
	p.expectCards = make([]int, 5)
	p.doneCards = make([]int, 5)

	p.Player = service.NewPlayer(p)
	p.initGame()
	return p.Player
}

func GetPlayer(id int) *NiuNiuPlayer {
	if p := service.GetPlayer(id); p != nil {
		return p.GameAction.(*NiuNiuPlayer)
	}
	return nil
}
