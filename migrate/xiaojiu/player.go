package xiaojiu

// 小九
// Guogeer 2018-02-08

import (
	"gofishing-game/service"

	"github.com/guogeer/quasar/log"
	"github.com/guogeer/quasar/util"
)

// 玩家信息
type XiaojiuUserInfo struct {
	service.UserInfo
	SeatId int
	Areas  [3]int64
}

type XiaojiuPlayer struct {
	*service.Player

	areas   [3]int64
	winGold int64
}

func (ply *XiaojiuPlayer) TryEnter() ErrCode {
	room := ply.Room()
	if room.Status != service.RoomStatusFree || room.ExistTimes != 0 {
		return PlayingInGame
	}
	return Ok
}

func (ply *XiaojiuPlayer) TryLeave() ErrCode {
	room := ply.Room()
	if room.IsUserCreate() && room.Status != service.RoomStatusFree {
		return Retry
	}

	return Ok
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
	if room.Status != service.RoomStatusPlaying {
		return
	}

	// OK
	ply.areas[area] += gold
	ply.AddGold(-gold, util.GUID(), "sum.xiaojiu_bet")

	room.areas[area] += gold
	room.Broadcast("Bet", map[string]any{"Code": Ok, "UId": ply.Id, "AreaId": area, "Gold": gold})
}

func (ply *XiaojiuPlayer) GetUserInfo(self bool) *XiaojiuUserInfo {
	info := &XiaojiuUserInfo{}
	info.UserInfo = ply.UserInfo
	info.SeatId = ply.SeatId
	info.Areas = ply.areas
	return info
}

func (ply *XiaojiuPlayer) Room() *XiaojiuRoom {
	if room := ply.RoomObj.CardRoom(); room != nil {
		return room.(*XiaojiuRoom)
	}
	return nil
}
