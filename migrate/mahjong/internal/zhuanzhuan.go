package internal

// 2017-6-6 Guogeer
// 转转麻将
import (
	mjutils "gofishing-game/migrate/mahjong/utils"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"

	"github.com/guogeer/quasar/config"
)

// 转转麻将
type ZhuanzhuanMahjong struct {
	room *MahjongRoom

	ghostCard int
	horses    []int // 马牌
}

func (zz *ZhuanzhuanMahjong) OnCreateRoom() {
	room := zz.room
	// 默认没有字牌
	zipai := []int{60, 70, 80, 90, 100, 110, 120}
	room.CardSet().Remove(zipai...)
	if room.CanPlay(OptHongZhongZuoGui) {
		room.CardSet().Recover(100)
	}
	// 两个人玩的时候只有万和字
	if room.CanPlay("seat_2") {
		// 没有筒、条
		for i := 1; i < 10; i++ {
			room.CardSet().Remove(20+i, 40+i)
		}
		room.CardSet().Recover(zipai...)
	}
}

func (zz *ZhuanzhuanMahjong) OnEnter(comer *MahjongPlayer) {
	data := map[string]any{
		"card":  zz.ghostCard,
		"ghost": zz.getAnyCards(),
	}
	comer.WriteJSON("getLocalMahjong", data)
}

func (zz *ZhuanzhuanMahjong) OnReady() {
	room := zz.room

	room.StartDealCard()
	room.Broadcast("chooseGhostCard", map[string]any{
		"card":  zz.ghostCard,
		"ghost": zz.getAnyCards(),
	})
	room.dealer.OnDraw()
}

func (zz *ZhuanzhuanMahjong) drawHorse(horseNum int) []int {
	room := zz.room

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

func (zz *ZhuanzhuanMahjong) countHorse() int {
	room := zz.room

	var horseNum int
	if room.CanPlay(OptBuyHorse2) {
		horseNum = 2
	} else if room.CanPlay(OptBuyHorse4) {
		horseNum = 4
	} else if room.CanPlay(OptBuyHorse6) {
		horseNum = 6
	} else if room.CanPlay(OptBuyHorse8) {
		horseNum = 8
	}
	return horseNum
}

func (zz *ZhuanzhuanMahjong) OnWin() {
	room := zz.room
	horseNum := zz.countHorse()

	zz.horses = zz.drawHorse(horseNum)
	if horseNum > 0 && zz.horses[0] > 0 {
		type Winner struct {
			SeatIndex int
			Horses    []int
		}
		var winners []Winner
		for _, p := range room.winPlayers {
			obj := p.localObj.(*ZhuanzhuanObj)
			winners = append(winners, Winner{
				SeatIndex: p.GetSeatIndex(),
				Horses:    obj.winHorse(),
			})
		}

		room.Broadcast("buyHorse", map[string]any{
			"horses":  zz.horses,
			"winners": winners,
		})
	}
	room.Award()
}

func (zz *ZhuanzhuanMahjong) Award() {
	room := zz.room
	unit, _ := config.Int("room", room.SubId, "unit")

	// 庄闲
	if room.CanPlay(OptZhuangXian) {
		bills := make([]Bill, room.NumSeat())
		for _, p := range room.winPlayers {
			detail := ChipDetail{
				Seats:   1 << uint(p.GetSeatIndex()),
				Operate: OpZhuangXian,
				Chip:    -unit,
			}
			// 放炮
			if p.drawCard == -1 && room.boomPlayer() == room.dealer {
				bill := &bills[room.dealer.GetSeatIndex()]
				bill.Details = append(bill.Details, detail)
			}
			// 自摸
			if p.drawCard != -1 && p == room.dealer {
				for i := 0; i < room.NumSeat(); i++ {
					if i != p.GetSeatIndex() {
						bill := &bills[i]
						bill.Details = append(bill.Details, detail)
					}
				}
			}
		}
		room.Billing(bills)
	}

	// 有人胡牌
	for i := 0; len(room.winPlayers) > 0 && i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		detail := ChipDetail{Seats: 1 << uint(p.GetSeatIndex())}
		for _, kong := range p.kongHistory {
			detail.Operate = kong.Type

			bills := make([]Bill, room.NumSeat())
			switch kong.Type {
			case mjutils.MeldInvisibleKong, mjutils.MeldBentKong:
				times := 1
				// 暗杠
				if kong.Type == mjutils.MeldInvisibleKong {
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
			case mjutils.MeldStraightKong:
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
		obj := p.localObj.(*ZhuanzhuanObj)
		detail := ChipDetail{Seats: 1 << uint(p.GetSeatIndex()), Operate: mjutils.OperateWin}
		// 玩家中马
		winHorses := obj.winHorse()
		addition2 := map[string]int{}

		if n := len(winHorses); n > 0 {
			addition2["ma"] = n
		}

		// 自摸
		if p.drawCard != -1 {
			addition2["zm"] = 2
		} else {
			addition2["jp"] = 1
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

func (zz *ZhuanzhuanMahjong) GameOver() {
	zz.ghostCard = -1
}

// 癞子牌
func (zz *ZhuanzhuanMahjong) getAnyCards() []int {
	room := zz.room

	m := make(map[int]bool)
	if room.CanPlay(OptHongZhongZuoGui) {
		// 白板做鬼
		m[100] = true
	}

	var a []int
	for c := range m {
		a = append(a, c)
	}
	return a
}

type ZhuanzhuanWorld struct{}

func NewZhuanzhuanWorld() *ZhuanzhuanWorld {
	return &ZhuanzhuanWorld{}
}

func (w *ZhuanzhuanWorld) GetName() string {
	return "zzmj"
}

func (w *ZhuanzhuanWorld) NewRoom(subId int) *roomutils.Room {
	r := NewMahjongRoom(subId)
	r.SetPlay(OptAbleRobKong)
	r.SetNoPlay(OptBoom)
	r.SetNoPlay(OptSevenPairs)
	r.SetNoPlay(OptHongZhongZuoGui)
	r.SetNoPlay(OptBuyHorse2)
	r.SetNoPlay(OptBuyHorse4)
	r.SetNoPlay(OptBuyHorse6)
	r.SetNoPlay(OptBuyHorse8)
	r.SetNoPlay(OptZhuangXian)

	r.SetNoPlay("seat_2")
	r.SetNoPlay("seat_3")
	r.SetPlay("seat_4")

	r.localMahjong = &ZhuanzhuanMahjong{room: r, ghostCard: -1}
	return r.Room
}

func (w *ZhuanzhuanWorld) NewPlayer() *service.Player {
	p := NewMahjongPlayer()
	p.Player = service.NewPlayer(p)
	p.localObj = &ZhuanzhuanObj{MahjongPlayer: p}
	return p.Player
}

// ZhuanzhuanObj 广东麻将玩家逻辑
type ZhuanzhuanObj struct {
	*MahjongPlayer
}

func (obj *ZhuanzhuanObj) winHorse() []int {
	directions := [][]int{
		{1, 5, 9, 21, 25, 29, 41, 45, 49, 60, 100},
		{2, 6, 22, 26, 42, 46, 70, 110},
		{3, 7, 23, 27, 43, 47, 80, 120},
		{4, 8, 24, 28, 44, 48, 90},
	}

	room := obj.Room()
	zz := room.localMahjong.(*ZhuanzhuanMahjong)

	seatId := obj.GetSeatIndex()
	d := (seatId - room.dealer.GetSeatIndex() + room.NumSeat()) % room.NumSeat()

	var cards []int
	for i := 0; i < zz.countHorse() && i < len(zz.horses); i++ {
		for _, c := range directions[d] {
			if zz.horses[i] == c || zz.horses[i] == 0 {
				cards = append(cards, zz.horses[i])
				break
			}
		}
	}

	return cards
}

func (ply *ZhuanzhuanObj) IsAbleWin() bool {
	// 没出牌之前，手牌中有4张癞子牌
	room := ply.Room()
	if room.CanPlay(OptHongZhongZuoGui) && ply.discardNum == 0 && ply.drawCard != -1 && CountSomeCards(ply.handCards, nil, 100) == 4 {
		return true
	}
	return ply.MahjongPlayer.IsAbleWin()
}
