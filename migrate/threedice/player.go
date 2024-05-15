package threedice

// 夺宝王
// Guogeer 2017-11-09

import (
	"container/list"
	"gofishing-game/internal/errcode"
	"gofishing-game/internal/gameutils"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
	"math/rand"
	"time"

	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
	"github.com/guogeer/quasar/util"
)

// 玩家信息
type ThreeDiceUserInfo struct {
	service.UserInfo
	SeatId        int
	Areas         []int64
	AreaId        int
	RobDealerGold int64 `json:",omitempty"`
	IsRobDealer   bool  `json:",omitempty"`
}

type ThreeDicePlayer struct {
	*service.Player

	robDealerGold int64

	areas            [2]int64
	areaId           int
	robDealerElement *list.Element
	winGold          int64
}

func (ply *ThreeDicePlayer) TryEnter() errcode.Error {
	lastClock := "00:00"
	clock := time.Now().Format("15:04")
	for i := 0; i < config.Row("threedicerobot"); i++ {
		rowId := config.RowId(i)
		clock1, _ := config.String("threedicerobot", rowId, "Clock")
		if clock < clock1 {
			break
		}
		lastClock = clock1
	}

	minOnline, _ := config.Int("threedicerobot", lastClock, "MinOnline")
	maxOnline, _ := config.Int("threedicerobot", lastClock, "MaxOnline")
	if room := ply.Room(); room != nil && ply.IsRobot() {
		n := int(maxOnline) - len(room.AllPlayers)
		if (n <= 0 || rand.Intn(n) == 0) && int(minOnline) < len(room.AllPlayers) {
			return errcode.Retry
		}
	}
	return nil
}

func (ply *ThreeDicePlayer) TryLeave() errcode.Error {
	room := ply.Room()
	if ply.areaId != -1 {
		return errcode.Retry
	}
	if ply == room.dealer {
		return errcode.Retry
	}
	return nil
}

func (ply *ThreeDicePlayer) BeforeLeave() {
	room := ply.Room()
	if e := ply.robDealerElement; e != nil {
		room.robDealerList.Remove(e)
		ply.robDealerElement = nil
	}
}

func (ply *ThreeDicePlayer) initGame() {
	for i := 0; i < len(ply.areas); i++ {
		ply.areas[i] = 0
	}
	ply.areaId = -1
	ply.robDealerElement = nil
	ply.winGold = 0
}

func (ply *ThreeDicePlayer) GameOver() {
	ply.initGame()
}

func (ply *ThreeDicePlayer) RobDealer(gold int64) {
	room := ply.Room()
	log.Debugf("player %d rob dealer gold %d", ply.Id, gold)

	code := Ok
	if gold > ply.BagObj().NumItem(gameutils.ItemIdGold) {
		code = MoreGold
	}
	if room.Status != service.RoomStatusRobDealer {
		code = errcode.Retry
	}
	minDealer, maxDealer := room.dealerRequiredGold()
	if gold < minDealer {
		code = MoreGold
	}
	if gold > maxDealer {
		code = TooMuchGold
	}
	if e := ply.robDealerElement; e != nil {
		code = errcode.Retry
	}
	ply.WriteJSON("RobDealer", map[string]any{"Code": code, "Msg": code.String(), "UId": ply.Id, "Gold": gold})
	if code == Ok {
		ply.robDealerGold = gold
		ply.robDealerElement = room.robDealerList.PushBack(ply.Id)
	}
}

func (ply *ThreeDicePlayer) Bet(area int, gold int64) {
	room := ply.Room()
	log.Debugf("player %d bet %d %d status %d", ply.Id, area, gold, room.Status)
	if ply.areaId != -1 && area != ply.areaId {
		return
	}
	if ply == room.dealer {
		return
	}
	if gold < 0 || ply.BagObj().NumItem(gameutils.ItemIdGold) < gold || area < 0 || area >= len(ply.areas) {
		return
	}
	if room.Status != roomutils.RoomStatusPlaying {
		return
	}
	if gold > room.limit[area] {
		return
	}

	// OK
	ply.areaId = area
	ply.areas[area] += gold
	ply.AddGold(-gold, util.GUID(), "user.threedice_bet")

	room.areas[area] += gold
	room.limit[0] = room.dealerGold + room.areas[1] - room.areas[0]
	room.limit[1] = room.dealerGold + room.areas[0] - room.areas[1]
	room.Broadcast("Bet", map[string]any{"UId": ply.Id, "AreaId": area, "Gold": gold, "Areas": room.areas, "Limit": room.limit})
}

func (ply *ThreeDicePlayer) GetUserInfo(self bool) *ThreeDiceUserInfo {
	info := &ThreeDiceUserInfo{}
	info.UserInfo = ply.UserInfo
	info.SeatIndex = ply.GetSeatIndex()
	info.Areas = ply.areas[:]
	info.AreaId = ply.areaId
	if self {
		info.RobDealerGold = ply.robDealerGold
	}

	room := ply.Room()
	if room.Status == service.RoomStatusRobDealer && ply.robDealerElement != nil && self {
		info.IsRobDealer = true
	}
	return info
}

// 坐下
func (ply *ThreeDicePlayer) SitDown(seatId int) {
	room := ply.Room()

	sitDownRequiredGold, _ := config.Int("threedice", "SitDownRequiredGold", "Value")
	if ply.BagObj().NumItem(gameutils.ItemIdGold) < sitDownRequiredGold {
		return
	}
	if code := roomutils.GetRoomObj(ply.Player).SitDown(seatId); code != Ok {
		return
	}
	// OK
	info := ply.GetUserInfo(false)
	room.Broadcast("SitDown", map[string]any{"Code": Ok, "Info": info})
}

func (ply *ThreeDicePlayer) Room() *ThreeDiceRoom {
	if room := roomutils.GetRoomObj(ply.Player).CustomRoom(); room != nil {
		return room.(*ThreeDiceRoom)
	}
	return nil
}
