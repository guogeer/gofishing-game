package internal

import (
	"container/list"
	"fmt"
	"gofishing-game/service"
	"strings"
	. "third/errcode"
	"third/gameutil"

	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
	"github.com/guogeer/quasar/script"
	"github.com/guogeer/quasar/util"
)

type Item = gameutils.Item

// 座位上的玩家
type userInfo struct {
	service.SimpleUserInfo
	// Gold   int64
	SeatId          int
	DealerLimitGold int64 `json:",omitempty"`
	DealerGold      int64 `json:",omitempty"`
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

func (ply *entertainmentPlayer) AfterEnter() {
	// 自动坐下
	room := ply.Room()
	seatId := room.GetEmptySeat()
	if ply.SeatId == service.NoSeat && seatId != service.NoSeat {
		if err := ply.TrySitDown(seatId); err.Code == Ok {
			ply.SitDown(seatId)
		}
	}
	// OK
	// 移除进入房间的通知
	/*(msg := fmt.Sprintf("%s进来了", ply.Nickname)
	room.Broadcast("Broadcast", map[string]any{"Info": msg, "Message": msg}, ply.Id)
	*/
	if len(ply.betAreas) != room.userAreaNum {
		ply.betAreas = make([]int64, room.userAreaNum)
	}
	if ply.fakeAreas == nil {
		ply.fakeAreas = make([]int64, len(ply.betAreas))
	}
}

func (ply *entertainmentPlayer) GetUserInfo(otherId int) *userInfo {
	info := &userInfo{
		SeatId:          ply.SeatId,
		DealerGold:      ply.dealerGold,
		DealerLimitGold: ply.dealerLimitGold,
	}
	simpleInfo := ply.GetSimpleInfo(otherId)
	info.SimpleUserInfo = *simpleInfo
	return info
}

func (ply *entertainmentPlayer) TrySitDown(seatId int) errcode.Error {
	room := ply.Room()
	args := &struct {
		SubId int
		UId   int
		Gold  int64
		Code  int
		Msg   string
	}{
		SubId: room.GetSubId(),
		UId:   ply.Id,
		Gold:  ply.Gold,
	}
	script.Call("room.lua", "try_sit_down", args)
	if args.Code != 0 {
		return Error{Code: ErrCode(args.Code), Msg: args.Msg}
	}
	if ply == room.dealer {
		return NewError(Retry)
	}
	return NewError(Ok)
}

func (ply *entertainmentPlayer) SitDown(seatId int) {
	err := NewError(Ok)
	room := ply.Room()

	defer func() {
		info := ply.GetUserInfo(ply.Id)
		data := map[string]any{
			"Code": err.Code,
			"Msg":  err.Msg,
			"Info": &info,
		}
		ply.WriteJSON("SitDown", data)
		if err.Code == Ok {
			info = ply.GetUserInfo(0)
			room.Broadcast("SitDown", data, ply.Id)
		}
	}()

	err = ply.TrySitDown(seatId)
	if err.Code != Ok {
		return
	}
	// 抢座
	other := room.GetPlayer(seatId)
	if other != nil {
		// 本人已坐下，或者房间不可抢座
		if ply == other || room.robSeat != seatId {
			err = NewError(Retry)
			return
		}
		// 金币不足
		if ply.Gold < other.Gold {
			err = NewError(MoreGold)
			return
		}

		other.RoomObj.SitUp()
	}

	if code := ply.RoomObj.TrySitDown(seatId); code != Ok {
		err = NewError(code)
		return
	}

	// OK
	if other != nil {
		room.robSeat = service.NoSeat
	}
}

func (ply *entertainmentPlayer) TryLeave() ErrCode {
	// 游戏中
	room := ply.Room()
	if room.Status != service.RoomStatusFree {
		// 庄家
		if room.dealer == ply {
			return PlayingInGame
		}
		// 玩家已下注
		if ply.totalBet() > 0 {
			return AlreadyBet
		}
	}
	if ply.IsRobotAD() {
		return Retry
	}
	return Ok
}

// 机器人头像广告位
func (ply *entertainmentPlayer) IsRobotAD() bool {
	return ply.SeatId != -1 && ply.IsRobot && strings.HasSuffix(ply.Icon, "AD.jpg")
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
	if room.Status != service.RoomStatusPlaying {
		return
	}
	err := NewError(Ok)
	// 无效的数据
	if area < 0 || area >= len(ply.betAreas) || gold <= 0 {
		err = NewError(Retry)
	}
	// 庄家不可投注
	if room.dealer == ply {
		err = NewError(Retry)
	}
	if gold > ply.Gold {
		err = NewError(MoreGold)
	}
	{
		total := ply.totalBet()
		percent, ok := config.Float("entertainment", room.GetSubId(), "MaxBetPercent")
		if ok && total+gold > int64(percent*float64(total+ply.Gold)/100) {
			err = NewError(TooMuchBet)
		}
		maxBet, _ := config.Int("entertainment", room.GetSubId(), "MaxBetLimit")
		if maxBet > 0 && total+gold > maxBet {
			err = NewError(TooMuchBet)
		}
	}
	{
		sum := room.totalBet()
		percent, ok := config.Float("entertainment", room.GetSubId(), "AllUserBetPercent")
		if ok == true && room.dealer != nil && float64(sum+gold) > float64(room.dealer.dealerGold)*percent/100 {
			err = NewError(DealerMoreGold)
		}
	}
	{
		// 最低押注金币要求
		minBetNeedGold, ok := config.Int("entertainment", room.GetSubId(), "MinBetNeedGold")
		if ok == true && ply.Gold < minBetNeedGold {
			scale, _ := config.Int("web", "ExchangeScale", "Value")
			if scale < 1 {
				scale = 1
			}
			err = NewError(MoreBetGold, minBetNeedGold/scale)
		}
	}

	data := map[string]any{
		"Code": err.Code,
		"Msg":  err.Msg,
		"UId":  ply.Id,
		"Area": area,
		"Gold": gold,
	}

	betArgs := &struct {
		SubId  int
		Name   string
		UId    int
		Gold   int64
		Area   int
		Bet    int64
		BigBet int64
		Code   int
		Msg    string
	}{
		Name:  service.GetName(),
		UId:   ply.Id,
		Gold:  ply.Gold,
		Bet:   gold,
		Area:  area,
		SubId: ply.GetSubId(),
	}
	script.Call("room.lua", "try_bet", betArgs)
	if betArgs.Code != 0 {
		err = Error{Code: ErrCode(betArgs.Code), Msg: betArgs.Msg}
	}
	if betArgs.BigBet > 0 {
		data["Info"] = ply.GetSimpleInfo(0)
	}
	ply.WriteJSON("Bet", data)

	if !ply.IsRobot {
		log.Infof("player %d bet area %d gold %d", ply.Id, area, gold)
	}
	if err.Code != Ok {
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
	if ply.SeatId != service.NoSeat || betArgs.BigBet > 0 {
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
	code := Ok
	room := ply.Room()
	// 玩家已申请或已当庄
	if room.dealer == ply {
		code = Retry
	}
	if e := ply.applyElement; e != nil {
		code = Retry
	}
	minDealerGold, _ := config.Int("entertainment", room.GetSubId(), "MinDealerGold")
	if ply.Gold < minDealerGold {
		code = MoreGold
	}
	if ply.IsRobotAD() {
		code = Retry
	}

	uid := ply.Id
	data := map[string]any{
		"Code": code,
		"Msg":  code.String(),
		"UId":  uid,
	}
	ply.WriteJSON("ApplyDealer", data)
	// OK
	if code == Ok {
		ply.applyElement = room.dealerQueue.PushBack(ply)
		room.Broadcast("ApplyDealer", data, ply.Id)
	}
}

func (ply *entertainmentPlayer) CancelDealer() {
	room := ply.Room()
	// 玩家未当庄或申请上庄
	if ply.applyElement == nil && ply != room.dealer {
		return
	}

	code := Ok
	limit, _ := config.Int("entertainment", room.GetSubId(), "ForceCancelDealerGold")
	if false && room.dealer == ply && room.dealer.dealerGold < limit {
		code = DealerNeedMoreGold
	}

	// 已当庄，游戏中不可下庄
	room.delayCancelDealer = false
	if room.dealer == ply && room.Status != service.RoomStatusFree {
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
	if room := ply.RoomObj.CardRoom(); room != nil {
		return room.(*entertainmentRoom)
	}
	return nil
}

func (ply *entertainmentPlayer) OnAddItems(items []*Item, guid, way string) {
	ply.Player.OnAddItems(items, guid, way)
	for _, item := range items {
		if item.Id == gameutil.ItemIdGold {
			ply.dealerGold += item.Num
		}
	}
}
