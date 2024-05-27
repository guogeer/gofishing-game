package zhajinhua

// 2017-9-5

import (
	"gofishing-game/internal/errcode"
	"gofishing-game/internal/gameutils"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
	"slices"

	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
	"github.com/guogeer/quasar/utils"
)

const (
	CauseNone = iota
	CauseFold // 弃牌
	CauseFail // 比牌输
)

const (
	ActionNone    = iota
	ActionFold    // 弃牌
	ActionCall    // 跟注
	ActionRaise   // 加注
	ActionCompare // 比牌
)

const (
	AutoNone = iota
	AutoCall // 自动跟注
)

type Item = gameutils.Item

// 愚蠢的玩家
// 机器人作弊
type StupidUser struct {
	UId       int
	Cards     []int
	SeatIndex int
	IsRobot   bool `json:",omitempty"`
}

type CardResult struct {
	Cards    []int
	CardType int
}

type CompareResult struct {
	Winner         int
	Seats          []int
	CompareSeats   []int
	CompareResults []CardResult `json:",omitempty"`

	Self  *CardResult `json:",omitempty"`
	Other *CardResult `json:",omitempty"`
}

// 玩家信息
type ZhajinhuaUserInfo struct {
	service.UserInfo
	SeatIndex int
	Cards     []int `json:",omitempty"`

	IsReady, IsLook, IsShow, IsAutoFold bool `json:",omitempty"`

	Cause  int `json:",omitempty"`
	Action int `json:",omitempty"`

	// 和谁比牌
	CompareSeats []int `json:",omitempty"`
	Bet          int64
	CallTimes    int // 跟注次数
	RaiseTimes   int // 加注次数
	CardType     int `json:",omitempty"`

	Auto int `json:",omitempty"` // 自动操作
}

type ZhajinhuaPlayer struct {
	*service.Player

	cards              []int // 手牌
	bet                int64 // 押注
	action             int
	cause              int  // 弃牌或比牌输
	callTimes          int  // 跟注次数
	raiseTimes         int  // 加注次数
	isLook             bool // 看牌
	isShow             bool // 亮牌
	loop               int
	compareSeats       []int   // 比牌时0请求和1比牌
	betHistory         []int64 // 投注历史记录
	winGold            int64
	extraGold          int64
	auto               int  // 自动操作
	isAutoFold         bool // 系统弃牌
	continuousAutoFold int  // 系统连续弃牌次数

	isAbleLook  bool  // 可看牌
	callGold    int64 // 跟注金币
	raiseGold   int64 // 加注金币
	compareGold int64 // 比牌金币，>0 可比牌
	isPaying    bool  // 正在充值
	isAllIn     bool

	autoTimer, operateTimer *utils.Timer
}

func (ply *ZhajinhuaPlayer) TryEnter() errcode.Error {
	return nil
}

func (ply *ZhajinhuaPlayer) BeforeEnter() {
}

func (ply *ZhajinhuaPlayer) AfterEnter() {
}

func (ply *ZhajinhuaPlayer) BeforeLeave() {
	if ply.GetSeatIndex() != roomutils.NoSeat {
		ply.SitUp()
	}
}

func (ply *ZhajinhuaPlayer) TryLeave() errcode.Error {
	room := ply.Room()
	if room.IsTypeScore() && room.Status != 0 {
		return errcode.Retry
	}
	return nil
}

func (ply *ZhajinhuaPlayer) initGame() {
	for i := range ply.cards {
		ply.cards[i] = 0
	}

	ply.bet = 0
	ply.cause = CauseNone
	ply.isLook = false
	ply.loop = 0
	ply.betHistory = nil
	ply.compareGold = 0
	ply.compareSeats = nil

	ply.action = ActionNone
	ply.callTimes = 0
	ply.raiseTimes = 0
	ply.isShow = false
	ply.isAutoFold = false
	ply.auto = AutoNone
	utils.StopTimer(ply.autoTimer)
	utils.StopTimer(ply.operateTimer)
	ply.isAllIn = false
	ply.isAbleLook = false
}

func (ply *ZhajinhuaPlayer) IsPlaying() bool {
	return roomutils.GetRoomObj(ply.Player).IsReady() && ply.cause == 0
}

func (ply *ZhajinhuaPlayer) GameOver() {
	ply.initGame()
}

// 看牌
func (ply *ZhajinhuaPlayer) LookCard() {
	room := ply.Room()
	if !ply.isAbleLook {
		return
	}
	data := map[string]any{"uid": ply.Id}
	room.Broadcast("LookCard", data, ply.Id)

	ply.isLook = true
	ply.isAbleLook = false
	data["Cards"] = ply.cards
	ply.WriteJSON("LookCard", data)
	if ply == room.activePlayer {
		ply.OnTurn()
	}
}

// 比牌
func (ply *ZhajinhuaPlayer) CompareCard(seatId int) {
	room := ply.Room()
	if ply != room.activePlayer {
		return
	}

	other := room.GetPlayer(seatId)
	if other == nil || !other.IsPlaying() {
		return
	}
	if ply.compareGold <= 0 || ply.compareGold > ply.BagObj().NumItem(gameutils.ItemIdGold) {
		return
	}

	loser, winner := ply, other
	if room.helper.Less(other.cards, ply.cards) {
		loser, winner = other, ply
	}
	activeUsers := 0
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil && p.IsPlaying() {
			activeUsers++
		}
	}

	ply.loop++
	ply.action = ActionCompare
	loser.cause = CauseFail

	room.currentChip = ply.compareGold
	if ply.isLook {
		room.currentChip = (room.currentChip + 1) / 2
	}

	seats := []int{ply.GetSeatIndex(), seatId}
	winner.compareSeats = seats
	loser.compareSeats = seats
	// loser.AddGoldLog(-loser.bet, guid, "user.zhajinhua_fail")

	utils.StopTimer(ply.operateTimer)

	ply.OnBet(ply.compareGold)
	ply.compareGold = 0 // 清除
	data := CompareResult{
		Winner:       winner.Id,
		Seats:        seats,
		CompareSeats: seats,
	}

	resultA := CardResult{Cards: ply.cards}
	resultA.CardType, _ = room.helper.GetType(ply.cards)

	resultB := CardResult{Cards: other.cards}
	resultB.CardType, _ = room.helper.GetType(other.cards)
	if activeUsers == 2 {
		tempData := data
		tempData.CompareResults = []CardResult{resultA, resultB}
		room.Broadcast("CompareCard", tempData)
	} else {
		room.Broadcast("CompareCard", data, ply.Id, other.Id)

		// 比牌时只有自己翻看了自己的牌才能看到对方的牌，如果在比牌时没有翻看自己的牌是不能看到对方以及自己的牌
		tempData := data
		if ply.isLook {
			tempData.Self = &resultA
			tempData.Other = &resultB
			tempData.CompareResults = []CardResult{resultA, resultB}
		}
		ply.WriteJSON("CompareCard", tempData)

		tempData = data
		other.WriteJSON("CompareCard", tempData)
	}
	room.OnTakeAction()
}

func (ply *ZhajinhuaPlayer) OnAddItems(items []gameutils.Item, way string) {
	room := ply.Room()
	ply.Player.OnAddItems(items, way)
	for _, item := range items {
		if item.GetId() == gameutils.ItemIdGold && ply.isPaying {
			ply.isPaying = false

			t := room.maxAutoTime()
			utils.ResetTimer(ply.operateTimer, t)
			room.OnTurn()
		}
	}
}

// gold = -1, fold
// gold = -2, system auto fold
// gold >  0, call or raise
func (ply *ZhajinhuaPlayer) TakeAction(gold int64) {
	room := ply.Room()
	if !ply.IsPlaying() {
		return
	}
	if room.Status != roomutils.RoomStatusPlaying {
		return
	}

	maxBet := ply.maxBet()
	log.Debugf("player %d gold %d bet %d max bet %d", ply.Id, ply.BagObj().NumItem(gameutils.ItemIdGold), gold, maxBet)
	if gold >= 0 && room.activePlayer != ply {
		return
	}
	chips, call, _ := ply.Chips()
	if len(chips) > 0 && gold >= 0 && !slices.Contains(chips, gold) {
		return
	}
	if len(chips) == 0 && gold >= 0 && gold < call && gold != maxBet {
		return
	}
	if maxBet > 0 && gold > maxBet {
		return
	}
	if !room.IsTypeScore() && gold > ply.BagObj().NumItem(gameutils.ItemIdGold) {
		e := errcode.MoreItem(gameutils.ItemIdGold)
		ply.WriteErr("TakeAction", e, "uid", ply.Id)

		ply.isPaying = true
		utils.ResetTimer(ply.operateTimer, maxPayTime)
		room.OnTurn()
		return
	}

	// OK
	if room.IsAbleAllIn() && gold == maxBet {
		ply.isAllIn = true
	}

	ply.loop++
	ply.compareGold = 0 // 清除
	ply.compareSeats = nil
	ply.action = ActionNone
	ply.isPaying = false

	if gold == -2 {
		ply.isAutoFold = true
		ply.continuousAutoFold++
	} else if !room.IsTypeScore() {
		ply.continuousAutoFold = 0
	}

	if gold < 0 {
		ply.bet = 0
		ply.cause = CauseFold
		ply.action = ActionFold
		// ply.AddGoldLog(-ply.bet, guid, "user.zhajinhua_bet")

	} else {
		ply.action = ActionRaise
		if gold == call {
			ply.action = ActionCall
		}

		ply.bet += gold
		ply.betHistory = append(ply.betHistory, gold)
		ply.BagObj().Add(gameutils.ItemIdGold, -gold, "zhajinhua_bet", service.WithNoItemLog())
		room.allBet[ply.GetSeatIndex()] = ply.bet

		currentChip := gold
		if ply.isLook {
			currentChip = (gold + 1) / 2
		}
		if currentChip > room.currentChip {
			room.currentChip = currentChip
		}
	}
	times := 0
	if ply.action == ActionCall {
		times = ply.callTimes + 1
	}
	ply.callTimes = times

	times = 0
	if ply.action == ActionRaise {
		times = ply.raiseTimes + 1
		for i := 0; i < room.NumSeat(); i++ {
			if other := room.GetPlayer(i); other != nil && other.IsPlaying() {
				other.callTimes = 0
			}
		}
	}
	ply.raiseTimes = times

	data := map[string]any{
		"code":       "ok",
		"uid":        ply.Id,
		"Bet":        ply.bet,
		"Gold":       gold,
		"Action":     ply.action,
		"CallTimes":  ply.callTimes,
		"RaiseTimes": ply.raiseTimes,
	}
	utils.StopTimer(ply.autoTimer)
	utils.StopTimer(ply.operateTimer)
	room.Broadcast("TakeAction", data)

	room.OnTakeAction()
	t, ok := config.Int("config", "ZhajinhuaAllowContinuousAutoFold", "Value")
	if ok && ply.continuousAutoFold > int(t) {
		ply.continuousAutoFold = 0
		ply.Leave()
	}
}

func (ply *ZhajinhuaPlayer) GetUserInfo(self bool) *ZhajinhuaUserInfo {
	info := &ZhajinhuaUserInfo{}
	info.UserInfo = ply.UserInfo
	// info.UId = ply.GetCharObj().Id
	info.SeatIndex = ply.GetSeatIndex()
	info.Auto = ply.auto
	info.IsAutoFold = ply.isAutoFold
	info.IsReady = roomutils.GetRoomObj(ply.Player).IsReady()

	room := ply.Room()
	if room.Status == roomutils.RoomStatusPlaying && roomutils.GetRoomObj(ply.Player).IsReady() {
		info.Bet = ply.bet
		info.CallTimes = ply.callTimes
		info.RaiseTimes = ply.raiseTimes
		info.Cause = ply.cause
		info.Action = ply.action
		if len(ply.compareSeats) > 0 {
			info.CompareSeats = ply.compareSeats[:]
		}

		info.IsLook = ply.isLook
		if self && ply.isLook {
			info.Cards = ply.cards[:]
			info.CardType, _ = room.helper.GetType(ply.cards[:])
		}
	}
	if room.Status == 0 && roomutils.GetRoomObj(ply.Player).IsReady() {
		info.IsShow = ply.isShow
		if ply.isShow {
			info.Cards = ply.cards[:]
			info.CardType, _ = room.helper.GetType(ply.cards[:])
		}
	}

	return info
}

// 站起
func (ply *ZhajinhuaPlayer) SitUp() {
	room := ply.Room()
	log.Debugf("player %d sit up", ply.Id)
	if room.Status == roomutils.RoomStatusPlaying {
		ply.TakeAction(-1)
	}
	roomutils.GetRoomObj(ply.Player).SitUp()
}

func (ply *ZhajinhuaPlayer) Room() *ZhajinhuaRoom {
	if room := roomutils.GetRoomObj(ply.Player).CustomRoom(); room != nil {
		return room.(*ZhajinhuaRoom)
	}
	return nil
}

func (ply *ZhajinhuaPlayer) Replay(messageId string, i interface{}) {
	switch messageId {
	case "SitDown":
		return
	case "StartDealCard":
		room := ply.Room()
		data := i.(map[string]any)
		all := make([][]int, room.NumSeat())
		for k := 0; k < room.NumSeat(); k++ {
			if other := room.GetPlayer(k); other != nil && other.cards != nil {
				all[k] = other.cards[:]
			}
		}
		data["All"] = all
		defer func() { delete(data, "All") }()
	}
	ply.Player.Replay(messageId, i)
}

func (ply *ZhajinhuaPlayer) ChangeRoom() {
	ply.SitUp()

	e := roomutils.GetRoomObj(ply.Player).ChangeRoom()
	ply.WriteErr("ChangeRoom", e)
	if e != nil {
		return
	}
	ply.initGame()
	ply.OnEnter()

	roomutils.GetRoomObj(ply.Player).Ready()
}

// 筹码、跟注、加注
func (ply *ZhajinhuaPlayer) Chips() ([]int64, int64, int64) {
	room := ply.Room()
	chips := make([]int64, 0, 8)
	chips = append(chips, room.chips...)
	// TODO
	if len(chips) == 0 {
		call := room.currentChip
		if ply.isLook {
			call *= 2
			if ply.BagObj().NumItem(gameutils.ItemIdGold)+1 == call {
				call = ply.BagObj().NumItem(gameutils.ItemIdGold)
			}
		}
	}
	for i, chip := range chips {
		if chip < room.currentChip {
			chips[i] = -chips[i]
		}
	}

	if ply.isLook {
		for i := range chips {
			chips[i] = chips[i] << 1
		}
	}

	start := 0
	for k, chip := range chips {
		if chip > 0 {
			start = k
			break
		}
	}
	if start+1 == len(chips) {
		return chips, chips[start], chips[start]
	}
	return chips, chips[start], chips[start+1]
}

func (ply *ZhajinhuaPlayer) OnTurn() {
	room := ply.Room()
	current := room.activePlayer
	data := map[string]any{
		"uid": current.Id,
		"Sec": room.Countdown(),
	}
	if current.isPaying {
		data["IsPaying"] = true
	}

	if ply.IsPlaying() {
		maxBet := ply.maxBet()
		chips, call, raise := ply.Chips()

		if !ply.isLook && ply.loop >= room.lookLoopLimit && ply.IsPlaying() {
			ply.isAbleLook = true
		}
		if ply.isAbleLook {
			data["IsAbleLook"] = true
		}
		// 第二轮开始比牌
		if ply == current && ply.loop >= room.compareLoopLimit {
			ply.compareGold = raise
		}
		if gold := ply.compareGold; gold > 0 {
			data["IsAbleCompare"] = true
			data["CompareGold"] = gold
		}

		if len(chips) == 0 {
			chips = []int64{call, raise}
		}
		data["Call"] = call
		data["Raise"] = chips
		data["AllIn"] = maxBet
	}
	ply.WriteJSON("Turn", data)
}

func (ply *ZhajinhuaPlayer) maxBet() int64 {
	room := ply.Room()

	chips, _, _ := ply.Chips()
	if n := len(chips); n > 0 {
		return chips[n-1]
	}

	maxBet := ply.BagObj().NumItem(gameutils.ItemIdGold)
	if room.maxBet > 0 && maxBet < room.maxBet {
		maxBet = room.maxBet
	}

	var currentChip int64
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil && p.IsPlaying() {
			chip := p.NumGold()
			if p.isLook {
				chip = (p.NumGold() + 1) / 2
			}
			// 有人已全压
			if p.isAllIn {
				currentChip = room.currentChip
				break
			}
			if currentChip == 0 || currentChip > chip {
				currentChip = chip
			}
		}
	}
	if room.IsAbleAllIn() {
		maxBet = currentChip
	}
	if ply.isLook {
		maxBet = 2 * maxBet
		if !room.IsTypeScore() && ply.BagObj().NumItem(gameutils.ItemIdGold)+1 == maxBet {
			maxBet = ply.BagObj().NumItem(gameutils.ItemIdGold)
		}
	}
	return maxBet
}

// 亮牌
func (ply *ZhajinhuaPlayer) ShowCard() {
	if ply.isShow {
		return
	}

	room := ply.Room()
	if room.Status != 0 {
		return
	}
	ply.isShow = true
	room.Broadcast("ShowCard", map[string]any{"uid": ply.Id})
}

func (ply *ZhajinhuaPlayer) AutoPlay() {
	room := ply.Room()
	act := room.activePlayer

	_, call, _ := ply.Chips()
	// 自动跟注
	if ply == act && ply.auto == AutoCall {
		utils.StopTimer(ply.autoTimer)
		ply.autoTimer = ply.TimerGroup.NewTimer(func() { ply.TakeAction(call) }, systemAutoPlayTime)
	}
}

func (ply *ZhajinhuaPlayer) OnBet(gold int64) {
	room := ply.Room()

	ply.bet += gold
	ply.betHistory = append(ply.betHistory, gold)
	ply.BagObj().Add(gameutils.ItemIdGold, -gold, "zhajinhua_compare", service.WithNoItemLog())

	room.allBet[ply.GetSeatIndex()] = ply.bet
	room.Broadcast("BetOk", map[string]any{"uid": ply.Id, "Gold": gold, "Action": ply.action})
}

func (ply *ZhajinhuaPlayer) GetSeatIndex() int {
	return roomutils.GetRoomObj(ply.Player).GetSeatIndex()
}
