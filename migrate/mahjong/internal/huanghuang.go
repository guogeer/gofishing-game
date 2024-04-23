package internal

// 2017-11-03 Guogeer
// 湖北晃晃麻将
import (
	"gofishing-game/internal/cardutils"
	mjutils "gofishing-game/migrate/mahjong/utils"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
	"quasar/utils"
	"strconv"
	"time"
)

var (
	huanghuangPiaoOptions = []int{0, 1, 3, 5}
)

// 晃晃麻将
type HuanghuangMahjong struct {
	room *MahjongRoom

	ghostCard int
}

func (mj *HuanghuangMahjong) OnCreateRoom() {
}

func (mj *HuanghuangMahjong) OnEnter(comer *MahjongPlayer) {
	room := mj.room

	data := map[string]any{
		"card":  mj.ghostCard,
		"Ghost": mj.getAnyCards(),
	}
	if room.Status == roomStatusChoosePiao {
		data["Piao"] = huanghuangPiaoOptions
	}
	if room.CanPlay(OptZiYouXuanPiao) && room.Status != 0 {
		all := make([]int, room.NumSeat())
		for i := 0; i < room.NumSeat(); i++ {
			if p := room.GetPlayer(i); p != nil {
				obj := p.localObj.(*HuanghuangObj)
				all[i] = obj.piao
				if room.Status == roomStatusChoosePiao && all[i] != -1 {
					all[i] = 0
				}
			}
		}
		data["PiaoList"] = all
	}

	comer.SetClientValue("localMahjong", data)
}

func (mj *HuanghuangMahjong) OnReady() {
	room := mj.room

	// 选漂
	if room.CanPlay(OptZiYouXuanPiao) {
		mj.startChoosePiao()
	} else {
		mj.startPlaying()
	}
}

func (mj *HuanghuangMahjong) startChoosePiao() {
	room := mj.room
	room.Status = roomStatusChoosePiao
	room.deadline = time.Now().Add(MaxOperateTime)
	room.Broadcast("StartChoosePiao", map[string]any{
		"Index": huanghuangPiaoOptions,
		"ts":    room.deadline.Unix(),
	})
	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		obj := p.localObj.(*HuanghuangObj)
		obj.piao = -1
		if !room.IsTypeScore() {
			p.operateTimer = p.TimerGroup.NewTimer(func() { obj.ChoosePiao(0) }, MaxOperateTime)
		}
	}
}

func (mj *HuanghuangMahjong) OnChoosePiao() {
	room := mj.room

	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		obj := p.localObj.(*HuanghuangObj)
		if obj.piao == -1 {
			return
		}
	}
	all := make([]int, room.NumSeat())
	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		obj := p.localObj.(*HuanghuangObj)
		all[i] = obj.piao
	}

	room.Broadcast("FinishChoosePiao", map[string]any{
		"PiaoList": all,
	})
	mj.startPlaying()
}

func (mj *HuanghuangMahjong) startPlaying() {
	room := mj.room

	room.Status = roomutils.RoomStatusPlaying
	room.StartDealCard()
	mj.ghostCard = room.CardSet().Deal()
	room.Broadcast("ChooseGhostCard", map[string]any{
		"card":  mj.ghostCard,
		"Ghost": mj.getAnyCards(),
	})

	room.dealer.OnDraw()
}

func (mj *HuanghuangMahjong) OnWin() {
	room := mj.room
	room.Award()
}

func (mj *HuanghuangMahjong) Score(cards []int, melds []mjutils.Meld) (int, int) {
	room := mj.room

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

	isSameColor := color&(color-1) == 0
	if pairNum+cards[NoneCard] > 6 {
		if pair2Num > 0 || pairNum+cards[NoneCard] > 7 {
			if isSameColor {
				return QingLongQiDui, 6
			}
			return LongQiDui, 3
		}
		if isSameColor {
			return QingQiDui, 3
		}
		return QiDui, 3
	}
	// 清一色
	if isSameColor {
		return QingYiSe, 3
	}
	// 碰碰胡
	opt := room.helper.Win(cards, melds)
	if opt != nil && opt.Duiduihu {
		return DuiDuiHu, 3
	}
	if cards[NoneCard] == 0 {
		return PingHu, 2
	}
	return PingHu, 1
}

func (mj *HuanghuangMahjong) Award() {
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
				case mjutils.MeldInvisibleKong, mjutils.MeldBentKong:
					detail.Times = 1
					if meld.Type == mjutils.MeldInvisibleKong {
						detail.Times = 2
					}
					for k := 0; k < room.NumSeat(); k++ {
						bill := &bills[k]
						if other := room.GetPlayer(k); other != nil && p != other {
							detail.Chip = -int64(detail.Times) * unit
							bill.Details = append(bill.Details, detail)
						}
					}
				case mjutils.MeldStraightKong:
					bill := &bills[meld.SeatId]
					detail.Times = 2
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
		obj := p.localObj.(*HuanghuangObj)

		copyCards := p.copyCards()
		score, points := mj.Score(copyCards, p.melds)
		c := mj.ghostCard
		mj.ghostCard = -1
		if p.IsAbleWin() {
			points *= 2
		}
		mj.ghostCard = c
		detail := ChipDetail{Seats: 1 << uint(p.GetSeatIndex()), Chip: int64(score), Operate: mjutils.OperateWin}

		addition2 := map[string]int{}
		// 自摸
		if p.drawCard != -1 {
			addition2["ZM"] = 0
		} else {
			addition2["JP"] = 0
		}
		detail.Addition2 = addition2
		if p.drawCard == -1 {
			// 放炮
			boom := room.boomPlayer()
			bill := &bills[boom.GetSeatIndex()]
			boomObj := boom.localObj.(*HuanghuangObj)
			if p.IsRobKong() {
				points *= room.NumSeat() - 1
			}

			// 漂分
			detail.Times = 2 * room.piao()
			if room.CanPlay(OptZiYouXuanPiao) {
				detail.Times = obj.piao + boomObj.piao
			}

			detail.Times += points
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
					otherObj := other.localObj.(*HuanghuangObj)

					// 漂分
					detail.Times = 2 * room.piao()
					if room.CanPlay(OptZiYouXuanPiao) {
						detail.Times = obj.piao + otherObj.piao
					}
					detail.Times += points
					detail.Chip = -int64(detail.Times) * unit
					bill.Details = append(bill.Details, detail)
				}
			}
		}
		room.Billing(bills[:])
	}
}

func (mj *HuanghuangMahjong) GameOver() {
	room := mj.room
	mj.ghostCard = -1

	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)

		obj := p.localObj.(*HuanghuangObj)
		obj.piao = 0
	}
}

// 癞子牌
func (mj *HuanghuangMahjong) getAnyCards() []int {
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

type HuanghuangWorld struct{}

func NewHuanghuangWorld() *HuanghuangWorld {
	return &HuanghuangWorld{}
}

func (w *HuanghuangWorld) GetName() string {
	return "huang2mj"
}

func (w *HuanghuangWorld) NewRoom(id, subId int) *roomutils.Room {
	r := NewMahjongRoom(id, subId)
	r.SetPlay(OptZiYouXuanPiao)
	r.SetPlay(OptZiMoJiaFan)
	r.SetPlay(OptPingHuLaiZi2)
	for _, v := range []int{1, 2, 5} {
		r.SetNoPlay("difen_" + strconv.Itoa(v))
	}
	for i := 0; i < 100; i++ {
		r.SetNoPlay("piao_" + strconv.Itoa(i))
	}

	r.SetPlay(OptBoom)
	r.SetPlay(OptAbleRobKong)
	r.SetPlay(OptSevenPairs)
	r.SetPlay(OptAbleKongAfterChowOrPong)

	r.localMahjong = &HuanghuangMahjong{room: r, ghostCard: -1}
	return r.Room
}

func (w *HuanghuangWorld) NewPlayer() *service.Player {
	p := NewMahjongPlayer()
	p.Player = service.NewPlayer(p)
	p.localObj = &HuanghuangObj{MahjongPlayer: p, piao: 0}
	return p.Player
}

type HuanghuangObj struct {
	*MahjongPlayer

	piao int
}

func (obj *HuanghuangObj) ChoosePiao(piao int) {
	p := obj.MahjongPlayer
	room := p.Room()
	if obj.piao != -1 {
		return
	}
	if room.Status != roomStatusChoosePiao {
		return
	}
	if utils.InArray(huanghuangPiaoOptions, piao) == 0 {
		return
	}
	obj.piao = piao
	room.Broadcast("ChoosePiao", map[string]any{"uid": p.Id, "Index": piao})

	mj := room.localMahjong.(*HuanghuangMahjong)
	mj.OnChoosePiao()
}

func (obj *HuanghuangObj) IsAbleWin() bool {
	room := obj.Room()
	p := obj.MahjongPlayer
	mj := room.localMahjong.(*HuanghuangMahjong)
	if obj.drawCard == -1 && room.IsAnyCard(room.lastCard) {
		return false
	}
	if !p.IsAbleWin() {
		return false
	}

	copyCards := obj.copyCards()
	score, _ := mj.Score(copyCards, obj.melds)
	// 平胡2个癞子仅可自摸
	if score == PingHu && copyCards[NoneCard] > 1 && p.drawCard == -1 && room.CanPlay(OptPingHuLaiZi2) {
		return false
	}
	// 碰碰胡、平胡等牌型胡任意牌时，不可以接炮
	if p.drawCard == -1 && (score == PingHu || score == DuiDuiHu) {
		c := mj.ghostCard
		mj.ghostCard = -1
		if !p.IsAbleWin() {
			for _, opt := range p.CheckWin() {
				if opt.WinCard == NoneCard {
					return false
				}
			}
		}
		mj.ghostCard = c
	}
	return true
}
