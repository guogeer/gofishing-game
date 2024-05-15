package shisanshui

import (
	"gofishing-game/internal/errcode"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
	"third/cardutil"
	"time"

	"github.com/guogeer/quasar/log"
)

// 玩家信息
type ShisanshuiUserInfo struct {
	service.UserInfo

	SeatId  int
	IsReady bool `json:",omitempty"`

	Cards, SplitCards []int                       `json:",omitempty"`
	IsSplitCards      bool                        `json:",omitempty"`
	ResultSet         []cardutil.ShisanshuiResult `json:",omitempty"`
}

type ShisanshuiPlayer struct {
	*service.Player

	splitCards, cards []int
	resultSet         []cardutil.ShisanshuiResult
}

func (ply *ShisanshuiPlayer) TryLeave() errcode.Error {
	room := ply.Room()
	if room.Status != service.RoomStatusFree && roomutils.GetRoomObj(ply.Player).IsReady() {
		return Retry
	}
	return Ok
}

func (ply *ShisanshuiPlayer) initGame() {
	for i := range ply.cards {
		ply.cards[i] = 0
	}
	for i := range ply.splitCards {
		ply.splitCards[i] = 0
	}
	ply.resultSet = nil
}

func (ply *ShisanshuiPlayer) GameOver() {
	ply.initGame()
}

func (ply *ShisanshuiPlayer) GetUserInfo(self bool) *ShisanshuiUserInfo {
	info := &ShisanshuiUserInfo{}
	info.UserInfo = ply.UserInfo
	info.SeatId = ply.GetSeatIndex()
	info.IsReady = roomutils.GetRoomObj(ply.Player).IsReady()

	if self == true {
		info.Cards = ply.cards
		if ply.splitCards[0] > 0 {
			info.SplitCards = ply.splitCards
		}
		info.ResultSet = ply.resultSet
	} else if ply.splitCards[0] > 0 {
		info.IsSplitCards = true
	}
	return info
}

func (ply *ShisanshuiPlayer) GetSortedCards() []int {
	cards := make([]int, 0, 16)
	for c, n := range ply.cards {
		for k := 0; k < n; k++ {
			cards = append(cards, c)
		}
	}
	return cards
}

func (ply *ShisanshuiPlayer) SplitCards(cards []int) {
	log.Debugf("player %d split cards %v", ply.Id, cards)
	if roomutils.GetRoomObj(ply.Player).IsReady() == false {
		return
	}
	if len(cards) != len(ply.cards) || ply.splitCards[0] > 0 {
		return
	}
	table := make(map[int]int)
	for _, c := range cards {
		table[c]++
	}
	for _, c := range ply.cards {
		table[c]--
	}
	for _, n := range table {
		if n != 0 {
			return
		}
	}

	code := Ok
	room := ply.Room()
	if room.helper.IsValid(cards) == false {
		code = ShisanshuiInvalidCards
	}

	typ := 0
	if code == Ok {
		_, typ = room.helper.GetSpecialType(cards)
	}

	ply.WriteJSON("SplitCards", map[string]any{"Code": code, "Msg": code.String(), "UId": ply.Id, "Cards": cards, "SpecialType": typ})
	if code != Ok {
		return
	}

	// OK
	log.Debugf("player %d split cards %v", ply.Id, cards)
	room.Broadcast("SplitCards", map[string]any{"Code": Ok, "UId": ply.Id, "SpecialType": typ}, ply.Id)

	copy(ply.splitCards, cards)
	ply.StopTimer(service.TimerEventOperate)

	room.OnSplitCards()
}

func (ply *ShisanshuiPlayer) Timeout(fn func(), d time.Duration) {
	room := ply.Room()
	if !room.IsTypeScore() {
		ply.AddTimer(service.TimerEventOperate, fn, d)
	}
}

func (ply *ShisanshuiPlayer) Room() *ShisanshuiRoom {
	if room := roomutils.GetRoomObj(ply.Player).CustomRoom(); room != nil {
		return room.(*ShisanshuiRoom)
	}
	return nil
}

func GetPlayer(id int) *ShisanshuiPlayer {
	if p := service.GetPlayer(id); p != nil {
		return p.GameAction.(*ShisanshuiPlayer)
	}
	return nil
}
