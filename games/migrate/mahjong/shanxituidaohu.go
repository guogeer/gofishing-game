package mahjong

// 2018-04-09 Guogeer
// 山西推倒胡
import (
	"gofishing-game/games/migrate/internal/cardrule"
	"gofishing-game/internal/cardutils"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
)

func init() {
	w := NewShanxituidaohuWorld()
	service.AddWorld(w)
	AddHandlers(w.GetName())

	var cards []int
	for _, c := range []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 21, 22, 23, 24, 25, 26, 27, 28, 29, 41, 42, 43, 44, 45, 46, 47, 48, 49, 60, 70, 80, 90} {
		cards = append(cards, c, c, c, c)
	}
	cardutils.GetCardSystem().Init(cards)
}

type ShanxituidaohuMahjong struct {
	room *MahjongRoom
}

func (mj *ShanxituidaohuMahjong) OnCreateRoom() {
	room := mj.room
	// 不带风
	if room.CanPlay(OptBuDaiFeng) {
		room.CardSet().Remove(60, 70, 80, 90)
	}
}

func (mj *ShanxituidaohuMahjong) OnEnter(comer *MahjongPlayer) {
}

func (mj *ShanxituidaohuMahjong) OnReady() {
	room := mj.room

	room.Status = roomutils.RoomStatusPlaying
	room.StartDealCard()
	room.dealer.OnDraw()
}

func (mj *ShanxituidaohuMahjong) OnWin() {
	room := mj.room
	room.Award()
}

func (mj *ShanxituidaohuMahjong) Score(cards []int, melds []cardrule.Meld) (int, int) {
	room := mj.room
	winOpt := room.helper.Win(cards, melds)
	// 龙七对
	if winOpt != nil && winOpt.Qidui && winOpt.Pair2 > 0 {
		return LongQiDui, 18
	}
	// 十三幺
	if winOpt != nil && winOpt.Shisanyao {
		return ShiSanYao, 9
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

func (mj *ShanxituidaohuMahjong) Award() {
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
					detail.Times = 1
					if meld.Type == cardrule.MeldInvisibleKong {
						detail.Times = 2
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
					detail.Times = 3
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
		if p.drawCard != -1 && score == PingHu {
			points = 2
		}

		tempScore, tempPoints := PingHu, 2
		if p.IsRobKong() {
			tempScore, tempPoints = PaiXingQiangGangHu, 6
		}
		if p.IsDrawAfterKong() {
			tempScore, tempPoints = PaiXingGangShangHua, 4
		}
		if p.IsWinAfterOtherKong() {
			tempScore, tempPoints = PaiXingGangShangPao, 6
		}
		if points < tempPoints {
			score, points = tempScore, tempPoints
		}

		addition2 := map[string]int{}
		detail := ChipDetail{Seats: 1 << uint(p.GetSeatIndex()), Chip: int64(score), Points: points, Operate: cardrule.OperateWin}

		// 自摸
		if p.drawCard != -1 {
			addition2["自摸"] = 0
		} else {
			addition2["接炮"] = 0
		}
		if p.IsRobKong() {
			addition2["抢杠胡"] = 0
		}
		if p.IsDrawAfterKong() {
			addition2["杠上花"] = 0
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
			if room.CanPlay(OptZiMoJiaFan) {
				points *= 2
			}
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

func (mj *ShanxituidaohuMahjong) GameOver() {
}

type ShanxituidaohuWorld struct{}

func NewShanxituidaohuWorld() *ShanxituidaohuWorld {
	return &ShanxituidaohuWorld{}
}

func (w *ShanxituidaohuWorld) GetName() string {
	return "shanxituidaohumj"
}

func (w *ShanxituidaohuWorld) NewRoom(subId int) *roomutils.Room {
	r := NewMahjongRoom(subId)
	r.SetPlay(OptBuDaiFeng) // 不带风
	r.SetPlay(OptBaoTing)   // 报听

	r.SetPlay(OptBoom)
	r.SetPlay(OptAbleRobKong)
	r.SetPlay(OptSevenPairs)
	r.localMahjong = &ShanxituidaohuMahjong{room: r}
	return r.Room
}

func (w *ShanxituidaohuWorld) NewPlayer() *service.Player {
	p := NewMahjongPlayer()
	p.Player = service.NewPlayer(p)
	p.localObj = &ShanxituidaohuObj{MahjongPlayer: p}
	return p.Player
}

type ShanxituidaohuObj struct {
	*MahjongPlayer
}

func (obj *ShanxituidaohuObj) IsAbleWin() bool {
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
