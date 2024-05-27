package xiaojiu

// 小九
// Guogeer 2018-02-08

import (
	"gofishing-game/internal/errcode"
	"gofishing-game/internal/gameutils"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"

	"github.com/guogeer/quasar/log"
)

// 玩家信息
type XiaojiuUserInfo struct {
	service.UserInfo
	SeatIndex int
	Areas     [3]int64
}

type XiaojiuPlayer struct {
	*service.Player

	areas   [3]int64
	winGold int64
}

func (ply *XiaojiuPlayer) BeforeEnter() {
}

func (ply *XiaojiuPlayer) AfterEnter() {
}

func (ply *XiaojiuPlayer) TryEnter() errcode.Error {
	room := ply.Room()
	if room.Status != 0 || room.ExistTimes != 0 {
		return roomutils.ErrPlaying
	}
	return nil
}

func (ply *XiaojiuPlayer) TryLeave() errcode.Error {
	room := ply.Room()
	if room.IsTypeScore() && room.Status != 0 {
		return errcode.Retry
	}

	return nil
}

func (ply *XiaojiuPlayer) BeforeLeave() {
}

func (ply *XiaojiuPlayer) initGame() {
	for i := range ply.areas {
		ply.areas[i] = 0
	}
	ply.winGold = 0
}

func (ply *XiaojiuPlayer) GameOver() {
	ply.initGame()
}

func (ply *XiaojiuPlayer) Bet(area int, gold int64) {
	room := ply.Room()
	log.Debugf("player %d bet %d %d status %d", ply.Id, area, gold, room.Status)
	if ply == room.dealer {
		return
	}
	if gold < 0 || area < 0 || area >= len(ply.areas) {
		return
	}
	var sum int64
	for _, n := range ply.areas {
		sum += n
	}

	limit := room.betLimitPerUser() // 单人限注
	if sum+gold > limit {
		return
	}
	if room.Status != roomutils.RoomStatusPlaying {
		return
	}

	// OK
	ply.areas[area] += gold
	ply.AddGold(-gold, "xiaojiu_bet", service.WithNoItemLog())

	room.areas[area] += gold
	room.Broadcast("bet", gameutils.MergeError(nil, map[string]any{"uid": ply.Id, "areaId": area, "gold": gold}))
}

func (ply *XiaojiuPlayer) GetUserInfo(self bool) *XiaojiuUserInfo {
	info := &XiaojiuUserInfo{}
	info.UserInfo = ply.UserInfo
	info.SeatIndex = ply.GetSeatIndex()
	info.Areas = ply.areas
	return info
}

func (ply *XiaojiuPlayer) Room() *XiaojiuRoom {
	if room := roomutils.GetRoomObj(ply.Player).CustomRoom(); room != nil {
		return room.(*XiaojiuRoom)
	}
	return nil
}

func (ply *XiaojiuPlayer) GetSeatIndex() int {
	return roomutils.GetRoomObj(ply.Player).GetSeatIndex()
}
