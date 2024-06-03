package lottery

// 扎金花押注场
import (
	"container/list"
	"gofishing-game/games/migrate/internal/cardrule"
	"gofishing-game/internal/cardutils"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
	"math/rand"

	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
	"github.com/guogeer/quasar/utils"
	"github.com/guogeer/quasar/utils/randutils"
)

var gZhajinhuaHelper = cardrule.NewZhajinhuaHelper((*bairenniuniuWorld)(nil).GetName())
var gZhajinhuaMultiples = []int{
	0,
	1, // 散牌
	1, // 对子
	2, // 顺子
	3, // 金花
	4, // 顺金
	5, // 豹子
}

func init() {
	var cards []int
	for color := 0; color < 4; color++ {
		for value := 2; value <= 14; value++ {
			c := (color << 4) | value
			cards = append(cards, c)
		}
	}

	w := &BairenzhajinhuaWorld{}
	cardutils.AddCardSystem(w.GetName(), cards)
	service.AddWorld(w)
	AddHandlers(w.GetName())
}

type bairenzhajinhuaHelper struct {
}

func (h *bairenzhajinhuaHelper) count(cards []int) (int, int) {
	helper := gZhajinhuaHelper
	typ, _ := helper.GetType(cards)
	return typ, gZhajinhuaMultiples[typ]
}

func (h bairenzhajinhuaHelper) Less(fromCards, toCards []int) bool {
	helper := gZhajinhuaHelper
	return helper.Less(fromCards[:], toCards[:])
}

type bairenzhajinhua struct {
	room *lotteryRoom
}

func (ent *bairenzhajinhua) OnEnter(player *lotteryPlayer) {
}

func (ent *bairenzhajinhua) winPrizePool(cards []int) float64 {
	return 0.0
}

func (ent *bairenzhajinhua) StartDealCard() {
	room := ent.room
	helper := gZhajinhuaHelper

	var samples []int
	config.Scan("lottery", room.SubId, "cardSamples", &samples)
	if len(samples) > 0 {
		start := rand.Intn(len(room.deals))
		for i := range room.deals {
			k := (start + i) % len(room.deals)

			cards := room.deals[k].Cards
			typ := randutils.Index(samples)
			// log.Debug("cheat type", typ)
			table := room.CardSet().GetRemainingCards()
			if a := helper.Cheat(typ, table); a != nil {
				copy(cards, a)
				room.CardSet().Cheat(a...)
			}
		}
	}
}

func (ent *bairenzhajinhua) Cheat(multiples int) []int {
	room := ent.room
	helper := gZhajinhuaHelper
	allowTypes := make([]int, 0, 8)
	log.Debug("cheat multiples", multiples)

	values := sortArrayValues(gZhajinhuaMultiples)
	for i := len(values) - 1; i > 0; i-- {
		current := values[i]
		allowTypes = allowTypes[:0]
		for t, m := range gZhajinhuaMultiples {
			if current <= multiples && current == m {
				allowTypes = append(allowTypes, t)
			}
		}
		if len(allowTypes) > 0 {
			t := allowTypes[rand.Intn(len(allowTypes))]
			table := room.CardSet().GetRemainingCards()
			if cards := helper.Cheat(t, table); cards != nil {
				room.CardSet().Cheat(cards...)
				log.Debug("current type", t)
				cardutils.Print(cards)
				return cards
			}
		}
	}
	log.Warnf("bairenzhajinhua cheat cards by multiples %d fail", multiples)
	return nil
}

type BairenzhajinhuaWorld struct{}

func (w *BairenzhajinhuaWorld) NewRoom(subId int) *roomutils.Room {
	room := &lotteryRoom{
		robSeat:         roomutils.NoSeat,
		betAreas:        make([]int64, 4),
		dealerQueue:     list.New(),
		helper:          &bairenzhajinhuaHelper{},
		multipleSamples: []int{0, 0, 310000, 790000, 990000, 1000000},
	}
	room.Room = roomutils.NewRoom(subId, room)
	room.lotteryGame = &bairenzhajinhua{
		room: room,
	}

	for i := 0; i < len(room.last); i++ {
		room.last[i] = -1
	}
	deals := make([]lotteryDeal, 5)
	for i := range deals {
		deals[i].Cards = make([]int, 3)
	}
	room.deals = deals

	// room.StartGame()
	// 定时同步
	utils.NewTimer(room.OnTime, syncTime)
	return room.Room
}

func (w *BairenzhajinhuaWorld) GetName() string {
	return "brzjh"
}

func (w *BairenzhajinhuaWorld) NewPlayer() *service.Player {
	p := &lotteryPlayer{}
	p.Player = service.NewPlayer(p)
	p.betAreas = make([]int64, 4)
	return p.Player
}
