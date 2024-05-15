package lottery

// 2018-12-07

import (
	"gofishing-game/internal/errcode"
	"gofishing-game/internal/gameutils"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
	"third/cardutil"

	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
	"github.com/guogeer/quasar/util"
)

// 玩家信息
type lotteryUserInfo struct {
	service.UserInfo
	SeatId    int
	BetAreas  []int64
	BetAreas2 []int64 // 增加散牌押注区域
}

type lotteryPlayer struct {
	*service.Player
	areas   [cardutil.ZhajinhuaTypeAll]int64
	winGold int64
}

func (ply *lotteryPlayer) AfterEnter() {
	room := ply.Room()
	seatId := room.GetEmptySeat()
	if seatId != roomutils.NoSeat && ply.GetSeatIndex() == roomutils.NoSeat {
		ply.SitDown(seatId)
	}
}

func (ply *lotteryPlayer) TryLeave() errcode.Error {
	for _, bet := range ply.areas {
		if bet > 0 {
			return AlreadyBet
		}
	}
	return nil
}

func (ply *lotteryPlayer) BeforeLeave() {
}

func (ply *lotteryPlayer) initGame() {
	for i := range ply.areas {
		ply.areas[i] = 0
	}
	ply.winGold = 0
}

func (ply *lotteryPlayer) GameOver() {
	ply.initGame()
}

func (ply *lotteryPlayer) Bet(clientArea int, gold int64) {
	room := ply.Room()
	subId := room.SubId
	if room.Status != roomutils.RoomStatusPlaying {
		return
	}
	log.Infof("player %d bet area %d gold %d", ply.Id, clientArea, gold)

	code := Ok
	if ply.BagObj().NumItem(gameutils.ItemIdGold) < gold || gold <= 0 {
		code = MoreGold
	}
	area := clientArea
	if area <= 0 || area >= len(ply.areas) {
		code = errcode.Retry
	}
	if gold%room.Unit() != 0 {
		code = errcode.Retry
	}

	chips := room.chips
	if len(chips) > 0 && ply.areas[area]+gold > chips[len(chips)-1] {
		code = TooMuchBet
	}
	err := NewError(code)
	// 最低押注金币要求
	minBetNeedGold, _ := config.Int("lottery", room.SubId, "MinBetNeedGold")
	if ply.BagObj().NumItem(gameutils.ItemIdGold) < minBetNeedGold {
		err = NewError(MoreBetGold, minBetNeedGold)
	}

	response := map[string]any{
		"Code":  err.Code,
		"Msg":   err.Msg,
		"UId":   ply.Id,
		"Area":  area - 2,
		"Area2": area,
		"Gold":  gold,
	}

	ply.WriteJSON("Bet", response)
	if err.Code != Ok {
		return
	}

	ply.areas[area] += gold
	room.areas[area] += gold
	if ply.IsRobot {
		room.robotAreas[area] += gold
	}

	way := service.ItemWay{Way: "sum.lottery_bet", SubId: subId}.String()
	ply.AddGold(-gold, util.GUID(), way)
	if ply.GetSeatIndex() != roomutils.NoSeat {
		room.Broadcast("Bet", response, ply.Id)
	}
}

func (ply *lotteryPlayer) GetUserInfo(otherId int) *lotteryUserInfo {
	info := &lotteryUserInfo{}
	info.UserInfo = ply.GetInfo(otherId)
	info.SeatId = ply.GetSeatIndex()
	info.BetAreas = ply.areas[2:]
	info.BetAreas2 = ply.areas[:]
	return info
}

func (ply *lotteryPlayer) Room() *lotteryRoom {
	if room := roomutils.GetRoomObj(ply.Player).CustomRoom(); room != nil {
		return room.(*lotteryRoom)
	}
	return nil
}

func (ply *lotteryPlayer) SitDown(seatId int) {
	room := ply.Room()

	code := roomutils.GetRoomObj(ply.Player).TrySitDown(seatId)
	info := ply.GetUserInfo(ply.Id)
	data := map[string]any{
		"Code": code,
		"Msg":  code.String(),
		"Info": &info,
	}
	ply.WriteJSON("SitDown", data)
	if code != Ok {
		return
	}
	info = ply.GetUserInfo(0)
	room.Broadcast("SitDown", data, ply.Id)
}
