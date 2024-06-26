package mahjong

// 2018-04-08 Guogeer
// 拐三角
import (
	"gofishing-game/games/migrate/internal/cardrule"
	"gofishing-game/internal/cardutils"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
)

func init() {
	w := NewGuaisanjiaoWorld()
	service.AddWorld(w)
	AddHandlers(w.GetName())

	var cards []int
	for _, c := range []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 21, 22, 23, 24, 25, 26, 27, 28, 29, 41, 42, 43, 44, 45, 46, 47, 48, 49, 60, 70, 80, 90} {
		cards = append(cards, c, c, c, c)
	}
	cardutils.AddCardSystem(w.GetName(), cards)
	cardutils.GetCardSystem(w.GetName()).ReserveAndHide(16)
}

// 拐三角麻将
type GuaisanjiaoMahjong struct {
	room *MahjongRoom
}

func (mj *GuaisanjiaoMahjong) OnCreateRoom() {
	room := mj.room
	// 不带风
	if room.CanPlay(OptBuDaiFeng) {
		room.CardSet().Remove(60, 70, 80, 90)
	}
}

func (mj *GuaisanjiaoMahjong) OnEnter(comer *MahjongPlayer) {
}

func (mj *GuaisanjiaoMahjong) OnReady() {
	room := mj.room

	room.Status = roomutils.RoomStatusPlaying
	room.StartDealCard()
	room.dealer.OnDraw()
}

func (mj *GuaisanjiaoMahjong) OnWin() {
	room := mj.room
	room.Award()
}

func (mj *GuaisanjiaoMahjong) Score(cards []int, melds []cardrule.Meld) (int, int) {
	// 清一色一条龙
	room := mj.room
	winOpt := room.helper.Win(cards, melds)
	if winOpt != nil && winOpt.Qingyise && winOpt.Yitiaolong {
		return QingYiSeYiTiaoLong, 18
	}
	// 龙七对
	if winOpt != nil && winOpt.Qidui && winOpt.Pair2 > 0 {
		return LongQiDui, 18
	}
	// 十三幺
	if winOpt != nil && winOpt.Shisanyao {
		return ShiSanYao, 18
	}
	if winOpt != nil && winOpt.Yitiaolong {
		return YiTiaoLong, 9
	}
	if winOpt != nil && winOpt.Qidui {
		return QiDui, 9
	}
	if winOpt != nil && winOpt.Qingyise {
		return QingYiSe, 9
	}
	return PingHu, 3
}

func (mj *GuaisanjiaoMahjong) Award() {
	room := mj.room
	unit := room.Unit()

	// 有人胡牌
	if len(room.winPlayers) > 0 {
		for i := 0; i < room.NumSeat(); i++ {
			p := room.GetPlayer(i)
			detail := ChipDetail{Seats: 1 << uint(p.GetSeatIndex())}
			for _, meld := range p.melds {
				detail.Operate = meld.Type
				bills := make([]Bill, room.NumSeat())
				switch meld.Type {
				case cardrule.MeldInvisibleKong, cardrule.MeldBentKong:
					detail.Times = 2
					if meld.Type == cardrule.MeldInvisibleKong {
						detail.Times = 3
					}
					for k := 0; k < room.NumSeat(); k++ {
						bill := &bills[k]
						if other := room.GetPlayer(k); other != nil && p != other {
							detail.Chip = -int64(detail.Times) * unit
							bill.Details = append(bill.Details, detail)
						}
					}
				case cardrule.MeldStraightKong:
					bill := &bills[meld.SeatIndex]
					detail.Times = 3 // 直杠3分
					detail.Chip = -int64(detail.Times) * unit
					bill.Details = append(bill.Details, detail)
				}
				room.Billing(bills)
			}
		}
	}

	// 胡牌
	for _, p := range room.winPlayers {
		bills := make([]Bill, room.NumSeat())

		copyCards := p.copyCards()
		score, points := mj.Score(copyCards, p.melds)

		addition2 := map[string]int{}
		detail := ChipDetail{Seats: 1 << uint(p.GetSeatIndex()), Chip: int64(score), Points: points, Operate: cardrule.OperateWin}
		if p.IsRobKong() {
			addition2["抢杠胡"] = points
			points += points
		}
		if p.IsDrawAfterKong() {
			addition2["杠上花"] = points
			points += points
		}
		if p.IsWinAfterOtherKong() {
			addition2["杠上炮"] = points
			points += points
		}

		if room.CanPlay(OptKanZhang) {
			// 坎张，只胡一张，凑成顺子
			if opts := p.CheckWin(); len(opts) == 1 {
				c := room.lastCard
				if copyCards[c-1] > 0 && copyCards[c] > 0 && copyCards[c+1] > 0 {
					copyCards[c-1]--
					copyCards[c]--
					copyCards[c+1]--

					if room.helper.Win(copyCards, p.melds) != nil {
						addition2["坎张"] = points
						points += points
					}

					copyCards[c-1]++
					copyCards[c]++
					copyCards[c+1]++
				}
			}
		}

		// 自摸
		if p.drawCard != -1 {
			addition2["自摸"] = 0
		} else {
			addition2["接炮"] = 0
		}
		// 连庄
		if t := p.continuousDealerTimes; t > 0 {
			addition2["连庄"] = t
			points += t
		}
		detail.Addition2 = addition2
		if p.drawCard == -1 {
			// 放炮
			boom := room.boomPlayer()
			bill := &bills[boom.GetSeatIndex()]

			detail.Times = points
			detail.Chip = -int64(detail.Times) * unit
			bill.Details = append(bill.Details, detail)
		} else {
			// 自摸
			for i := 0; i < room.NumSeat(); i++ {
				if other := room.GetPlayer(i); other != p {
					bill := &bills[other.GetSeatIndex()]
					detail.Times = points
					detail.Chip = -int64(detail.Times) * unit
					bill.Details = append(bill.Details, detail)
				}
			}
		}
		room.Billing(bills[:])
	}
}

func (mj *GuaisanjiaoMahjong) GameOver() {
}

type GuaisanjiaoWorld struct{}

func NewGuaisanjiaoWorld() *GuaisanjiaoWorld {
	return &GuaisanjiaoWorld{}
}

func (w *GuaisanjiaoWorld) GetName() string {
	return "guaisanjiaomj"
}

func (w *GuaisanjiaoWorld) NewRoom(subId int) *roomutils.Room {
	r := NewMahjongRoom(subId)
	r.SetPlay(OptBuDaiFeng) // 不带风
	r.SetPlay(OptBaoTing)   // 报听
	r.SetPlay(OptKanZhang)  // 坎张

	r.SetPlay(OptBoom)
	r.SetPlay(OptAbleRobKong)
	r.SetPlay(OptSevenPairs)
	r.localMahjong = &GuaisanjiaoMahjong{room: r}
	return r.Room
}

func (w *GuaisanjiaoWorld) NewPlayer() *service.Player {
	p := NewMahjongPlayer()
	p.Player = service.NewPlayer(p)
	p.localObj = &GuaisanjiaoObj{MahjongPlayer: p}
	return p.Player
}

type GuaisanjiaoObj struct {
	*MahjongPlayer
}

func (obj *GuaisanjiaoObj) IsAbleWin() bool {
	room := obj.Room()
	if room.CanPlay(OptBaoTing) && !obj.isReadyHand {
		return false
	}
	// 过胡不胡
	if obj.isPassWin {
		return false
	}
	return obj.MahjongPlayer.IsAbleWin()
}
