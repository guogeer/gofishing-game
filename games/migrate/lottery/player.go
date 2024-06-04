package lottery

import (
	"container/list"
	"fmt"
	"gofishing-game/internal/errcode"
	"gofishing-game/internal/gameutils"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"

	"github.com/guogeer/quasar/v2/config"
	"github.com/guogeer/quasar/v2/log"
)

var (
	errTooMuchBet         = errcode.New("bet_too_much", "too much bet")
	errDealerNeedMoreGold = errcode.New("dealer_need_more_gold", "dealer need more gold")
	errDelayCancelDealer  = errcode.New("delay_cancel_dealer", "delay cancel dealer")
)

// 座位上的玩家
type userInfo struct {
	service.UserInfo
	SeatIndex       int   `json:"seatIndex,omitempty"`
	DealerLimitGold int64 `json:"dealerLimitGold,omitempty"`
	DealerGold      int64 `json:"dealerGold,omitempty"`
}

// 押注类游戏的玩家
type lotteryPlayer struct {
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

func (ply *lotteryPlayer) TryEnter() errcode.Error {
	return nil
}

func (ply *lotteryPlayer) BeforeEnter() {
}

func (ply *lotteryPlayer) GetSeatIndex() int {
	roomObj := roomutils.GetRoomObj(ply.Player)
	return roomObj.GetSeatIndex()
}

func (ply *lotteryPlayer) AfterEnter() {
	// 自动坐下
	room := ply.Room()
	if len(ply.betAreas) != room.userAreaNum {
		ply.betAreas = make([]int64, room.userAreaNum)
	}
	if ply.fakeAreas == nil {
		ply.fakeAreas = make([]int64, len(ply.betAreas))
	}
}

func (ply *lotteryPlayer) GetUserInfo(otherId int) *userInfo {
	info := &userInfo{
		SeatIndex:       ply.GetSeatIndex(),
		DealerGold:      ply.dealerGold,
		DealerLimitGold: ply.dealerLimitGold,
	}
	simpleInfo := ply.UserInfo
	info.UserInfo = simpleInfo
	return info
}

func (ply *lotteryPlayer) TryLeave() errcode.Error {
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

func (ply *lotteryPlayer) BeforeLeave() {
	ply.CancelDealer()
}

func (ply *lotteryPlayer) totalBet() int64 {
	var sum int64
	for _, n := range ply.betAreas {
		sum += n
	}
	return sum
}

func (ply *lotteryPlayer) Chat(t int, msg string) {
	uid := ply.Id
	nickname := ply.Nickname
	room := ply.Room()
	room.Broadcast("chat", map[string]any{
		"uid":      uid,
		"nickname": nickname,
		"message":  msg,
	})
}

func (ply *lotteryPlayer) Bet(area int, gold int64) {
	room := ply.Room()

	// 游戏未开始
	if room.Status != roomutils.RoomStatusPlaying {
		return
	}
	var e errcode.Error
	// 无效的数据
	if area < 0 || area >= len(ply.betAreas) || gold <= 0 {
		e = errcode.Retry
	}
	// 庄家不可投注
	if room.dealer == ply {
		e = errcode.Retry
	}
	if !ply.BagObj().IsEnough(gameutils.ItemIdGold, gold) {
		e = errcode.MoreItem(gameutils.ItemIdGold)
	}
	{
		total := ply.totalBet()
		percent, ok := config.Float("lottery", room.SubId, "maxBetPercent")
		if ok && total+gold > int64(percent*float64(total+ply.BagObj().NumItem(gameutils.ItemIdGold))/100) {
			e = errTooMuchBet
		}
		maxBet, _ := config.Int("lottery", room.SubId, "maxBetLimit")
		if maxBet > 0 && total+gold > maxBet {
			e = errTooMuchBet
		}
	}
	{
		sum := room.totalBet()
		percent, ok := config.Float("lottery", room.SubId, "allUserBetPercent")
		if ok && room.dealer != nil && float64(sum+gold) > float64(room.dealer.dealerGold)*percent/100 {
			e = errDealerNeedMoreGold
		}
	}
	{
		// 最低押注金币要求
		minBetNeedGold, ok := config.Int("lottery", room.SubId, "minBetNeedGold")
		if ok && ply.BagObj().NumItem(gameutils.ItemIdGold) < minBetNeedGold {
			scale, _ := config.Int("web", "exchangeScale", "value")
			if scale < 1 {
				scale = 1
			}
			e = errcode.MoreItem(gameutils.ItemIdGold)
		}
	}

	data := map[string]any{
		"uid":  ply.Id,
		"area": area,
		"gold": gold,
	}

	ply.WriteErr("bet", e, data)

	if !ply.IsRobot {
		log.Infof("player %d bet area %d gold %d", ply.Id, area, gold)
	}
	if e != nil {
		return
	}

	// OK
	ply.betAreas[area] += gold
	ply.AddGold(-gold, roomutils.GetServerName(room.SubId))
	room.OnBet(area, gold)
	if !ply.IsRobot {
		room.userBetAreas[area] += gold
	}

	// 玩家有座位
	if ply.GetSeatIndex() != roomutils.NoSeat {
		room.Broadcast("bet", data, ply.Id)
	}
	// 移除房间通知
	if false && ply.continuousBetTimes%21 == 20 && !ply.onceBet {
		ply.onceBet = true
		msg := fmt.Sprintf("%s玩上瘾了", ply.Nickname)
		room.Broadcast("droadcast", map[string]any{"info": msg, "message": msg})
	}
}

func (ply *lotteryPlayer) GameOver() {
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

func (ply *lotteryPlayer) GetLastHistory(n int) {
	room := ply.Room()
	ply.WriteJSON("getLastHistory", map[string]any{"last": room.GetLast(n)})
}

// 申请当庄
func (ply *lotteryPlayer) ApplyDealer() {
	var e errcode.Error

	room := ply.Room()
	// 玩家已申请或已当庄
	if room.dealer == ply {
		e = errcode.Retry
	}
	if ply.applyElement != nil {
		e = errcode.Retry
	}
	minDealerGold, _ := config.Int("lottery", room.SubId, "minDealerGold")
	if ply.BagObj().NumItem(gameutils.ItemIdGold) < minDealerGold {
		e = errcode.MoreItem(gameutils.ItemIdGold)
	}

	uid := ply.Id
	data := map[string]any{
		"uid": uid,
	}
	ply.WriteErr("applyDealer", e, data)
	if e != nil {
		return
	}
	ply.applyElement = room.dealerQueue.PushBack(ply)
	room.Broadcast("applyDealer", data, ply.Id)
}

func (ply *lotteryPlayer) CancelDealer() {
	room := ply.Room()
	// 玩家未当庄或申请上庄
	if ply.applyElement == nil && ply != room.dealer {
		return
	}

	var e errcode.Error
	limit, _ := config.Int("lottery", room.SubId, "forceCancelDealerGold")
	if false && room.dealer == ply && room.dealer.dealerGold < limit {
		e = errDealerNeedMoreGold
	}

	// 已当庄，游戏中不可下庄
	room.delayCancelDealer = false
	if room.dealer == ply && room.Status != 0 {
		// 结算后自动下庄
		e = errDelayCancelDealer
		room.delayCancelDealer = true
	}
	uid := ply.Id
	data := map[string]any{
		"code":     "ok",
		"msg":      "ok",
		"uid":      uid,
		"isDealer": (room.dealer == ply),
	}
	if e != nil {
		data["code"] = e.GetCode()
		data["msg"] = e
	}

	ply.WriteJSON("cancelDealer", data)
	if room.delayCancelDealer {
		return
	}

	ply.dealerGold = 0
	if ply.applyElement != nil {
		room.dealerQueue.Remove(ply.applyElement)
		ply.applyElement = nil
	}
	if room.dealer == ply {
		room.dealer = nil
		room.dealerLoop = 0
		room.delayCancelDealer = false
		room.Broadcast("cancelDealer", data, ply.Id)
	}
}

// 0 表示没有申请上庄
func (ply *lotteryPlayer) dealerQueueRank() int {
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
	Top       []*userInfo `json:"top"`
	Len       int         `json:"len"`
	Rank      int         `json:"rank"`
	LimitGold int64       `json:"limitGold"`
}

// 上庄列表
func (ply *lotteryPlayer) dealerQueue() *DealerQueue {
	room := ply.Room()

	var dealers []*userInfo
	for e := room.dealerQueue.Front(); e != nil; e = e.Next() {
		p := e.Value.(*lotteryPlayer)
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

func (ply *lotteryPlayer) Room() *lotteryRoom {
	if room := roomutils.GetRoomObj(ply.Player).CustomRoom(); room != nil {
		return room.(*lotteryRoom)
	}
	return nil
}

func (ply *lotteryPlayer) OnAddItems(items []gameutils.Item, way string) {
	ply.Player.OnAddItems(items, way)
	for _, item := range items {
		if item.GetId() == gameutils.ItemIdGold {
			ply.dealerGold += item.GetNum()
		}
	}
}
