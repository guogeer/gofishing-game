package internal

// 2017-8-28 Guogeer
// 郑州麻将
import (
	mjutils "gofishing-game/migrate/mahjong/utils"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
	"time"

	"github.com/guogeer/quasar/utils"

	"github.com/guogeer/quasar/config"
)

// 转转麻将
type ZhengzhouMahjong struct {
	room *MahjongRoom

	ghostCard int
}

func (mj *ZhengzhouMahjong) OnCreateRoom() {
	room := mj.room
	// 默认没有字牌
	zipai := []int{60, 70, 80, 90, 100, 110, 120}
	// 不带字牌
	if room.CanPlay(OptBuDaiZiPai) {
		room.CardSet().Remove(zipai...)
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

func (mj *ZhengzhouMahjong) OnEnter(comer *MahjongPlayer) {
	room := mj.room

	data := map[string]any{
		"card":  mj.ghostCard,
		"ghost": mj.getAnyCards(),
	}
	if room.CanPlay(OptDaiPao) && room.Status != 0 {
		all := make([]int, room.NumSeat())
		for i := 0; i < room.NumSeat(); i++ {
			if p := room.GetPlayer(i); p != nil {
				obj := p.localObj.(*ZhengzhouObj)
				all[i] = obj.pao
				if room.Status == roomStatusChoosePao && all[i] != -1 {
					all[i] = 0
				}
			}
		}
		data["paoList"] = all
	}
	comer.SetClientValue("localMahjong", data)
}

func (mj *ZhengzhouMahjong) OnReady() {
	room := mj.room

	// 带跑
	if room.CanPlay(OptDaiPao) {
		mj.startChoosePao()
	} else {
		mj.startPlaying()
	}
}

func (mj *ZhengzhouMahjong) startChoosePao() {
	room := mj.room
	room.Status = roomStatusChoosePao
	room.deadline = time.Now().Add(MaxOperateTime)
	room.Broadcast("StartChoosePao", map[string]any{"ts": room.deadline.Unix()})
	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		obj := p.localObj.(*ZhengzhouObj)
		obj.pao = -1
		utils.StopTimer(p.operateTimer)
		p.operateTimer = utils.NewTimer(func() {
			if !room.IsTypeScore() {
				obj.ChoosePao(0)
			}
		}, MaxOperateTime)
	}
}

func (mj *ZhengzhouMahjong) OnChoosePao() {
	room := mj.room

	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		obj := p.localObj.(*ZhengzhouObj)
		if obj.pao == -1 {
			return
		}
	}
	all := make([]int, room.NumSeat())
	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		obj := p.localObj.(*ZhengzhouObj)
		all[i] = obj.pao
	}

	room.Broadcast("FinishChoosePao", map[string]any{
		"PaoList": all,
	})
	mj.startPlaying()
}

func (mj *ZhengzhouMahjong) startPlaying() {
	room := mj.room

	room.Status = roomutils.RoomStatusPlaying
	room.StartDealCard()
	if room.CanPlay(OptDaiHun) {
		mj.ghostCard = room.CardSet().Deal()
	}
	room.Broadcast("ChooseGhostCard", map[string]any{
		"card":  mj.ghostCard,
		"Ghost": mj.getAnyCards(),
	})

	room.dealer.OnDraw()
}

func (mj *ZhengzhouMahjong) OnWin() {
	room := mj.room
	room.Award()
}

func (mj *ZhengzhouMahjong) Award() {
	room := mj.room
	unit, _ := config.Int("Room", room.SubId, "Unit")

	// 有人胡牌或荒庄不荒杠
	if len(room.winPlayers) > 0 || room.CanPlay(OptHuangZhuangBuHuangGang) {
		for i := 0; i < room.NumSeat(); i++ {
			p := room.GetPlayer(i)
			obj := p.localObj.(*ZhengzhouObj)
			detail := ChipDetail{Seats: 1 << uint(p.GetSeatIndex())}
			for _, meld := range p.melds {
				detail.Operate = meld.Type
				bills := make([]Bill, room.NumSeat())
				switch meld.Type {
				case mjutils.MeldInvisibleKong:
					for k := 0; k < room.NumSeat(); k++ {
						bill := &bills[k]
						if other := room.GetPlayer(k); other != nil && p != other {
							detail.Times = 1
							if room.CanPlay(OptGangPao) {
								otherObj := other.localObj.(*ZhengzhouObj)
								detail.Times += obj.pao + otherObj.pao
							}
							if room.CanPlay(OptZhuangJiaJiaDi) {
								if p == room.dealer || other == room.dealer {
									detail.Times++
								}
							}
							detail.Chip = -int64(detail.Times) * unit
							bill.Details = append(bill.Details, detail)
						}
					}
				case mjutils.MeldBentKong, mjutils.MeldStraightKong:
					other := room.GetPlayer(meld.SeatIndex)
					bill := &bills[meld.SeatIndex]
					detail.Times = 1
					if room.CanPlay(OptGangPao) {
						otherObj := other.localObj.(*ZhengzhouObj)
						detail.Times += obj.pao + otherObj.pao
					}
					if room.CanPlay(OptZhuangJiaJiaDi) {
						if p == room.dealer || other == room.dealer {
							detail.Times++
						}
					}

					detail.Chip = -int64(detail.Times) * unit
					bill.Details = append(bill.Details, detail)
				default:
					continue
				}
				room.Billing(bills)
			}
		}
	}

	// 胡牌
	for _, p := range room.winPlayers {
		bills := make([]Bill, room.NumSeat())
		obj := p.localObj.(*ZhengzhouObj)
		detail := ChipDetail{Seats: 1 << uint(p.GetSeatIndex()), Operate: mjutils.OperateWin}

		addition2 := map[string]int{}
		// 自摸
		if p.drawCard != -1 {
			addition2["ZM"] = 0
		} else {
			addition2["JP"] = 0
		}
		points := 1
		// 七对加倍
		pairNum := 0
		copyCards := p.copyCardsWithoutNoneCard()
		for _, n := range copyCards {
			pairNum += n / 2
		}
		if room.CanPlay(OptQiDuiJiaBei) {
			if pairNum+copyCards[NoneCard] > 6 && len(p.melds) == 0 {
				addition2["QDJB"] = 0
				points *= 2
			}
		}
		// 4混加倍
		if room.CanPlay(OptSiHunJiaBei) {
			if copyCards[NoneCard] == 4 {
				addition2["SHJB"] = 0
				points *= 2
			}
		}
		// 杠上花加倍
		if room.CanPlay(OptGangShangHuaJiaBei) {
			if p.drawCard != -1 && room.kongPlayer == p {
				addition2["GSHJB"] = 0
				points *= 2
			}
		}

		detail.Addition2 = addition2
		if p.drawCard == -1 {
			// 放炮
			boom := room.boomPlayer()
			bill := &bills[boom.GetSeatIndex()]
			boomObj := boom.localObj.(*ZhengzhouObj)
			detail.Times = 1 + obj.pao + boomObj.pao
			if room.CanPlay(OptZhuangJiaJiaDi) {
				if p == room.dealer || boom == room.dealer {
					detail.Times++
				}
			}
			detail.Times *= points
			detail.Chip = -int64(detail.Times) * unit
			bill.Details = append(bill.Details, detail)
		} else {
			// 自摸
			for i := 0; i < room.NumSeat(); i++ {
				if other := room.GetPlayer(i); other != p {
					bill := &bills[other.GetSeatIndex()]
					otherObj := other.localObj.(*ZhengzhouObj)
					detail.Times = 1 + obj.pao + otherObj.pao
					if room.CanPlay(OptZhuangJiaJiaDi) {
						if p == room.dealer || other == room.dealer {
							detail.Times++
						}
					}
					detail.Times *= points
					detail.Chip = -int64(detail.Times) * unit
					bill.Details = append(bill.Details, detail)
				}
			}
		}
		room.Billing(bills[:])
	}
}

func (mj *ZhengzhouMahjong) GameOver() {
	room := mj.room
	mj.ghostCard = -1

	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)

		obj := p.localObj.(*ZhengzhouObj)
		obj.pao = 0
	}
}

// 癞子牌
func (mj *ZhengzhouMahjong) getAnyCards() []int {
	m := make(map[int]bool)
	for _, c := range GetNextCards(mj.ghostCard, 1) {
		m[c] = true
	}
	var a []int
	for c := range m {
		a = append(a, c)
	}
	return a
}

type ZhengzhouWorld struct{}

func NewZhengzhouWorld() *ZhengzhouWorld {
	return &ZhengzhouWorld{}
}

func (w *ZhengzhouWorld) GetName() string {
	return "zhengzhoumj"
}

func (w *ZhengzhouWorld) NewRoom(subId int) *roomutils.Room {
	r := NewMahjongRoom(id, subId)
	r.SetPlay(OptBoom)
	r.SetPlay(OptDaiPao)
	r.SetPlay(OptDaiHun)
	r.SetPlay(OptGangPao)
	r.SetPlay(OptZhuangJiaJiaDi)
	r.SetPlay(OptGangShangHuaJiaBei)
	r.SetPlay(OptQiDuiJiaBei)
	r.SetPlay(OptSiHunJiaBei)
	r.SetNoPlay(OptHuangZhuangBuHuangGang)
	r.SetNoPlay(OptBuDaiZiPai)

	r.SetNoPlay("seat_2")
	r.SetNoPlay("seat_3")
	r.SetPlay("seat_4")

	r.SetPlay(OptAbleRobKong)
	r.SetPlay(OptSevenPairs)
	r.SetPlay(OptQiDuiLaiZiZuoDui)

	r.localMahjong = &ZhengzhouMahjong{room: r, ghostCard: -1}
	return r.Room
}

func (w *ZhengzhouWorld) NewPlayer() *service.Player {
	p := NewMahjongPlayer()
	p.Player = service.NewPlayer(p)
	p.localObj = &ZhengzhouObj{MahjongPlayer: p, pao: 0}
	return p.Player
}

type ZhengzhouObj struct {
	*MahjongPlayer

	pao int
}

func (obj *ZhengzhouObj) IsAbleWin() bool {
	// 手牌中有4张癞子牌
	p := obj.MahjongPlayer
	room := p.Room()
	cards := room.GetAnyCards()
	if p.drawCard != -1 && CountSomeCards(p.handCards, nil, cards...) == 4 {
		return true
	}
	if p.drawCard == -1 && !room.CanPlay(OptBoom) {
		return false
	}
	return p.IsAbleWin()
}

func (obj *ZhengzhouObj) ChoosePao(pao int) {
	p := obj.MahjongPlayer
	room := p.Room()
	if obj.pao != -1 {
		return
	}
	if room.Status != roomStatusChoosePao {
		return
	}
	if utils.InArray([]int{0, 1, 2, 3}, pao) == 0 {
		return
	}
	obj.pao = pao
	room.Broadcast("ChoosePao", map[string]any{"uid": p.Id, "Index": pao})

	mj := room.localMahjong.(*ZhengzhouMahjong)
	mj.OnChoosePao()
}
