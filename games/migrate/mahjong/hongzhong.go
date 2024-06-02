package mahjong

// 2017-6-6 Guogeer
// 转转麻将
import (
	"gofishing-game/games/migrate/internal/cardrule"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
)

type HongzhongMahjong struct {
	room *MahjongRoom

	ghostCard int
	horses    []int // 马牌
}

func (hz *HongzhongMahjong) OnCreateRoom() {
	room := hz.room
	// 默认没有字牌
	zipai := []int{60, 70, 80, 90, 100, 110, 120}
	room.CardSet().Remove(zipai...)
	room.CardSet().Recover(100)
	// 两个人玩的时候只有万和字
	if room.CanPlay("seat_2") {
		// 没有筒、条
		for i := 1; i < 10; i++ {
			room.CardSet().Remove(20+i, 40+i)
		}
		room.CardSet().Recover(zipai...)
	}
}

func (hz *HongzhongMahjong) OnEnter(comer *MahjongPlayer) {
	data := map[string]any{
		"card":  hz.ghostCard,
		"ghost": hz.getAnyCards(),
	}
	comer.SetClientValue("localMahjong", data)

}

func (hz *HongzhongMahjong) OnReady() {
	room := hz.room
	// 预留马牌
	hz.horses = hz.horses[:0]
	for i := 0; i < hz.countHorse(); i++ {
		hz.horses = append(hz.horses, 0)
	}

	room.StartDealCard()
	room.Broadcast("chooseGhostCard", map[string]any{
		"card":  hz.ghostCard,
		"ghost": hz.getAnyCards(),
	})
	room.dealer.OnDraw()
}

func (hz *HongzhongMahjong) countHorse() int {
	room := hz.room

	var horseNum = 2
	if room.CanPlay(OptBuyHorse2) {
		horseNum = 2
	} else if room.CanPlay(OptBuyHorse3) {
		horseNum = 3
	} else if room.CanPlay(OptBuyHorse4) {
		horseNum = 4
	}
	return horseNum
}

func (hz *HongzhongMahjong) drawHorse(horseNum int) []int {
	room := hz.room

	var horses []int
	var last = room.lastCard
	for i := 0; i < horseNum; i++ {
		c := room.CardSet().Deal()
		if c == -1 {
			break
		}
		last = c
		horses = append(horses, c)
	}
	for i := len(horses); i < horseNum; i++ {
		horses = append(horses, last)
	}
	return horses
}

func (hz *HongzhongMahjong) OnWin() {
	room := hz.room
	horseNum := hz.countHorse()

	hz.horses = hz.drawHorse(horseNum)
	if horseNum > 0 && hz.horses[0] > 0 {
		type Winner struct {
			SeatIndex int
			Horses    []int
		}
		var winners []Winner
		for _, p := range room.winPlayers {
			obj := p.localObj.(*HongzhongObj)
			winners = append(winners, Winner{
				SeatIndex: p.GetSeatIndex(),
				Horses:    obj.winHorse(),
			})
		}

		room.Broadcast("buyHorse", map[string]any{
			"horses":  hz.horses,
			"winners": winners,
		})
	}
	room.Award()
}

func (hz *HongzhongMahjong) Award() {
	room := hz.room
	unit := room.Unit()

	// 有人胡牌
	for i := 0; len(room.winPlayers) > 0 && i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		detail := ChipDetail{Seats: 1 << uint(p.GetSeatIndex())}
		for _, kong := range p.kongHistory {
			detail.Operate = kong.Type

			bills := make([]Bill, room.NumSeat())
			switch kong.Type {
			case cardrule.MeldInvisibleKong, cardrule.MeldBentKong:
				times := 1
				// 暗杠
				if kong.Type == cardrule.MeldInvisibleKong {
					times = 2
				}
				for k := 0; k < room.NumSeat(); k++ {
					bill := &bills[k]
					if other := room.GetPlayer(k); other != nil && p != other {
						detail.Times = times
						detail.Chip = -int64(detail.Times) * unit
						bill.Details = append(bill.Details, detail)
					}
				}
			case cardrule.MeldStraightKong:
				// 直杠
				bill := &bills[kong.other.GetSeatIndex()]
				detail.Times = 3
				detail.Chip = -int64(detail.Times) * unit
				bill.Details = append(bill.Details, detail)
			}
			room.Billing(bills)
		}
	}

	// 胡牌
	for _, p := range room.winPlayers {
		bills := make([]Bill, room.NumSeat())

		obj := p.localObj.(*HongzhongObj)
		detail := ChipDetail{Seats: 1 << uint(p.GetSeatIndex()), Operate: cardrule.OperateWin}
		// 玩家中马
		winHorses := obj.winHorse()
		addition2 := map[string]int{}

		if n := len(winHorses); n > 0 {
			addition2["马"] = 2 * n
		}

		// 自摸
		if p.drawCard != -1 {
			addition2["自摸"] = 2
		} else {
			addition2["接炮"] = 1
		}

		sum := 0
		for _, n := range addition2 {
			sum += n
		}
		detail.Times = sum
		detail.Chip = -unit * int64(sum)
		detail.Addition2 = addition2
		if p.drawCard == -1 {
			// 放炮
			boom := room.boomPlayer()
			// 抢杠胡出三家
			if room.kongPlayer != nil && room.kongPlayer != p && room.discardPlayer == nil {
				n := room.NumSeat() - 1
				detail.Addition2["抢杠胡"] = n
				detail.Times *= n
				detail.Chip *= int64(n)
			}
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

func (hz *HongzhongMahjong) GameOver() {
	hz.ghostCard = -1
}

// 癞子牌
func (hz *HongzhongMahjong) getAnyCards() []int {
	return []int{100}
}

type HongzhongWorld struct{}

func NewHongzhongWorld() *HongzhongWorld {
	return &HongzhongWorld{}
}

func (w *HongzhongWorld) GetName() string {
	return "hzmj"
}

func (w *HongzhongWorld) NewRoom(subId int) *roomutils.Room {
	r := NewMahjongRoom(subId)
	r.SetNoPlay(OptSevenPairs)

	r.SetPlay(OptBuyHorse2)
	r.SetNoPlay(OptBuyHorse3)
	r.SetNoPlay(OptBuyHorse4)

	r.SetNoPlay("seat_2")
	r.SetNoPlay("seat_3")
	r.SetPlay("seat_4")

	r.SetPlay(OptAbleRobKong)
	r.localMahjong = &HongzhongMahjong{room: r, ghostCard: -1}
	return r.Room
}

func (w *HongzhongWorld) NewPlayer() *service.Player {
	p := NewMahjongPlayer()
	p.Player = service.NewPlayer(p)
	p.localObj = &HongzhongObj{MahjongPlayer: p}
	return p.Player
}

// HongzhongObj 广东麻将玩家逻辑
type HongzhongObj struct {
	*MahjongPlayer
}

func (obj *HongzhongObj) winHorse() []int {
	directions := [][]int{
		{1, 5, 9, 21, 25, 29, 41, 45, 49, 60, 100},
		{2, 6, 22, 26, 42, 46, 70, 110},
		{3, 7, 23, 27, 43, 47, 80, 120},
		{4, 8, 24, 28, 44, 48, 90},
	}

	room := obj.Room()
	hz := room.localMahjong.(*HongzhongMahjong)

	// seatId := obj.GetSeatIndex()
	// d := (seatId - room.dealer.GetSeatIndex() + room.NumSeat()) % room.NumSeat()
	d := 0 // TODO 按照赢的人抓马

	var cards []int
	for i := 0; i < hz.countHorse() && i < len(hz.horses); i++ {
		for _, c := range directions[d] {
			if hz.horses[i] == c || hz.horses[i] == 0 {
				cards = append(cards, hz.horses[i])
				break
			}
		}
	}

	return cards
}

func (ply *HongzhongObj) IsAbleWin() bool {
	// 没出牌之前，手牌中有4张癞子牌
	if ply.discardNum == 0 && ply.drawCard != -1 && CountSomeCards(ply.handCards, nil, 100) == 4 {
		return true
	}
	return ply.MahjongPlayer.IsAbleWin()
}
