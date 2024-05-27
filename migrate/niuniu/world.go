package niuniu

import (
	"gofishing-game/internal/cardutils"
	"gofishing-game/migrate/internal/cardrule"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
	"strings"
	"time"

	"github.com/guogeer/quasar/config"
)

type NiuNiuWorld struct{}

func init() {
	service.CreateWorld(&NiuNiuWorld{})

	var cards []int
	for color := 0; color < 4; color++ {
		for value := 2; value <= 14; value++ {
			c := (color << 4) | value
			cards = append(cards, c)
		}
	}

	cardutils.GetCardSystem().Init(cards)
}

func (w *NiuNiuWorld) NewRoom(subId int) *roomutils.Room {
	r := &NiuNiuRoom{
		helper: cardrule.NewNiuNiuHelper(),
	}
	r.Room = roomutils.NewRoom(subId, r)

	r.SetFreeDuration(8 * time.Second)

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

	r.SetPlay(roomutils.OptAutoPlay)                    // 自动代打
	r.SetNoPlay(roomutils.OptForbidEnterAfterGameStart) // 游戏开始后禁止进入游戏

	roomName, _ := config.String("room", subId, "roomName")
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
	return "niuniu"
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
