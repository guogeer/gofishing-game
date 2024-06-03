package lottery

import (
	"container/list"
	"gofishing-game/games/migrate/internal/cardrule"
	"gofishing-game/internal/cardutils"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
	"math/rand"

	"github.com/guogeer/quasar/log"
	"github.com/guogeer/quasar/utils"
)

var gNiuNiuHelper = cardrule.NewNiuNiuHelper((*bairenniuniuWorld)(nil).GetName())
var gNiuNiuMultiples = []int{1, 1, 1, 1, 1, 1, 1, 1, 2, 2, 3,
	0, // 五小牛
	5, // 炸弹牛
	4, // 五花牛
	4, // 四花牛
}

func init() {
	helper := gNiuNiuHelper
	helper.SetOption(cardrule.NNZhaDanNiu)
	helper.SetOption(cardrule.NNWuHuaNiu)
	helper.SetOption(cardrule.NNSiHuaNiu)

	var cards []int
	for color := 0; color < 4; color++ {
		for value := 2; value <= 14; value++ {
			c := (color << 4) | value
			cards = append(cards, c)
		}
	}

	w := &bairenniuniuWorld{}
	cardutils.AddCardSystem(w.GetName(), cards)
	service.AddWorld(w)
	AddHandlers(w.GetName())
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
	room *lotteryRoom
}

func (ent *bairenniuniu) OnEnter(player *lotteryPlayer) {
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

type bairenniuniuWorld struct{}

func (w *bairenniuniuWorld) NewRoom(subId int) *roomutils.Room {
	room := &lotteryRoom{
		robSeat:         roomutils.NoSeat,
		betAreas:        make([]int64, 4),
		dealerQueue:     list.New(),
		helper:          &bairenniuniuHelper{},
		multipleSamples: []int{0, 0, 330000, 870000, 980000, 1000000},
	}
	room.Room = roomutils.NewRoom(subId, room)

	room.lotteryGame = &bairenniuniu{
		room: room,
	}

	for i := 0; i < len(room.last); i++ {
		room.last[i] = -1
	}
	deals := make([]lotteryDeal, 5)
	for i := range deals {
		deals[i].Cards = make([]int, 5)
	}
	room.deals = deals
	utils.NewTimer(room.OnTime, syncTime)
	return room.Room
}

func (w *bairenniuniuWorld) GetName() string {
	return "brnn"
}

func (w *bairenniuniuWorld) NewPlayer() *service.Player {
	p := &lotteryPlayer{}
	p.Player = service.NewPlayer(p)
	p.betAreas = make([]int64, 4)
	return p.Player
}
