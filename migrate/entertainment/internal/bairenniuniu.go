package internal

import (
	"container/list"
	"gofishing-game/migrate/entertainment/utils"
	"gofishing-game/service"
	"math/rand"

	"github.com/guogeer/quasar/log"
	"github.com/guogeer/quasar/utils"
)

var gNiuNiuHelper = utils.NewNiuNiuHelper()
var gNiuNiuMultiples = []int{1, 1, 1, 1, 1, 1, 1, 1, 2, 2, 3,
	0, // 五小牛
	5, // 炸弹牛
	4, // 五花牛
	4, // 四花牛
}

func init() {
	helper := gNiuNiuHelper
	helper.SetOption(utils.NNZhaDanNiu)
	helper.SetOption(utils.NNWuHuaNiu)
	helper.SetOption(utils.NNSiHuaNiu)
}

type bairenniuniuHelper struct{}

func (h bairenniuniuHelper) Less(fromCards, toCards []int) bool {
	helper := gNiuNiuHelper
	return helper.Less(fromCards, toCards)
}

func (h *bairenniuniuHelper) count(cards []int) (int, int) {
	helper := gNiuNiuHelper
	typ, _ := helper.Weight(cards)
	multiples := gNiuNiuMultiples[typ]
	return typ, multiples
}

type bairenniuniu struct {
	room *entertainmentRoom
}

func (ent *bairenniuniu) OnEnter(player *entertainmentPlayer) {
}

func (ent *bairenniuniu) StartDealCard() {
}

func (ent *bairenniuniu) winPrizePool(cards []int) float64 {
	return 0.0
}

func (ent *bairenniuniu) Cheat(multiples int) []int {
	room := ent.room
	helper := gNiuNiuHelper
	allowTypes := make([]int, 0, 8)
	log.Debug("cheat multiples", multiples)

	values := sortArrayValues(gNiuNiuMultiples)
	for i := len(values) - 1; i > 0; i-- {
		current := values[i]
		allowTypes = allowTypes[:0]
		for t, m := range gNiuNiuMultiples {
			if current <= multiples && current == m {
				allowTypes = append(allowTypes, t)
			}
		}
		if len(allowTypes) > 0 {
			t := allowTypes[rand.Intn(len(allowTypes))]
			table := room.CardSet().GetRemainingCards()
			if cards := helper.Cheat(t, table); cards != nil {
				room.CardSet().Cheat(cards...)
				return cards
			}
		}
	}

	log.Warnf("bairenniuniu cheat cards by multiples %d fail", multiples)
	return nil
}

type BairenniuniuWorld struct {
	helper *utils.NiuNiuHelper
}

func (w *BairenniuniuWorld) NewRoom(id, subId int) *roomutils.Room {
	room := &entertainmentRoom{
		robSeat:         roomutils.NoSeat,
		betAreas:        make([]int64, 4),
		dealerQueue:     list.New(),
		helper:          &bairenniuniuHelper{},
		multipleSamples: []int{0, 0, 330000, 870000, 980000, 1000000},
	}
	room.Room = roomutils.NewRoom(id, subId, room)

	room.entertainmentGame = &bairenniuniu{
		room: room,
	}

	for i := 0; i < len(room.last); i++ {
		room.last[i] = -1
	}
	deals := make([]entertainmentDeal, 5)
	for i := range deals {
		deals[i].Cards = make([]int, 5)
	}
	room.deals = deals
	utils.NewTimer(room.OnTime, syncTime)
	return room.Room
}

func (w *BairenniuniuWorld) GetName() string {
	return "brnn"
}

func (w *BairenniuniuWorld) NewPlayer() *service.Player {
	p := &entertainmentPlayer{}
	p.Player = service.NewPlayer(p)
	p.betAreas = make([]int64, 4)
	return p.Player
}
