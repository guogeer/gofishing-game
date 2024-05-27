package internal

// 2017-6-6 Guogeer
// 内蒙古麻将
import (
	"gofishing-game/internal/cardutils"
	mjutils "gofishing-game/migrate/mahjong/utils"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"

	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
)

type NeimengguMahjong struct {
	room *MahjongRoom
}

func (mj *NeimengguMahjong) OnCreateRoom() {
	room := mj.room
	zipai := []int{60, 70, 80, 90, 100, 110, 120}
	// 不带字牌
	if room.CanPlay(OptBuDaiZiPai) {
		room.CardSet().Remove(zipai...)
	}
	// cardutils.GetCardSystem().Reserve(0)
	if room.CanPlay("seat_2") {
		// 没有筒、条
		for i := 1; i < 10; i++ {
			room.CardSet().Recover(i, 20+i, 40+i)
		}
		room.CardSet().Remove(zipai...)
	}
}

func (mj *NeimengguMahjong) OnEnter(comer *MahjongPlayer) {
}

func (mj *NeimengguMahjong) OnReady() {
	room := mj.room

	room.StartDealCard()
	// 补花
	cards := make([]int, 0, 8)
	flowers := make([]int, 0, 8)
	for isFlower := true; isFlower; {
		isFlower = false
		for i := 0; i < room.NumSeat(); i++ {
			p := room.GetPlayer((i + room.dealer.GetSeatIndex()) % room.NumSeat())

			cards = cards[:0]
			flowers = flowers[:0]
			for _, c := range cardutils.GetAllCards() {
				for k := 0; IsFlower(c) && k < p.handCards[c]; k++ {
					flowers = append(flowers, c)
				}
			}
			if len(flowers) > 0 {
				isFlower = true
				for _, c := range flowers {
					p.handCards[c] = 0
				}
				for k := 0; k < len(flowers); k++ {
					c := room.CardSet().Deal()
					log.Debug("deal card", c)
					p.handCards[c]++
					cards = append(cards, c)

					if p.drawCard != -1 {
						p.drawCard = c
					}
				}

				p.flowers = append(p.flowers, flowers...)
				p.WriteJSON("dealFlower", map[string]any{
					"uid":     p.Id,
					"cards":   cards,
					"flowers": flowers,
				})
				room.Broadcast("dealFlower", map[string]any{
					"uid":     p.Id,
					"flowers": flowers,
				}, p.Id)
			}
		}
	}

	room.dealer.OnDraw()
}

func (hz *NeimengguMahjong) OnWin() {
	room := hz.room

	room.Award()
}

func (mj *NeimengguMahjong) Score(cards []int, melds []mjutils.Meld) (int, int) {
	var pairNum, pair2Num, color int
	for _, c := range cardutils.GetAllCards() {
		pairNum += cards[c] / 2
		pair2Num += cards[c] / 4
		if cards[c] > 0 {
			color = color | int(1<<uint(c/10))
		}
	}
	for _, m := range melds {
		color = color | int(1<<uint(m.Card/10))
	}

	if pairNum == 7 {
		if pair2Num > 0 {
			return LongQiDui, 20
		}
		return QiDui, 10
	}
	// 清一色
	if color&(color-1) == 0 {
		return QingYiSe, 10
	}

	room := mj.room
	winOpt := room.helper.Win(cards, melds)
	// 十三幺
	if winOpt != nil && winOpt.Shisanyao {
		return ShiSanYao, 50
	}
	// 一条龙
	if winOpt != nil && winOpt.Yitiaolong {
		return YiTiaoLong, 10
	}
	return PingHu, 0
}

func (mj *NeimengguMahjong) Award() {
	room := mj.room
	unit, _ := config.Int("room", room.SubId, "unit")

	boom := room.boomPlayer()
	// 有人胡牌
	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		detail := ChipDetail{Seats: 1 << uint(p.GetSeatIndex())}
		if len(room.winPlayers) > 0 {
			winner := room.winPlayers[0]
			if winner.drawCard == -1 && p == boom {
				continue
			}
		}
		for _, kong := range p.kongHistory {
			detail.Operate = kong.Type

			bills := make([]Bill, room.NumSeat())
			for k := 0; k < room.NumSeat(); k++ {
				times := 1
				if kong.Type == mjutils.MeldInvisibleKong {
					times = 2
				}

				// 有人胡牌，放炮不给钱
				bill := &bills[k]
				detail.Seats = 1 << uint(i)
				// 流局倒给钱
				if len(room.winPlayers) == 0 {
					bill = &bills[i]
					detail.Seats = 1 << uint(k)
				}
				if other := room.GetPlayer(k); other != nil && p != other {
					detail.Times = times
					detail.Chip = -int64(detail.Times) * unit
					bill.Details = append(bill.Details, detail)
				}
			}
			room.Billing(bills)
		}
	}

	// 胡牌
	for _, p := range room.winPlayers {
		bills := make([]Bill, room.NumSeat())

		detail := ChipDetail{Seats: 1 << uint(p.GetSeatIndex()), Operate: mjutils.OperateWin}
		addition2 := map[string]int{}

		copyCards := p.copyCards()
		// 够张
		var colors [8]int
		for c, n := range CountAllCards(copyCards, p.melds) {
			if cardutils.IsColorValid(c / 10) {
				colors[c/10] += n
			}
		}
		var counter int
		for color, n := range colors {
			if n > 0 && cardutils.IsColorValid(color) {
				counter++
			}
		}
		log.Debug(colors)
		for color, n := range colors {
			if cardutils.IsColorValid(color) {
				if n > 7 {
					addition2["够张"] = 1
				}
				if n == 0 && counter == 2 {
					addition2["缺门"] = 1
				}
			}
		}

		score, points := mj.Score(copyCards, p.melds)
		// 自摸
		if p.drawCard != -1 {
			addition2["自摸"] = 1
		}
		// 点炮
		if p.drawCard == -1 {
			addition2["接炮"] = 1
		}
		// 海底捞月
		if p.drawCard != -1 && room.CardSet().Count() == -1 {
			addition2["海底捞月"] = 1
		}
		// 庄
		if p == room.dealer {
			addition2["庄"] = 1
		}
		// 杠上开花
		if p.drawCard != -1 && room.kongPlayer == p {
			addition2["杠上花"] = 1
		}
		// 花牌
		if n := len(p.flowers); n > 0 {
			addition2["花牌"] = n
		}

		// 门清
		if CountMeldsByType(p.melds, mjutils.MeldInvisibleKong) == len(p.melds) {
			switch score {
			case QiDui, LongQiDui, ShiSanYao:
			default:
				addition2["门清"] = 1
			}
		}

		// 坎张，只胡一张，凑成顺子
		if opts := p.CheckWin(); len(opts) == 1 {
			c := room.lastCard
			if copyCards[c-1] > 0 && copyCards[c] > 0 && copyCards[c+1] > 0 {
				copyCards[c-1]--
				copyCards[c]--
				copyCards[c+1]--

				if winOpt := room.helper.Win(copyCards, p.melds); winOpt != nil {
					addition2["坎张"] = 1
				}

				copyCards[c-1]++
				copyCards[c]++
				copyCards[c+1]++
			}
		}

		sum := 0
		for _, n := range addition2 {
			sum += n
		}
		detail.Chip = int64(score)
		detail.Points = points
		detail.Times = sum + 4 + points
		detail.Chip = -unit * int64(detail.Times)
		detail.Addition2 = addition2
		if p.drawCard == -1 {
			// 放炮
			boom := room.boomPlayer()
			bill := &bills[boom.GetSeatIndex()]
			bill.Details = append(bill.Details, detail)
		} else {
			// 自摸
			for i := 0; i < room.NumSeat(); i++ {
				other := room.GetPlayer(i)
				if other.isBust || other == p {
					continue
				}
				bill := &bills[other.GetSeatIndex()]
				bill.Details = append(bill.Details, detail)
			}
		}
		room.Billing(bills[:])
	}
}

func (mj *NeimengguMahjong) GameOver() {
}

type NeimengguWorld struct{}

func NewNeimengguWorld() *NeimengguWorld {
	return &NeimengguWorld{}
}

func (w *NeimengguWorld) GetName() string {
	return "nmgmj"
}

func (w *NeimengguWorld) NewRoom(subId int) *roomutils.Room {
	r := NewMahjongRoom(subId)
	r.SetNoPlay(OptYiKouXiang)
	r.SetNoPlay(OptBuDaiZiPai)
	r.SetNoPlay(OptBoom)

	r.SetNoPlay("seat_2")
	r.SetNoPlay("seat_3")
	r.SetPlay("seat_4")

	r.SetPlay(OptShiSanYao)
	r.SetPlay(OptSevenPairs)
	r.localMahjong = &NeimengguMahjong{room: r}
	return r.Room
}

func (w *NeimengguWorld) NewPlayer() *service.Player {
	p := NewMahjongPlayer()
	p.Player = service.NewPlayer(p)
	p.localObj = &NeimengguObj{
		MahjongPlayer: p,
	}
	return p.Player
}

// NeimengguObj 广东麻将玩家逻辑
type NeimengguObj struct {
	*MahjongPlayer
}

func (obj *NeimengguObj) IsAblePong() bool {
	room := obj.Room()
	p := obj.MahjongPlayer
	if !p.IsAblePong() {
		return false
	}
	// 已经听牌了
	if obj.isReadyHand {
		return false
	}
	if !room.CanPlay(OptYiKouXiang) {
		return true
	}
	c := room.lastCard
	obj.handCards[c] -= 2

	other := room.expectDiscardPlayer
	room.expectDiscardPlayer = p
	opts := obj.ReadyHand()
	room.expectDiscardPlayer = other
	obj.handCards[c] += 2
	return len(opts) > 0
}

func (obj *NeimengguObj) GetKongType(c int) int {
	// room := obj.Room()
	p := obj.MahjongPlayer

	typ := p.GetKongType(c)
	/*if typ == -1 ||
		typ == mjutils.MeldBentKong ||
		(typ == mjutils.MeldInvisibleKong && !obj.kaikou()) ||
		!room.CanPlay(OptYiKouXiang) {
		return typ
	}

	// 直杠或已开口暗杠
	cards := obj.handCards
	n := cards[c]
	cards[c] = 0
	if c == p.drawCard {
		cards[c] = 1
	}
	opts := obj.CheckWin()
	cards[c] = n
	if len(opts) == 0 {
		return -1
	}
	*/
	return typ
}

func (obj *NeimengguObj) OnPong() {
	room := obj.Room()
	if room.CanPlay(OptYiKouXiang) && !obj.isReadyHand {
		obj.forceReadyHand = true
	}
}

func (obj *NeimengguObj) OnKong() {
	room := obj.Room()
	meld := obj.lastMeld()
	if room.CanPlay(OptYiKouXiang) && meld.Type != mjutils.MeldInvisibleKong && !obj.isReadyHand {
		obj.forceReadyHand = true
	}
}
