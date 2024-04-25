package internal

import (
	"container/list"
	// "github.com/guogeer/quasar/log"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
	"math/rand"
	"third/cardutil"

	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/randutil"
	"github.com/guogeer/quasar/util"
)

var (
	erbagangCardValues = map[int]int{
		21:  1,
		22:  2,
		23:  3,
		24:  4,
		25:  5,
		26:  6,
		27:  7,
		28:  8,
		29:  9,
		100: 30, // 中
		110: 20, // 发
		120: 10, // 白
	}
	erbagangTypeValues = map[int]int{
		0:  0,
		1:  1,
		2:  2,
		3:  3,
		4:  4,
		5:  5,
		6:  6,
		7:  7,
		8:  8,
		9:  9,
		22: 30,
		28: 20,
	}
)

// 0~9、点数；22、豹子；28、天杠
type erbagangHelper struct{}

func (h *erbagangHelper) count(cards []int) (int, int) {
	if cards[0] == 22 && cards[1] == 28 {
		return 28, 2
	} else if cards[0] == 28 && cards[1] == 22 {
		return 28, 2
	} else if cards[0] == cards[1] {
		return 22, 3
	}
	cValues := erbagangCardValues
	return (cValues[cards[0]] + cValues[cards[1]]) % 10, 1
}

func (h erbagangHelper) Less(fromCards, toCards []int) bool {
	typ1, _ := h.count(fromCards)
	typ2, _ := h.count(toCards)

	tValues := erbagangTypeValues
	cValues := erbagangCardValues
	// log.Debug(fromCards, toCards, tValues[typ1], tValues[typ2])
	if typ1 != typ2 {
		return tValues[typ1] < tValues[typ2]
	}
	max := func(cards []int) int {
		mc := 0
		for _, c := range cards {
			if cValues[mc] < cValues[c] {
				mc = c
			}
		}
		return mc
	}

	max1 := max(fromCards)
	max2 := max(toCards)
	return cValues[max1] < cValues[max2]
}

type erbagang struct {
	room *entertainmentRoom
}

func (ent *erbagang) OnEnter(player *entertainmentPlayer) {
}

func (ent *erbagang) winPrizePool(cards []int) float64 {
	subId := ent.room.SubId
	if cards[0] != cards[1] {
		return 0.0
	}
	var a []float64
	config.Scan("entertainment", subId, "CTPrizePoolPercent", &a)
	if cards[0] == 100 && len(a) > 0 {
		return a[0]
	}
	if cards[0] == 110 && len(a) > 1 {
		return a[1]
	}
	if cards[0] == 120 && len(a) > 2 {
		return a[2]
	}
	return 0.0
}

func (ent *erbagang) StartDealCard() {
	room := ent.room

	var samples []int
	config.Scan("entertainment", room.SubId, "CardSamples", &samples)

	start := rand.Intn(len(room.deals))
	for i := range room.deals {
		k := (start + i) % len(room.deals)

		typ := RandInArray(samples)
		// TODO
		if typ == -1 {
			break
		}
		cards := room.deals[k].Cards
		// log.Debug("cheat type", typ)
		table := room.CardSet().GetRemainingCards()

		var res []int
		switch typ {
		case 0: // 1-9点
			var single []int
			for _, c := range cardutil.GetAllCards() {
				if table[c] > 0 {
					single = append(single, c)
				}
			}
			if len(single) > 1 {
				randutil.ShuffleN(single, 2)
				res = single[:2]
			}
		case 1: // 豹子
			var pairs []int
			for _, c := range []int{21, 22, 23, 24, 25, 26, 27, 28, 29} {
				if table[c] > 1 {
					pairs = append(pairs, c)
				}
			}
			if len(pairs) > 0 {
				pair := pairs[rand.Intn(len(pairs))]
				res = []int{pair, pair}
			}
		case 2: // 二八杠
			if table[22] > 0 && table[28] > 0 {
				res = []int{22, 28}
			}
		case 3, 4, 5: // 中、发、白
			c := []int{0, 0, 0, 100, 110, 120}[typ]
			if table[c] > 1 {
				res = []int{c, c}
			}
		}

		if len(res) > 0 {
			copy(cards, res)
			room.CardSet().Cheat(res...)
		}
	}
}

func (ent *erbagang) Cheat(multiples int) []int {
	room := ent.room
	remainingCards := room.CardSet().GetRemainingCards()
	validCards := make([]int, 0, 64)
	// tempMultiples := multiples

	var cards []int
	if multiples > 0 {
		validCards = validCards[:0]
		for _, c := range cardutil.GetAllCards() {
			if remainingCards[c] > 0 {
				validCards = append(validCards, c)
			}
		}
		if len(validCards) > 1 {
			randutil.ShuffleN(validCards, 2)
			cards = []int{validCards[0], validCards[1]}
		}
	}
	if multiples >= 2 && remainingCards[22] > 0 && remainingCards[28] > 0 {
		cards = []int{22, 28}
		if rand.Intn(2) == 0 {
			cards = []int{28, 22}
		}
	}

	if multiples >= 3 {
		validCards = validCards[:0]
		for _, c := range []int{21, 22, 23, 24, 25, 26, 27, 28, 29} {
			if remainingCards[c] > 1 {
				validCards = append(validCards, c)
			}
		}
		if len(validCards) > 0 {
			c := validCards[rand.Intn(len(validCards))]
			cards = []int{c, c}
		}
	}

	room.CardSet().Cheat(cards...)
	// log.Debug("//////////////////", tempMultiples, cards)
	return cards
}

type ErbagangWorld struct {
}

func (w *ErbagangWorld) NewRoom(id, subId int) *service.Room {
	room := &entertainmentRoom{
		robSeat:         roomutils.NoSeat,
		dealerQueue:     list.New(),
		chips:           []int64{100, 500, 1000, 5000, 10000},
		helper:          &erbagangHelper{},
		multipleSamples: []int{0, 330000, 870000, 990000, 1000000},
	}
	room.Room = service.NewRoom(id, subId, room)
	room.entertainmentGame = &erbagang{
		room: room,
	}
	room.userAreaNum = 3
	tags, _ := config.String("Room", subId, "Tags")
	if config.IsPart(tags, "area_4") {
		room.userAreaNum = 4
	}

	for i := 0; i < len(room.last); i++ {
		room.last[i] = -1
	}
	deals := make([]entertainmentDeal, room.userAreaNum+1)
	for i := range deals {
		deals[i].Cards = make([]int, 2)
	}
	room.deals = deals
	room.betAreas = make([]int64, room.userAreaNum)

	// room.StartGame()
	// 定时同步
	util.NewTimer(room.OnTime, syncTime)
	return room.Room
}

func (w *ErbagangWorld) GetName() string {
	return "ebg"
}

func (w *ErbagangWorld) NewPlayer() *service.Player {
	p := &entertainmentPlayer{}
	p.Player = service.NewPlayer(p)
	p.betAreas = make([]int64, 3)
	return p.Player
}
