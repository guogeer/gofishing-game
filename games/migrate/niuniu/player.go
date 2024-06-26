package niuniu

import (
	"gofishing-game/internal/errcode"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"

	"github.com/guogeer/quasar/v2/log"
)

// 玩家信息
type NiuNiuPlayerInfo struct {
	service.UserInfo
	SeatIndex int   `json:"seatIndex,omitempty"`
	Cards     []int `json:"cards,omitempty"`
	Chips     []int `json:"chips,omitempty"`
	TriCards  []int `json:"triCards,omitempty"` // 自动算牛的三张牌
	// 准备、房主开始游戏，亮牌、看牌
	IsReady        bool `json:"isReady,omitempty"`
	StartGameOrNot bool `json:"startGameOrNot,omitempty"`
	EndGameOrNot   bool `json:"endGameOrNot,omitempty"`
	IsDone         bool `json:"isDone,omitempty"`

	// 牛几
	RobTimes     int `json:"robTimes,omitempty"`
	BetTimes     int `json:"betTimes,omitempty"`
	ExpectWeight int `json:"expectWeight,omitempty"`
	Weight       int `json:"weight,omitempty"`
	WeightTimes  int `json:"weightTimes,omitempty"`

	RobOrNot int `json:"robOrNot,omitempty"`
}

type NiuNiuPlayer struct {
	*service.Player

	cards         []int // 手牌
	weight        int   // 牛几最终结果
	expectWeight  int   // 预期的牛
	expectCards   []int // 预期的牌顺序
	doneCards     []int // 显示的牌
	robOrNot      int   // 自由抢庄
	robTimes      int   // 抢庄倍数
	betTimes      int   // 押注倍数
	autoTimes     int   // 自动托管押注
	lastWinGold   int64
	betInAddition bool // 闲家推注
}

// 已亮牌
func (ply *NiuNiuPlayer) IsDone() bool {
	return ply.doneCards[0] != 0
}

func (ply *NiuNiuPlayer) TryEnter() errcode.Error {
	return nil
}

func (ply *NiuNiuPlayer) BeforeEnter() {
}

func (ply *NiuNiuPlayer) AfterEnter() {
}

func (ply *NiuNiuPlayer) BeforeLeave() {
}

func (ply *NiuNiuPlayer) TryLeave() errcode.Error {
	room := ply.Room()
	if room.Status != 0 {
		return errcode.Retry
	}
	return nil
}

func (ply *NiuNiuPlayer) initGame() {
	ply.weight = -1
	for i := 0; i < len(ply.cards); i++ {
		ply.cards[i] = 0
	}
	for i := 0; i < len(ply.doneCards); i++ {
		ply.doneCards[i] = 0
	}

	ply.robOrNot = -1
	ply.robTimes = -1
	ply.betTimes = -1
}

// 算牌
func (ply *NiuNiuPlayer) ChooseTriCards(tri [3]int) {
	if !roomutils.GetRoomObj(ply.Player).IsReady() {
		return
	}
	if ply.IsDone() {
		return
	}

	room := ply.Room()
	log.Debugf("player %d %v choose tri cards %v", ply.Id, ply.cards, tri)
	if room.Status != RoomStatusLook {
		return
	}

	// 默认无牛
	ply.weight = 0
	copy(ply.doneCards, ply.cards)
	if w := ply.expectWeight; w > 10 {
		ply.weight = w
	} else if tri[0] > 0 {
		var total int
		counter := map[int]int{}
		for _, c := range ply.cards {
			counter[c]++
		}
		for _, c := range tri {
			counter[c]--
			total += room.helper.GetCardWeight(c)
			if counter[c] < 0 {
				return
			}
		}
		n := copy(ply.doneCards, tri[:])
		ply.doneCards = ply.doneCards[:n]
		for c, n := range counter {
			for k := 0; k < n; k++ {
				ply.doneCards = append(ply.doneCards, c)
			}
		}
		if total%10 == 0 {
			ply.weight = ply.expectWeight
		}
	}

	// 庄家最后显示
	room.Broadcast("chooseTriCards", map[string]any{
		"uid":    ply.Id,
		"weight": ply.weight,
		"times":  room.getWeightTimes(ply.weight),
		"cards":  ply.doneCards,
	})
	room.OnChooseTriCards()
}

func (ply *NiuNiuPlayer) GameOver() {
	ply.initGame()
}

func (ply *NiuNiuPlayer) Bet(times int) {
	if !roomutils.GetRoomObj(ply.Player).IsReady() {
		return
	}

	room := ply.Room()
	log.Debugf("player %d bet %d %d status %d chips %v", ply.Id, ply.betTimes, times, room.Status, ply.Chips())
	if ply.betTimes != -1 {
		return
	}
	if room.Status != RoomStatusBet {
		return
	}

	exist := false
	for _, chip := range ply.Chips() {
		if chip == times {
			exist = true
		}
	}

	if !exist {
		return
	}

	if times == ply.additionChip() {
		ply.betInAddition = true
	}
	// OK
	// ply.times = times
	ply.betTimes = times
	room.Broadcast("bet", map[string]any{"uid": ply.Id, "times": times})
	room.OnBet()
}

func (ply *NiuNiuPlayer) ChooseDealer(b bool) {
	if !roomutils.GetRoomObj(ply.Player).IsReady() {
		return
	}
	if ply.robOrNot != -1 {
		return
	}

	// OK
	room := ply.Room()

	ply.robOrNot = 0
	if b {
		ply.robOrNot = 1
	}
	room.Broadcast("chooseDealer", map[string]any{"code": "ok", "uid": ply.Id, "ans": b})
	room.OnChooseDealer()
}

func (ply *NiuNiuPlayer) DoubleAndRob(times int) {
	if !roomutils.GetRoomObj(ply.Player).IsReady() {
		return
	}
	if ply.robTimes != -1 {
		return
	}
	if times < 0 {
		return
	}

	room := ply.Room()
	// 加倍抢庄
	if room.Status != RoomStatusRobDealer {
		return
	}

	// OK
	// ply.times = times
	ply.robTimes = times
	room.Broadcast("doubleAndRob", map[string]any{"code": "ok", "uid": ply.Id, "times": times})
	room.OnDoubleAndRob()
}

func (ply *NiuNiuPlayer) GetSeatIndex() int {
	return roomutils.GetRoomObj(ply.Player).GetSeatIndex()
}

func (ply *NiuNiuPlayer) GetUserInfo(self bool) *NiuNiuPlayerInfo {
	info := &NiuNiuPlayerInfo{}
	info.UserInfo = ply.UserInfo
	// info.UId = ply.GetCharObj().Id
	info.SeatIndex = ply.GetSeatIndex()

	info.IsReady = roomutils.GetRoomObj(ply.Player).IsReady()
	if t := ply.betTimes; t >= 0 {
		info.BetTimes = t
	}

	room := ply.Room()
	info.RobTimes = ply.robTimes
	info.RobOrNot = ply.robOrNot
	info.BetTimes = ply.betTimes

	if roomutils.GetRoomObj(ply.Player).IsReady() && len(ply.cards) > 0 && ply.cards[0] > 0 {
		info.IsDone = ply.IsDone()
		if room.CanPlay(OptMingPaiShangZhuang) {
			info.Cards = make([]int, len(ply.cards))
			if self {
				copy(info.Cards, ply.cards)
				info.Cards[len(info.Cards)-1] = 0
			}
		}
		if room.Status == RoomStatusLook {
			info.Cards = make([]int, len(ply.cards))
			if self {
				info.ExpectWeight = ply.expectWeight
				copy(info.Cards, ply.cards)

				info.TriCards = make([]int, 3)
				copy(info.TriCards, ply.expectCards[:3])
			}
		}

		// 已亮牌
		if ply.IsDone() {
			info.IsDone = true
			info.Weight = ply.weight
			info.WeightTimes = room.getWeightTimes(ply.weight)
			copy(info.Cards, ply.doneCards)
		}
	}
	if room.Status == RoomStatusBet && roomutils.GetRoomObj(ply.Player).IsReady() {
		info.Chips = ply.Chips()
	}
	return info
}

// 坐下
func (ply *NiuNiuPlayer) SitDown(seatIndex int) {
	room := ply.Room()
	if e := roomutils.GetRoomObj(ply.Player).SitDown(seatIndex); e != nil {
		return
	}
	// OK
	info := ply.GetUserInfo(false)
	room.Broadcast("sitDown", map[string]any{"code": "ok", "info": info})
}

// 固定当庄结束游戏
func (ply *NiuNiuPlayer) EndGame() {
	room := ply.Room()
	if ply != room.dealer {
		return
	}
	if !room.isAbleEnd {
		return
	}
	room.LimitTimes = room.ExistTimes - 1
	room.GameOver()
}

func (ply *NiuNiuPlayer) additionChip() int {
	room := ply.Room()

	chips := ply.Chips()
	// 闲家推注
	if !room.CanPlay(OptXianJiaTuiZhu) {
		return 0
	}
	g := int(ply.lastWinGold)
	if ply == room.dealer || ply.betInAddition || len(chips) <= 0 || g <= 0 {
		return 0
	}
	lastChip := chips[len(chips)-1]
	if sub := 10*chips[0] - lastChip; sub > 0 {
		g = sub
	}
	return g + lastChip
}

func (ply *NiuNiuPlayer) Room() *NiuNiuRoom {
	if room := roomutils.GetRoomObj(ply.Player).CustomRoom(); room != nil {
		return room.(*NiuNiuRoom)
	}
	return nil
}

func (ply *NiuNiuPlayer) Less(other *NiuNiuPlayer) bool {
	room := ply.Room()
	if ply.weight == other.weight {
		return room.helper.LessMaxCard(ply.cards, other.cards)
	}
	return room.helper.LessWeight(ply.weight, other.weight)
}

func (ply *NiuNiuPlayer) Chips() []int {
	room := ply.Room()
	if ply == room.dealer {
		return nil
	}
	if !roomutils.GetRoomObj(ply.Player).IsReady() {
		return nil
	}

	var chips []int
	if room.CanPlay(OptDiZhu1_2) {
		chips = []int{1, 2}
	} else if room.CanPlay(OptDiZhu2_4) {
		chips = []int{4, 8}
	} else if room.CanPlay(OptDiZhu4_8) {
		chips = []int{4, 8}
	} else {
		chips = []int{1, 2, 4, 8, 20}
		if room.IsTypeScore() {
			chips = []int{1, 2, 3, 4, 5}
		}
	}
	if g := ply.additionChip(); g > 0 {
		chips = append(chips, g)
	}
	return chips
}
