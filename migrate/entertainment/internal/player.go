package internal

import (
	"container/list"
	"fmt"
	"gofishing-game/internal/errcode"
	"gofishing-game/internal/gameutils"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"

	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
	"github.com/guogeer/quasar/util"
)

var errTooMuchBet = errcode.New("bet_too_much", "too much bet")
var errDealerMoreGold = errcode.New("dealer_more_gold", "dealer need more gold")

// 座位上的玩家
type userInfo struct {
	service.UserInfo
	SeatId          int   `json:"seatId,omitempty"`
	DealerLimitGold int64 `json:"dealerLimitGold,omitempty"`
	DealerGold      int64 `json:"dealerGold,omitempty"`
}

// 押注类游戏的玩家
type entertainmentPlayer struct {
	*service.Player

	winGold            int64
	winPrize           int64
	fakeGold           int64   // 不考虑扣税、奖池，得到的金币
	fakeAreas          []int64 // 不考虑扣税、奖池，得到的金币
	betAreas           []int64
	applyElement       *list.Element
	continuousBetTimes int   // 连续押注次数
	onceBet            bool  // 投注过一次
	dealerGold         int64 // 上庄时金币
	dealerLimitGold    int64 // 上庄时设置的金币
}

func (ply *entertainmentPlayer) TryEnter() errcode.Error {
	return nil
}

func (ply *entertainmentPlayer) BeforeEnter() {
}

func (ply *entertainmentPlayer) GetSeatId() int {
	roomObj := roomutils.GetRoomObj(ply.Player)
	return roomObj.GetSeatIndex()
}

func (ply *entertainmentPlayer) AfterEnter() {
	// 自动坐下
	room := ply.Room()
	if len(ply.betAreas) != room.userAreaNum {
		ply.betAreas = make([]int64, room.userAreaNum)
	}
	if ply.fakeAreas == nil {
		ply.fakeAreas = make([]int64, len(ply.betAreas))
	}
}

func (ply *entertainmentPlayer) GetUserInfo(otherId int) *userInfo {
	info := &userInfo{
		SeatId:          ply.GetSeatId(),
		DealerGold:      ply.dealerGold,
		DealerLimitGold: ply.dealerLimitGold,
	}
	simpleInfo := ply.UserInfo
	info.UserInfo = simpleInfo
	return info
}

func (ply *entertainmentPlayer) TryLeave() errcode.Error {
	// 游戏中
	room := ply.Room()
	if room.Status != 0 {
		// 庄家
		if room.dealer == ply {
			return errcode.New("playing_in_game", "playing in game")
		}
		// 玩家已下注
		if ply.totalBet() > 0 {
			return errcode.New("bet_already", "bet_already")
		}
	}
	return nil
}

func (ply *entertainmentPlayer) BeforeLeave() {
	ply.CancelDealer()
}

func (ply *entertainmentPlayer) totalBet() int64 {
	var sum int64
	for _, n := range ply.betAreas {
		sum += n
	}
	return sum
}

func (ply *entertainmentPlayer) Chat(t int, msg string) {
	uid := ply.Id
	nickname := ply.Nickname
	room := ply.Room()
	room.Broadcast("Chat", map[string]any{
		"UId":      uid,
		"Nickname": nickname,
		"Message":  msg,
	})
}

func (ply *entertainmentPlayer) Bet(area int, gold int64) {
	room := ply.Room()

	// 游戏未开始
	if room.Status != roomutils.RoomStatusPlaying {
		return
	}
	var err errcode.Error
	// 无效的数据
	if area < 0 || area >= len(ply.betAreas) || gold <= 0 {
		err = errcode.Retry
	}
	// 庄家不可投注
	if room.dealer == ply {
		err = errcode.Retry
	}
	if !ply.BagObj().IsEnough(gameutils.ItemIdGold, gold) {
		err = errcode.MoreItem(gameutils.ItemIdGold)
	}
	{
		total := ply.totalBet()
		percent, ok := config.Float("entertainment", room.SubId, "MaxBetPercent")
		if ok && total+gold > int64(percent*float64(total+ply.BagObj().NumItem(gameutils.ItemIdGold))/100) {
			err = errTooMuchBet
		}
		maxBet, _ := config.Int("entertainment", room.SubId, "MaxBetLimit")
		if maxBet > 0 && total+gold > maxBet {
			err = errTooMuchBet
		}
	}
	{
		sum := room.totalBet()
		percent, ok := config.Float("entertainment", room.SubId, "AllUserBetPercent")
		if ok == true && room.dealer != nil && float64(sum+gold) > float64(room.dealer.dealerGold)*percent/100 {
			err = errDealerMoreGold
		}
	}
	{
		// 最低押注金币要求
		minBetNeedGold, ok := config.Int("entertainment", room.SubId, "MinBetNeedGold")
		if ok == true && ply.BagObj().NumItem(gameutils.ItemIdGold) < minBetNeedGold {
			scale, _ := config.Int("web", "ExchangeScale", "Value")
			if scale < 1 {
				scale = 1
			}
			err = errcode.MoreItem(gameutils.ItemIdGold)
		}
	}

	data := map[string]any{
		"uid":  ply.Id,
		"area": area,
		"gold": gold,
	}

	betArgs := &struct {
		SubId  int    `json:"subId,omitempty"`
		Name   string `json:"name,omitempty"`
		Uid    int    `json:"uid,omitempty"`
		Gold   int64  `json:"gold,omitempty"`
		Area   int    `json:"area,omitempty"`
		Bet    int64  `json:"bet,omitempty"`
		BigBet int64  `json:"bigBet,omitempty"`
		Code   int    `json:"code,omitempty"`
		Msg    string `json:"msg,omitempty"`
	}{
		Name:  service.GetServerName(),
		Uid:   ply.Id,
		Gold:  ply.BagObj().NumItem(gameutils.ItemIdGold),
		Bet:   gold,
		Area:  area,
		SubId: ply.Room().SubId,
	}

	ply.WriteJSON("Bet", data)

	if !ply.IsRobot {
		log.Infof("player %d bet area %d gold %d", ply.Id, area, gold)
	}
	if err != nil {
		return
	}

	// OK
	ply.betAreas[area] += gold
	ply.AddGold(-gold, util.GUID(), "sum."+service.GetName())
	ply.RoomObj.BetGold += gold
	room.OnBet(area, gold)
	if ply.IsRobot == false {
		room.userBetAreas[area] += gold
	}

	// 玩家有座位
	if ply.SeatId != roomutils.NoSeat || betArgs.BigBet > 0 {
		room.Broadcast("Bet", data, ply.Id)
	}
	// 移除房间通知
	if false && ply.continuousBetTimes%21 == 20 && ply.onceBet == false {
		ply.onceBet = true
		msg := fmt.Sprintf("%s玩上瘾了", ply.Nickname)
		room.Broadcast("Broadcast", map[string]any{"Info": msg, "Message": msg})
	}
}

func (ply *entertainmentPlayer) GameOver() {
	ply.onceBet = false
	ply.winGold = 0
	for i := range ply.betAreas {
		ply.betAreas[i] = 0
	}
	ply.fakeGold = 0
	ply.winPrize = 0
	for i := range ply.fakeAreas {
		ply.fakeAreas[i] = 0
	}
}

func (ply *entertainmentPlayer) GetLastHistory(n int) {
	room := ply.Room()
	ply.WriteJSON("GetLastHistory", map[string]any{"Last": room.GetLast(n)})
}

// 申请当庄
func (ply *entertainmentPlayer) ApplyDealer() {
	var e errcode.Error

	room := ply.Room()
	// 玩家已申请或已当庄
	if room.dealer == ply {
		e = errcode.Retry
	}
	if ply.applyElement != nil {
		e = errcode.Retry
	}
	minDealerGold, _ := config.Int("entertainment", room.SubId, "MinDealerGold")
	if ply.BagObj().NumItem(gameutils.ItemIdGold) < minDealerGold {
		e = errcode.MoreItem(gameutils.ItemIdGold)
	}

	uid := ply.Id
	data := map[string]any{
		"UId": uid,
	}
	ply.WriteErr("applyDealer", e, data)
	if e != nil {
		return
	}
	ply.applyElement = room.dealerQueue.PushBack(ply)
	room.Broadcast("ApplyDealer", data, ply.Id)
}

func (ply *entertainmentPlayer) CancelDealer() {
	room := ply.Room()
	// 玩家未当庄或申请上庄
	if ply.applyElement == nil && ply != room.dealer {
		return
	}

	code := Ok
	limit, _ := config.Int("entertainment", room.SubId, "ForceCancelDealerGold")
	if false && room.dealer == ply && room.dealer.dealerGold < limit {
		code = DealerNeedMoreGold
	}

	// 已当庄，游戏中不可下庄
	room.delayCancelDealer = false
	if room.dealer == ply && room.Status != 0 {
		// 结算后自动下庄
		code = DelayCancelDealer
		room.delayCancelDealer = true
	}
	uid := ply.Id
	data := map[string]any{
		"Code":     code,
		"Msg":      code.String(),
		"UId":      uid,
		"IsDealer": (room.dealer == ply),
	}

	ply.WriteJSON("CancelDealer", data)
	if room.delayCancelDealer {
		return
	}

	ply.dealerGold = 0
	if ply.applyElement != nil {
		room.dealerQueue.Remove(ply.applyElement)
		ply.applyElement = nil
	}
	if room.dealer == ply {
		ply.RoomObj.IsVisible = false

		room.dealer = nil
		room.dealerLoop = 0
		room.delayCancelDealer = false
		room.Broadcast("CancelDealer", data, ply.Id)
	}
}

// 0 表示没有申请上庄
func (ply *entertainmentPlayer) dealerQueueRank() int {
	room := ply.Room()

	var counter int
	for e := room.dealerQueue.Front(); e != nil; e = e.Next() {
		counter++
		if e == ply.applyElement {
			return counter
		}
	}
	return 0
}

type DealerQueue struct {
	Top       []*userInfo
	Len       int
	Rank      int   `json:",omitempty"`
	LimitGold int64 `json:",omitempty"`
}

// 上庄列表
func (ply *entertainmentPlayer) dealerQueue() *DealerQueue {
	room := ply.Room()

	var dealers []*userInfo
	for e := room.dealerQueue.Front(); e != nil; e = e.Next() {
		p := e.Value.(*entertainmentPlayer)
		dealers = append(dealers, p.GetUserInfo(0))
	}
	q := &DealerQueue{
		Top:       dealers,
		Len:       room.dealerQueue.Len(),
		LimitGold: ply.dealerLimitGold,
	}
	if n := ply.dealerQueueRank(); n != 0 {
		q.Rank = n
	}
	return q
}

func (ply *entertainmentPlayer) Room() *entertainmentRoom {
	if room := ply.RoomObj.CustomRoom(); room != nil {
		return room.(*entertainmentRoom)
	}
	return nil
}

func (ply *entertainmentPlayer) OnAddItems(items []gameutils.Item, way string) {
	ply.Player.OnAddItems(items, way)
	for _, item := range items {
		if item.Id == gameutil.ItemIdGold {
			ply.dealerGold += item.Num
		}
	}
}
