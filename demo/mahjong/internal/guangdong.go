package internal

// 2017-5-11 Guogeer
// 二人、三人、四人推倒胡，鸡胡
import (
	mjutils "gofishing-game/demo/mahjong/utils"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
	"slices"
	"strings"

	"github.com/guogeer/quasar/config"
)

// 广东麻将
type guangdongBranch interface {
	drawHorse(int) []int // 摸马
}

// 推倒胡
type tuidaohuMahjong struct {
	*GuangdongMahjong
}

func (tdh *tuidaohuMahjong) drawHorse(horseNum int) []int {
	room := tdh.room

	var horses []int
	// 推倒胡
	for i := 0; i < horseNum; i++ {
		c := room.CardSet().Deal()
		// 重新洗牌
		if c == -1 {
			room.CardSet().Shuffle()
			c = room.CardSet().Deal()
		}
		horses = append(horses, c)
	}
	return horses
}

// 广东鸡胡
type guangdongJihuMahjong struct {
	*GuangdongMahjong
}

func (jh *guangdongJihuMahjong) drawHorse(horseNum int) []int {
	room := jh.room

	var horses []int
	// 广东鸡胡
	if room.CardSet().Count() == 0 {
		horses = make([]int, horseNum)
	} else {
		var last int
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
	}
	return horses
}

type GuangdongMahjong struct {
	room *MahjongRoom

	// 分支，如推倒胡、鸡胡
	branch guangdongBranch

	ghostCard int   // 翻鬼的牌
	horses    []int // 马牌
	genZhuang bool  // 跟庄
}

func NewGuangdongMahjong() *GuangdongMahjong {
	return &GuangdongMahjong{
		ghostCard: -1,
	}
}

func (gd *GuangdongMahjong) OnCreateRoom() {
	room := gd.room
	zipai := []int{60, 70, 80, 90, 100, 110, 120}
	// 不带字牌
	if room.CanPlay(OptBuDaiZiPai) {
		room.CardSet().Remove(zipai...)
	}
	// 不带万
	if room.CanPlay(OptBuDaiWan) {
		room.CardSet().Remove(1, 2, 3, 4, 5, 6, 7, 8, 9)
	}

	if room.CanPlay(OptBaiBanZuoGui) {
		room.CardSet().Recover(120)
	}
	// 两个人玩的时候只有万和字
	if room.CanPlay("seat_2") {
		for i := 1; i < 10; i++ {
			room.CardSet().Remove(20+i, 40+i)
		}
		room.CardSet().Recover(zipai...)
	}
}

func (gd *GuangdongMahjong) OnEnter(comer *MahjongPlayer) {
	data := map[string]any{
		"card":  gd.ghostCard,
		"Ghost": gd.getAnyCards(),
	}
	comer.WriteJSON("GetLocalMahjong", data)
}

func (gd *GuangdongMahjong) OnReady() {
	room := gd.room

	gd.ghostCard = -1
	// 开始选鬼牌
	if room.CanPlay(OptFanGui1) || room.CanPlay(OptFanGui2) {
		gd.ghostCard = room.CardSet().Deal()
	}
	room.Broadcast("ChooseGhostCard", map[string]any{
		"card":  gd.ghostCard,
		"Ghost": gd.getAnyCards(),
	})
	//重新洗牌
	room.CardSet().Shuffle()
	room.StartDealCard()
	room.dealer.OnDraw()
}

func (gd *GuangdongMahjong) countHorse() int {
	room := gd.room

	var horseNum int
	if room.CanPlay(OptBuyHorse2) {
		horseNum = 2
	} else if room.CanPlay(OptBuyHorse4) {
		horseNum = 4
	} else if room.CanPlay(OptBuyHorse6) {
		horseNum = 6
	} else if room.CanPlay(OptBuyHorse8) {
		horseNum = 8
	} else if room.CanPlay(OptBuyHorse10) {
		horseNum = 10
	} else if room.CanPlay(OptBuyHorse20) {
		horseNum = 20
	}
	return horseNum
}

func (gd *GuangdongMahjong) OnWin() {
	room := gd.room

	var horseNum, extraHorse int
	if room.CanPlay(OptMaiMaJieJieGao) {
		for _, p := range room.winPlayers {
			obj := p.localObj.(*GuangdongObj)
			if n := obj.extraHorse(); extraHorse < n {
				extraHorse = n
			}
		}
	}
	horseNum = gd.countHorse() + extraHorse
	gd.horses = gd.branch.drawHorse(horseNum)
	if horseNum > 0 && gd.horses[0] > 0 {
		type Winner struct {
			SeatId int
			Horses []int
		}
		var winners []Winner
		for _, p := range room.winPlayers {
			obj := p.localObj.(*GuangdongObj)
			winners = append(winners, Winner{
				SeatId: p.GetSeatIndex(),
				Horses: obj.winHorse(),
			})
		}

		room.Broadcast("BuyHorse", map[string]any{
			"Horses":  gd.horses,
			"Extra":   extraHorse,
			"Winners": winners,
		})
	}
	room.Award()
}

func (gd *GuangdongMahjong) Billing(bills []Bill) {
	room := gd.room

	// 杠爆全包
	for _, p := range room.winPlayers {
		obj := p.localObj.(*GuangdongObj)
		bad := obj.checkAllInclude()
		for i := 0; bad != nil && i < len(bills); i++ {
			if i == bad.GetSeatIndex() {
				continue
			}

			bill := &bills[i]
			for j := 0; j < len(bill.Details); j++ {
				detail := &bill.Details[j]
				if detail.GetSeatIndex() != bad.GetSeatIndex() {
					bills[bad.GetSeatIndex()].Details = append(bills[bad.GetSeatIndex()].Details, *detail)
				}
				detail.Chip = 0
			}
		}
	}
	room.Billing(bills)
}

func (gd *GuangdongMahjong) Score(cards []int, melds []Meld) (int, int) {
	return PingHu, 2
}

func (gd *GuangdongMahjong) Award() {
	room := gd.room
	unit := room.Unit()
	dealer := room.dealer

	// 跟庄
	if gd.genZhuang && dealer != nil {
		bills := make([]Bill, room.NumSeat())
		bill := &bills[dealer.GetSeatIndex()]
		for i := 0; i < room.NumSeat(); i++ {
			if p := room.GetPlayer(i); p != nil && p != dealer {
				detail := ChipDetail{
					Seats:   1 << uint(p.GetSeatIndex()),
					Operate: OpGenZhuang,
					Times:   1,
				}
				detail.Chip = -int64(detail.Times) * unit
				bill.Details = append(bill.Details, detail)
			}
		}
		gd.Billing(bills)
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
				detail.Times = 1
				detail.Chip = -int64(detail.Times) * unit
				bill.Details = append(bill.Details, detail)
			}
			gd.Billing(bills)
		}
	}

	// 胡牌
	for _, p := range room.winPlayers {
		bills := make([]Bill, room.NumSeat())

		obj := p.localObj.(*GuangdongObj)
		detail := ChipDetail{Seats: 1 << uint(p.GetSeatIndex()), Operate: mjutils.OperateWin}
		// 玩家中马
		winHorses := obj.winHorse()
		addition2 := map[string]int{}

		points := 2
		// 无鬼加倍
		if cards := gd.getAnyCards(); len(cards) > 0 && room.CanPlay(OptWuGuiJiaBei) && CountSomeCards(p.handCards, nil, cards...) == 0 {
			addition2["WGJB"] = 0
			points *= 2
		}
		pairNum := 0
		copyCards := p.copyCards()
		for _, n := range copyCards {
			pairNum += n / 2
		}
		// 七对加番
		if room.CanPlay(OptQiDuiJiaFan) && pairNum+copyCards[NoneCard] > 6 && len(p.melds) == 0 {
			addition2["QDJF"] = 0
			points *= 2
		}
		// 节节高
		if n := obj.continuousDealerTimes; room.CanPlay(OptJieJieGao) && n > 0 {
			addition2["JJG"] = 2 * n
		}

		times := 2
		// 马跟底分
		if room.CanPlay(OptMaGenDiFen) {
			times = points
		}
		if n := len(winHorses); n > 0 {
			addition2["MA"] = times * n
		}

		times = 0
		for _, kong := range p.kongHistory {
			switch kong.Type {
			case mjutils.MeldInvisibleKong:
				times += 2
			case mjutils.MeldStraightKong:
				times += 1
			case mjutils.MeldBentKong:
				times += 1
			}
		}
		// 马跟杠
		if room.CanPlay(OptMaGenGang) {
			addition2["MGG"] = times * len(winHorses)
		}

		times = points * p.multiples
		for _, t := range addition2 {
			times += t
		}

		// 抢杠胡，包马分、胡牌分
		if room.kongPlayer != nil && room.kongPlayer != p && room.discardPlayer == nil &&
			room.CanPlay(OptQiangGangQuanBao) {
			if extra := room.NumSeat() - 1; extra > 0 {
				addition2["QGQB"] = 0

				times += points * extra
				if t, ok := addition2["MA"]; ok {
					times += t * extra
				}
				if t, ok := addition2["MGG"]; ok {
					times += t * extra
				}
			}
		}
		if p.drawCard != -1 {
			// 自摸
			addition2["ZM"] = 0
			// 杠爆全包
			if obj.checkAllInclude() != nil {
				addition2["GBQB"] = 0
			}
		}

		detail.Times = times
		detail.Multiples = p.multiples
		detail.Chip = -unit * int64(times)
		detail.Addition2 = addition2
		if p.drawCard == -1 {
			// 放炮
			boom := room.kongPlayer
			if other := room.discardPlayer; other != nil {
				boom = other
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
		gd.Billing(bills[:])
	}
}

func (gd *GuangdongMahjong) GameOver() {
	gd.ghostCard = -1
	gd.genZhuang = false
}

// 癞子牌
func (gd *GuangdongMahjong) getAnyCards() []int {
	room := gd.room

	m := make(map[int]bool)
	if room.CanPlay(OptBaiBanZuoGui) {
		// 白板做鬼
		m[120] = true
	}

	ghostNum := 0
	if room.CanPlay(OptFanGui1) {
		// 翻鬼
		ghostNum = 1
	} else if room.CanPlay(OptFanGui2) {
		// 翻双鬼
		ghostNum = 2
	}
	for _, c := range GetNextCards(gd.ghostCard, ghostNum) {
		m[c] = true
	}
	var a []int
	for c := range m {
		a = append(a, c)
	}
	return a
}

type GuangdongWorld struct {
	name string
}

func NewGuangdongWorld() *GuangdongWorld {
	return &GuangdongWorld{}
}

func (w *GuangdongWorld) NewRoom(id, subId int) *roomutils.Room {
	r := NewMahjongRoom(id, subId)
	r.SetNoPlay(OptAbleRobKong)
	r.SetNoPlay(OptSevenPairs)
	r.SetNoPlay(OptQiangGangQuanBao)
	r.SetNoPlay(OptGangBaoQuanBao)
	r.SetNoPlay(OptQiDuiJiaFan)
	r.SetNoPlay(OptWuGuiJiaBei)
	r.SetNoPlay(OptBuDaiZiPai)
	r.SetNoPlay(OptGenZhuang)
	r.SetNoPlay(OptBuDaiWan)
	r.SetNoPlay(OptJieJieGao)
	r.SetNoPlay(OptBaiBanZuoGui)
	r.SetNoPlay(OptFanGui1)
	r.SetNoPlay(OptFanGui2)
	r.SetNoPlay(OptBuyHorse2)
	r.SetNoPlay(OptBuyHorse4)
	r.SetNoPlay(OptBuyHorse6)
	r.SetNoPlay(OptBuyHorse8)
	r.SetNoPlay(OptBuyHorse20)
	r.SetNoPlay(OptMaGenDiFen)
	r.SetNoPlay(OptMaGenGang)
	r.SetNoPlay(OptMingGangKeQiang)
	r.SetNoPlay(OptMaiMaJieJieGao)

	local := NewGuangdongMahjong()
	local.room = r

	tags, _ := config.String("room", subId, "tags")
	local.branch = &tuidaohuMahjong{GuangdongMahjong: local}
	if slices.Index(strings.Split(tags, ","), "gdjh") >= 0 {
		local.branch = &guangdongJihuMahjong{GuangdongMahjong: local}
	}
	if slices.Index(strings.Split(tags, ","), "double") >= 0 {
		r.SetPlay(OptBaoTing)                      // 报听
		r.SetPlay(OptAbleDouble)                   // 加倍
		r.SetPlay(OptAbleLookOthersAfterReadyHand) // 听牌可看牌
	}
	if slices.Index(strings.Split(tags, ","), "boom") >= 0 {
		r.SetPlay(OptBoom)
	}

	r.localMahjong = local
	return r.Room
}

func (w *GuangdongWorld) GetName() string {
	return w.name
}

func (w *GuangdongWorld) SetName(name string) {
	w.name = name
}

func (w *GuangdongWorld) NewPlayer() *service.Player {
	p := NewMahjongPlayer()
	p.Player = service.NewPlayer(p)
	p.localObj = &GuangdongObj{MahjongPlayer: p}
	return p.Player
}

// GuangdongObj 广东麻将玩家逻辑
type GuangdongObj struct {
	*MahjongPlayer
}

func (obj *GuangdongObj) extraHorse() int {
	room := obj.Room()
	if room.CanPlay(OptMaiMaJieJieGao) {
		return obj.continuousDealerTimes * 2
	}
	return 0
}

func (obj *GuangdongObj) winHorse() []int {
	directions := [][]int{
		{1, 5, 9, 21, 25, 29, 41, 45, 49, 60, 100},
		{2, 6, 22, 26, 42, 46, 70, 110},
		{3, 7, 23, 27, 43, 47, 80, 120},
		{4, 8, 24, 28, 44, 48, 90},
	}

	room := obj.Room()
	gd := room.localMahjong.(*GuangdongMahjong)

	seatId := obj.GetSeatIndex()
	// 二人推倒胡按庄家中马
	if room.NumSeat() == 2 {
		seatId = room.dealer.GetSeatIndex()
	}
	d := (seatId - room.dealer.GetSeatIndex() + room.NumSeat()) % room.NumSeat()

	var cards []int
	for i := 0; i < gd.countHorse()+obj.extraHorse() && i < len(gd.horses); i++ {
		for _, c := range directions[d] {
			if gd.horses[i] == c || gd.horses[i] == 0 {
				cards = append(cards, gd.horses[i])
				break
			}
		}
	}

	return cards
}

func (obj *GuangdongObj) OnDiscard() {
	room := obj.Room()

	dealer := room.dealer
	genZhuang := room.CanPlay(OptGenZhuang)
	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		if len(p.melds) > 0 || len(dealer.discardHistory) != 1 || len(p.discardHistory) != 1 || dealer.discardHistory[0] != p.discardHistory[0] {
			genZhuang = false
		}
	}
	if gd := room.localMahjong.(*GuangdongMahjong); !gd.genZhuang {
		gd.genZhuang = genZhuang
	}
}

// 全包
func (obj *GuangdongObj) checkAllInclude() *MahjongPlayer {
	room := obj.Room()

	p := obj.MahjongPlayer
	if !room.CanPlay(OptGangBaoQuanBao) {
		return nil
	}
	if p.drawCard == -1 {
		return nil
	}
	if len(p.continuousKong) == 0 {
		return nil
	}
	if kong := p.continuousKong[0]; kong.Type == mjutils.MeldStraightKong {
		return kong.other
	}
	return nil
}

func (ply *GuangdongObj) IsAbleWin() bool {
	// 没出牌之前，手牌中有4张癞子牌
	room := ply.Room()
	if room.CanPlay(OptBaiBanZuoGui) && ply.discardNum == 0 && ply.drawCard != -1 && CountSomeCards(ply.handCards, nil, 120) == 4 {
		return true
	}
	return ply.MahjongPlayer.IsAbleWin()
}

func (obj *GuangdongObj) OnDouble() {
	obj.forceReadyHand = true
}
