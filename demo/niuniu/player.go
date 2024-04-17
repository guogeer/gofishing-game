package niuniu

import (
	"gofishing-game/service"
	. "third/errcode"

	"github.com/guogeer/quasar/log"
)

// 玩家信息
type NiuNiuPlayerInfo struct {
	service.UserInfo
	SeatId   int
	Cards    []int `json:",omitempty"`
	Chips    []int `json:",omitempty"`
	TriCards []int `json:",omitempty"` // 自动算牛的三张牌
	// 准备、房主开始游戏，亮牌、看牌
	IsReady, StartGameOrNot, EndGameOrNot, IsDone bool `json:",omitempty"`

	// 牛几
	RobTimes, BetTimes, ExpectWeight, Weight, WeightTimes int

	RobOrNot int `josn:"omitempty"`
}

type NiuNiuPlayer struct {
	cards        []int // 手牌
	weight       int   // 牛几最终结果
	expectWeight int   // 预期的牛
	expectCards  []int // 预期的牌顺序
	doneCards    []int // 显示的牌
	robOrNot     int   // 自由抢庄
	robTimes     int   // 抢庄倍数
	betTimes     int   // 押注倍数

	autoTimes int // 自动托管押注

	// autoTimer *util.Timer

	lastWinGold   int64
	betInAddition bool // 闲家推注

	*service.Player
}

// 已亮牌
func (ply *NiuNiuPlayer) IsDone() bool {
	return ply.doneCards[0] != 0
}

func (ply *NiuNiuPlayer) TryLeave() ErrCode {
	room := ply.Room()
	if room.Status != service.RoomStatusFree {
		return Retry
	}
	return Ok
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
	if ply.RoomObj.IsReady() == false {
		return
	}
	if ply.IsDone() == true {
		return
	}

	room := ply.Room()
	log.Debugf("player %d %v choose tri cards %v", ply.Id, ply.cards, tri)
	if room.Status != service.RoomStatusLook {
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
	room.Broadcast("ChooseTriCards", map[string]any{
		"UId":    ply.Id,
		"Weight": ply.weight,
		"Times":  room.getWeightTimes(ply.weight),
		"Cards":  ply.doneCards,
	})
	room.OnChooseTriCards()
}

func (ply *NiuNiuPlayer) GameOver() {
	ply.initGame()
}

func (ply *NiuNiuPlayer) Bet(times int) {
	if ply.RoomObj.IsReady() == false {
		return
	}

	room := ply.Room()
	log.Debugf("player %d bet %d %d status %d chips %v", ply.Id, ply.betTimes, times, room.Status, ply.Chips())
	if ply.betTimes != -1 {
		return
	}
	if room.Status != service.RoomStatusBet {
		return
	}

	exist := false
	for _, chip := range ply.Chips() {
		if chip == times {
			exist = true
		}
	}

	if exist == false {
		return
	}

	if times == ply.additionChip() {
		ply.betInAddition = true
	}
	// OK
	// ply.times = times
	ply.betTimes = times
	room.Broadcast("Bet", map[string]any{"UId": ply.Id, "Times": times})
	room.OnBet()
}

func (ply *NiuNiuPlayer) ChooseDealer(b bool) {
	if ply.RoomObj.IsReady() == false {
		return
	}
	if ply.robOrNot != -1 {
		return
	}

	// OK
	room := ply.Room()

	ply.robOrNot = 0
	if b == true {
		ply.robOrNot = 1
	}
	room.Broadcast("ChooseDealer", map[string]any{"Code": Ok, "UId": ply.Id, "Ans": b})
	room.OnChooseDealer()
}

func (ply *NiuNiuPlayer) DoubleAndRob(times int) {
	if ply.RoomObj.IsReady() == false {
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
	if room.Status != service.RoomStatusRobDealer {
		return
	}

	// OK
	// ply.times = times
	ply.robTimes = times
	room.Broadcast("DoubleAndRob", map[string]any{"Code": Ok, "UId": ply.Id, "Times": times})
	room.OnDoubleAndRob()
}

func (ply *NiuNiuPlayer) GetUserInfo(self bool) *NiuNiuPlayerInfo {
	info := &NiuNiuPlayerInfo{}
	info.UserInfo = ply.UserInfo
	// info.UId = ply.GetCharObj().Id
	info.SeatId = ply.SeatId

	info.IsReady = ply.RoomObj.IsReady()
	if t := ply.betTimes; t >= 0 {
		info.BetTimes = t
	}

	room := ply.Room()
	info.RobTimes = ply.robTimes
	info.RobOrNot = ply.robOrNot
	info.BetTimes = ply.betTimes

	if ply.RoomObj.IsReady() && len(ply.cards) > 0 && ply.cards[0] > 0 {
		info.IsDone = ply.IsDone()
		if room.CanPlay(OptMingPaiShangZhuang) {
			info.Cards = make([]int, len(ply.cards))
			if self == true {
				copy(info.Cards, ply.cards)
				info.Cards[len(info.Cards)-1] = 0
			}
		}
		if room.Status == service.RoomStatusLook {
			info.Cards = make([]int, len(ply.cards))
			if self == true {
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
	if room.Status == service.RoomStatusBet && ply.RoomObj.IsReady() {
		info.Chips = ply.Chips()
	}
	return info
}

// 坐下
func (ply *NiuNiuPlayer) SitDown(seatId int) {
	room := ply.Room()
	if code := ply.RoomObj.SitDown(seatId); code != Ok {
		return
	}
	// OK
	info := ply.GetUserInfo(false)
	room.Broadcast("SitDown", map[string]any{"Code": Ok, "Info": info})
}

// 固定当庄结束游戏
func (ply *NiuNiuPlayer) EndGame() {
	room := ply.Room()
	if ply != room.dealer {
		return
	}
	if room.isAbleEnd == false {
		return
	}
	room.LimitTimes = room.ExistTimes - 1
	room.GameOver()
}

func (ply *NiuNiuPlayer) additionChip() int {
	room := ply.Room()
	return 0

	var chips = ply.Chips()
	// 闲家推注
	if room.CanPlay(OptXianJiaTuiZhu) == false {
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
	if room := ply.RoomObj.CardRoom(); room != nil {
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
	if ply.RoomObj.IsReady() == false {
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
		if room.GetRoomType() == service.RoomTypeScore {
			chips = []int{1, 2, 3, 4, 5}
		} else {
			// 庄家金币
			chips = []int{1, 2, 4, 8, 20}
		}
	}
	if g := ply.additionChip(); g > 0 {
		chips = append(chips)
	}
	return chips
}

func (ply *NiuNiuPlayer) Replay(messageId string, i interface{}) {
	switch messageId {
	case "SitDown":
		return
	}
	ply.Player.Replay(messageId, i)
}
