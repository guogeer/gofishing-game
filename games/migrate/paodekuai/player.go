package paodekuai

import (
	"gofishing-game/games/migrate/internal/cardrule"
	"gofishing-game/internal/errcode"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
	"time"

	"github.com/guogeer/quasar/v2/log"
	"github.com/guogeer/quasar/v2/utils"
)

// 玩家信息
type PaodekuaiUserInfo struct {
	service.UserInfo

	SeatIndex int   `json:"seatIndex,omitempty"`
	CardNum   int   `json:"cardNum,omitempty"`
	BoomTimes int   `json:"boomTimes,omitempty"`
	Cards     []int `json:"cards,omitempty"`
	Action    []int `json:"action,omitempty"` // 最近打出去的牌
	// 准备
	IsReady bool `json:"isReady,omitempty"`
}

type PaodekuaiPlayer struct {
	*service.Player

	cards            []int // 手牌
	action           []int // 本轮打出去的牌
	isAutoPlay       bool
	boomTimes        int
	forceDiscardCard int // 必须打出去的牌
	discardNum       int

	winTimes       int   // 赢的局数
	maxWinGold     int64 // 最大赢的金币
	totalBoomTimes int   // 总的炸弹数
	operateTimer   *utils.Timer
}

func (ply *PaodekuaiPlayer) TryEnter() errcode.Error {
	return nil
}

func (ply *PaodekuaiPlayer) BeforeEnter() {
}

func (ply *PaodekuaiPlayer) AfterEnter() {
}

func (ply *PaodekuaiPlayer) BeforeLeave() {
}

func (ply *PaodekuaiPlayer) TryLeave() errcode.Error {
	room := ply.Room()
	if room.Status != 0 {
		return errcode.Retry
	}
	return nil
}

func (ply *PaodekuaiPlayer) initGame() {
	for i := 0; i < len(ply.cards); i++ {
		ply.cards[i] = 0
	}
	ply.action = nil
	ply.discardNum = 0
	ply.boomTimes = 0
	ply.isAutoPlay = false
}

func (ply *PaodekuaiPlayer) GameOver() {
	ply.initGame()
}

func (ply *PaodekuaiPlayer) GetUserInfo(self bool) *PaodekuaiUserInfo {
	info := &PaodekuaiUserInfo{}
	info.UserInfo = ply.UserInfo
	// info.UId = ply.GetCharObj().Id
	info.SeatIndex = ply.GetSeatIndex()
	info.IsReady = roomutils.GetRoomObj(ply.Player).IsReady()
	info.Cards = ply.GetSortedCards()
	info.BoomTimes = ply.boomTimes
	info.Action = ply.action
	return info
}

func (ply *PaodekuaiPlayer) GetSortedCards() []int {
	cards := make([]int, 0, 16)
	for c, n := range ply.cards {
		for k := 0; k < n; k++ {
			cards = append(cards, c)
		}
	}
	return cards
}

// 自动出牌或过
func (ply *PaodekuaiPlayer) AutoPlay() {
	isAuto := ply.isAutoPlay
	d := maxOperateTime
	room := ply.Room()
	// 必须管时，轮到玩家出牌，要不起，自动过
	cards := ply.GetSortedCards()
	typ, _, _ := room.helper.GetType(cards)

	var action []int
	if other := room.discardPlayer; other == nil {
		// 首轮黑桃三先出
		if c := ply.forceDiscardCard; c > 0 {
			action = []int{c}
		} else {
			// 最后一轮，玩家自动出牌
			if typ != cardrule.PaodekuaiNone &&
				typ != cardrule.PaodekuaiSidaisan {
				isAuto = true
				action = cards
			} else { // 没有其他玩家出牌
				var value int
				for action == nil {
					for c, n := range ply.cards {
						if n > 0 && value == room.helper.Value(c) {
							action = []int{c}
							break
						}
					}
					value++
				}
			}
		}
	} else {
		// 如果能全部出完，优先自动出完
		if typ != cardrule.PaodekuaiSidaisan && room.helper.Less(other.action, cards) {
			isAuto = true
			action = cards
		} else {
			ans := room.helper.Match(cards, other.action)
			if len(ans) > 0 {
				action = ans
			} else if room.CanPlay(OptBixuguan) {
				isAuto = true // 必须管，要不起
			}
		}
	}

	log.Debug("time out", isAuto)
	room.autoTime = time.Now().Add(maxOperateTime)
	if isAuto {
		d = maxAutoTime
	} else {
		if room.IsTypeScore() {
			return
		}
	}

	utils.StopTimer(ply.operateTimer)
	if len(action) == 0 {
		ply.operateTimer = ply.TimerGroup.NewTimer(ply.Pass, d)
	} else {
		ply.operateTimer = ply.TimerGroup.NewTimer(func() { ply.Discard(action) }, d)
	}
}

func (ply *PaodekuaiPlayer) Discard(cards []int) {
	log.Debugf("player %d discard %v", ply.Id, cards)
	room := ply.Room()
	if ply != room.expectDiscardPlayer {
		return
	}
	// 判断牌是否有效、数量足够
	var m = make(map[int]int)
	for _, c := range cards {
		if !room.CardSet().IsCardValid(c) {
			return
		}
		m[c]++
	}
	// 首轮必须出黑桃三
	if c := ply.forceDiscardCard; c > 0 {
		if n, _ := m[c]; n == 0 {
			return
		}
	}
	for c, n := range m {
		if ply.cards[c] < n {
			return
		}
	}
	typ, _, _ := room.helper.GetType(cards)
	if typ == cardrule.PaodekuaiNone {
		return
	}

	total := 0
	for _, n := range ply.cards {
		total += n
	}
	helper := room.helper
	if other := room.discardPlayer; other == nil && total != len(cards) && !helper.Sandaidui {
		switch typ {
		case cardrule.PaodekuaiSandai0, cardrule.PaodekuaiSandai1:
			return
		case cardrule.PaodekuaiFeiji:
			if len(cards)%5 != 0 {
				return
			}
		}
	}

	if other := room.discardPlayer; other != nil && !room.helper.Less(other.action, cards) {
		return
	}
	// 下家报单必须出最大的牌
	next := room.GetPlayer((ply.GetSeatIndex() + 1) % room.NumSeat())
	if len(cards) == 1 && len(next.GetSortedCards()) == 1 {
		maxCard := room.helper.MaxCard(ply.GetSortedCards())
		if room.helper.Value(cards[0]) != room.helper.Value(maxCard) {
			return
		}
	}
	// OK
	log.Debugf("player %d discard ok %v", ply.Id, cards)

	ply.discardNum++
	ply.forceDiscardCard = 0
	/*if typ == cardutils.PaodekuaiZhaDan {
		ply.boomTimes++
		ply.totalBoomTimes++
	}
	*/
	// utils.StopTimer(ply.autoTimer)
	utils.StopTimer(ply.operateTimer)
	data := map[string]interface{}{"cards": cards, "uid": ply.Id}
	if cardNum := total - len(cards); cardNum < 3 {
		data["cardNum"] = cardNum
	}
	room.Broadcast("discard", data)

	for _, c := range cards {
		ply.cards[c]--
	}
	ply.action = cards
	room.discardPlayer = ply

	room.Turn()
}

func (ply *PaodekuaiPlayer) Pass() {
	log.Debugf("player %d pass", ply.Id)
	room := ply.Room()
	if room.expectDiscardPlayer != ply {
		return
	}

	other := room.discardPlayer
	if other == nil {
		return
	}
	// 可以大起，必须管的时候不能过
	ans := room.helper.Match(ply.GetSortedCards(), other.action)
	if len(ans) > 0 && room.CanPlay(OptBixuguan) {
		return
	}

	// OK
	ply.action = nil
	// utils.StopTimer(ply.autoTimer)
	utils.StopTimer(ply.operateTimer)
	room.Broadcast("pass", map[string]interface{}{"uid": ply.Id})
	room.Turn()
}

func (ply *PaodekuaiPlayer) Room() *PaodekuaiRoom {
	if room := roomutils.GetRoomObj(ply.Player).CustomRoom(); room != nil {
		return room.(*PaodekuaiRoom)
	}
	return nil
}

func (ply *PaodekuaiPlayer) Replay(messageId string, i interface{}) {
	switch messageId {
	case "startDealCard":
		room := ply.Room()
		data := i.(map[string]interface{})
		all := make([][]int, room.NumSeat())
		for k := 0; k < room.NumSeat(); k++ {
			other := room.GetPlayer(k)
			all[k] = other.GetSortedCards()
		}
		data["all"] = all
		defer func() { delete(data, "all") }()
	}
	ply.Player.Replay(messageId, i)

}

func (ply *PaodekuaiPlayer) GetSeatIndex() int {
	return roomutils.GetRoomObj(ply.Player).GetSeatIndex()
}
