package internal

// 2018-3-14 Guogeer
// 潮汕麻将
import (
	"gofishing-game/internal/cardutils"
	mjutils "gofishing-game/migrate/mahjong/utils"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
)

var chaoshanScoreList = map[int]int{
	PingHu:            2,
	DuiDuiHu:          4,
	HunYiSe:           4,
	QingYiSe:          6,
	QiDui:             6,
	LongQiDui:         10,
	ShuangHaoHuaQiDui: 20,
	SanHaoHuaQiDui:    30,
	ZiYiSe:            20,
	DaiYaoJiu:         10,
	QingYaoJiu:        20,
	ShiSanYao:         26,
	ShiBaLuoHan:       36,
}

type chaoshanMahjong struct {
	room *MahjongRoom

	horses []int
	maima  [][]int // 买马
	fama   []int   // 罚马

	isBigBoom bool // 大牌点炮胡
}

func NewchaoshanMahjong() *chaoshanMahjong {
	return &chaoshanMahjong{}
}

func (mj *chaoshanMahjong) OnCreateRoom() {
	room := mj.room
	room.helper.ReserveCardNum = room.GetPlayValue("jiangma")
	mj.maima = make([][]int, room.NumSeat())
	for i := 0; i < room.NumSeat(); i++ {
		mj.maima[i] = make([]int, room.GetPlayValue("maima"))
	}
	mj.horses = make([]int, room.GetPlayValue("jiangma"))
	mj.fama = make([]int, room.GetPlayValue("fama"))

	mj.isBigBoom = false
	if room.CanPlay(OptBoom) {
		mj.isBigBoom = true
	}
	room.isAbleBoom = true
}

func (mj *chaoshanMahjong) OnEnter(comer *MahjongPlayer) {
	comer.SetClientValue("localMahjong", map[string]any{"maima": len(mj.maima[0]), "fama": len(mj.fama)})
}

func (mj *chaoshanMahjong) OnReady() {
	room := mj.room

	for i := range mj.maima {
		for j := range mj.maima[i] {
			mj.maima[i][j] = room.CardSet().Deal()
		}
	}
	for i := range mj.fama {
		mj.fama[i] = room.CardSet().Deal()
	}
	room.Broadcast("StartBuyHorses", map[string]any{"Maima": len(mj.maima[0]), "Fama": len(mj.fama)})

	room.StartDealCard()
	room.dealer.OnDraw()
}

func (mj *chaoshanMahjong) Score(cards []int, melds []mjutils.Meld) (int, int) {
	room := mj.room
	scores := make(map[int]bool)
	winOpt := room.helper.Win(cards, melds)

	var colors, total, zipai, yaojiu int
	var all = CountAllCards(cards, melds)
	for _, c := range cardutils.GetAllCards() {
		total += all[c]
		if c%10 == 0 {
			zipai += all[c]
		}
		if all[c] > 0 && c%10 != 0 {
			colors |= 1 << uint(c/10)
		}
		if IsSameValue(c, 1, 9) {
			yaojiu += all[c]
		}
	}

	scores[PingHu] = true
	if winOpt != nil && winOpt.Duiduihu {
		scores[DuiDuiHu] = true
	}
	if colors&(colors-1) == 0 && zipai > 0 {
		scores[HunYiSe] = true
	}
	if winOpt != nil && winOpt.Qingyise {
		scores[QingYiSe] = true
	}
	// 七对
	if winOpt != nil && winOpt.Qidui {
		scores[QiDui] = true
		switch winOpt.Pair2 {
		case 1:
			scores[LongQiDui] = true
		case 2:
			scores[ShuangHaoHuaQiDui] = true
		case 3:
			scores[SanHaoHuaQiDui] = true
		}
	}

	// 十八罗汉
	if winOpt != nil && winOpt.KongNum == 4 {
		scores[ShiBaLuoHan] = true
	}
	// 十三幺
	if winOpt != nil && winOpt.Shisanyao {
		scores[ShiSanYao] = true
	}

	if total == zipai {
		scores[ZiYiSe] = true
	}
	/*
		// 大四喜
		if all[60] > 2 && all[70] > 2 && all[80] > 2 && all[90] > 2 {
			scores[DaSiXi] = true
		}
		// 大三元
		if all[100] > 2 && all[110] > 2 && all[120] > 2 {
			scores[DaSanYuan] = true
		}
		// 小四喜
		if all[60]/3+all[70]/3+all[80]/3+all[90]/3 == 4 &&
			(all[60] == 2 || all[70] == 2 || all[80] == 2 || all[90] == 2) {
			scores[DaSiXi] = true
		}
		// 小三元
		if all[100]/3+all[110]/3+all[120]/3 > 0 &&
			(all[100] == 2 || all[110] == 2 || all[120] == 2) {
			scores[DaSanYuan] = true
		}
	*/
	// 带幺九、清幺九
	for _, pair := range cardutils.GetAllCards() {
		tempZipai, tempYaojiu := zipai, yaojiu
		if cards[pair] > 1 {
			cards[pair] -= 2
			if pair%10 == 0 {
				tempZipai -= 2
			}
			if IsSameValue(pair, 1, 9) {
				tempYaojiu -= 2
			}
			if tempZipai+tempYaojiu+2 == total {
				if opts := room.helper.Split(cards); len(opts) > 0 {
					scores[DaiYaoJiu] = true
					if yaojiu+2 == total {
						scores[QingYaoJiu] = true
					}
				}
			}
			cards[pair] += 2
		}
	}

	var bestScoreId, bestPoints int
	for k := range scores {
		if points := chaoshanScoreList[k]; bestPoints < points {
			bestScoreId = k
			bestPoints = points
		}
	}
	if room.CanPlay(OptXiaoHu) {
		bestPoints = 2
		if bestScoreId == PingHu {
			bestPoints = 1
		}
	}
	if max := room.GetPlayValue("fengding"); max > 0 && bestPoints > max {
		bestPoints = max
	}
	return bestScoreId, bestPoints
}

// 中马的座位
func (mj *chaoshanMahjong) winHorse(horse int) int {
	room := mj.room
	allDirections := [][][]int{
		{},
		{},
		{
			{1, 4, 7, 21, 24, 27, 41, 44, 47, 60, 90, 120},
			{2, 5, 8, 22, 25, 28, 42, 45, 48, 70, 100},
		},
		{
			{1, 4, 7, 21, 24, 27, 41, 44, 47, 60, 90, 120},
			{2, 5, 8, 22, 25, 28, 42, 45, 48, 70, 100},
			{3, 6, 9, 23, 26, 29, 43, 46, 49, 80, 110},
		},
		{
			{1, 5, 9, 21, 25, 29, 41, 45, 49, 60, 100},
			{2, 6, 22, 26, 42, 46, 70, 110},
			{3, 7, 23, 27, 43, 47, 80, 120},
			{4, 8, 24, 28, 44, 48, 90},
		},
	}

	startSeatId := room.dealer.GetSeatIndex()
	directions := allDirections[room.NumSeat()]
	for d := range directions {
		for _, c := range directions[d] {
			if horse == c {
				return (d + startSeatId) % room.NumSeat()
			}
		}
	}
	return -1
}

func (mj *chaoshanMahjong) jiangmaSeatId() int {
	room := mj.room

	jiangmaSeatId := -1
	for _, p := range room.winPlayers {
		jiangmaSeatId = p.GetSeatIndex()
	}
	if len(room.winPlayers) > 1 {
		boom := room.boomPlayer()
		if boom != nil {
			jiangmaSeatId = boom.GetSeatIndex()
		}
	}
	return jiangmaSeatId
}

func (mj *chaoshanMahjong) OnWin() {
	room := mj.room

	jiangmaSeatId := mj.jiangmaSeatId()
	if jiangmaSeatId != -1 {
		for i := range mj.horses {
			mj.horses[i] = room.CardSet().Deal()
		}
	}

	room.Award()
}

func (mj *chaoshanMahjong) Award() {
	room := mj.room
	unit := room.Unit()

	type HorseResult struct {
		Goods []int // 赢钱的马
		Bads  []int // 输钱的牌
	}
	horseResults := make([]HorseResult, room.NumSeat())
	// 有人胡牌
	if room.CanPlay(OptHuangZhuangBuHuangGang) || len(room.winPlayers) > 0 {
		for i := 0; i < room.NumSeat(); i++ {
			p := room.GetPlayer(i)

			kongHorses := 0
			if room.CanPlay(OptMaGenGang) && len(room.winPlayers) > 0 {
				if p.isWin {
					for _, c := range mj.horses {
						seatId := mj.winHorse(c)
						if seatId == i {
							kongHorses++

							jiangmaSeatId := mj.jiangmaSeatId()
							result := &horseResults[jiangmaSeatId]
							result.Goods = append(result.Goods, c)
						}
					}
				}
				/*for _, c := range mj.maima[i] {
					seatId := mj.winHorse(c)
					if seatId == i {
						kongHorses++
						result.Goods = append(result.Goods, c)
					}
				}
				if p == room.dealer {
					for _, c := range mj.fama {
						seatId := mj.winHorse(c)
						if seatId == i {
							kongHorses++
							result.Goods = append(result.Goods, c)
						}
					}
				}
				*/
			}

			detail := ChipDetail{Seats: 1 << uint(p.GetSeatIndex())}
			maDetail := ChipDetail{Seats: 1 << uint(p.GetSeatIndex()), Operate: OpMaGenGang}
			for _, kong := range p.kongHistory {
				detail.Operate = kong.Type

				bills := make([]Bill, room.NumSeat())
				maBills := make([]Bill, room.NumSeat())
				switch kong.Type {
				case mjutils.MeldInvisibleKong, mjutils.MeldBentKong:
					times := 1
					// 暗杠
					if kong.Type == mjutils.MeldInvisibleKong {
						times = 2
					}
					for k := 0; k < room.NumSeat(); k++ {
						bill := &bills[k]
						maBill := &maBills[k]
						if other := room.GetPlayer(k); other != nil && p != other {
							detail.Times = times
							detail.Chip = -int64(detail.Times) * unit
							bill.Details = append(bill.Details, detail)

							maDetail.Points = times
							maDetail.Times = times * kongHorses
							maDetail.Chip = -int64(maDetail.Times) * unit
							maBill.Details = append(maBill.Details, maDetail)
						}
					}
				case mjutils.MeldStraightKong:
					// 直杠
					times := 3
					bill := &bills[kong.other.GetSeatIndex()]
					detail.Times = times
					detail.Chip = -int64(detail.Times) * unit
					bill.Details = append(bill.Details, detail)

					maBill := &maBills[kong.other.GetSeatIndex()]
					maDetail.Points = times
					maDetail.Times = times * kongHorses
					maDetail.Chip = -int64(maDetail.Times) * unit
					maBill.Details = append(maBill.Details, maDetail)
				}
				room.Billing(bills)
				if kongHorses > 0 {
					room.Billing(maBills)
				}
			}
		}
	}
	if boom := room.boomPlayer(); room.CanPlay(OptShiBeiBuJiFen) && boom != nil {
		score := PingHu
		for _, opt := range boom.CheckWin() {
			if chaoshanScoreList[score] < chaoshanScoreList[opt.Chip] {
				score = opt.Chip
			}
		}
		if chaoshanScoreList[score] >= 10 {
			return
		}
	}

	// 胡牌
	for _, p := range room.winPlayers {
		bills := make([]Bill, room.NumSeat())

		detail := ChipDetail{Seats: 1 << uint(p.GetSeatIndex()), Operate: mjutils.OperateWin}
		addition2 := make(map[string]int)

		copyCards := p.copyCards()
		chip, points := mj.Score(copyCards, p.melds)

		times := points
		for _, t := range addition2 {
			times += t
		}

		// 抢杠胡包胡牌分
		if p.IsRobKong() && room.CanPlay(OptQiangGangQuanBao) {
			if extra := room.NumSeat() - 2; extra > 0 {
				addition2["QGQB"] = 0

				times += points * extra
			}
		}
		// 奖/买/罚马
		{
			boom := room.boomPlayer()
			maima := func(masterSeatId int, horses []int, operate int) {
				var result = &horseResults[masterSeatId]
				var jiepaoNum, fangpaoNum, zimoNum, meizimoNum int

				winSeatId := p.GetSeatIndex()
				if operate == OpJiangMa {
					winSeatId = mj.jiangmaSeatId()
					result = &horseResults[winSeatId]
				}
				for _, c := range horses {
					otherSeatId := mj.winHorse(c)
					if p.drawCard == -1 && otherSeatId == winSeatId && masterSeatId != boom.GetSeatIndex() {
						jiepaoNum++
						result.Goods = append(result.Goods, c)
					}
					if p.drawCard == -1 && otherSeatId == boom.GetSeatIndex() &&
						masterSeatId != p.GetSeatIndex() && operate != OpJiangMa {
						fangpaoNum++
						result.Bads = append(result.Bads, c)
					}
					if p.drawCard != -1 && otherSeatId == winSeatId {
						zimoNum++
						result.Goods = append(result.Goods, c)
					}
					if p.drawCard != -1 && otherSeatId != winSeatId && operate != OpJiangMa {
						meizimoNum++
						result.Bads = append(result.Bads, c)
					}
				}
				d := ChipDetail{Operate: operate, Points: points}
				d.Addition2 = map[string]int{"MasterSeatId": masterSeatId}
				if jiepaoNum > 0 && boom != nil {
					d.Seats = 1 << uint(masterSeatId)
					d.Times = jiepaoNum * points
					d.Chip = -unit * int64(d.Times)

					myBills := make([]Bill, room.NumSeat())
					b := &myBills[boom.GetSeatIndex()]
					b.Details = append(b.Details, d)
					room.Billing(myBills)
				}
				if fangpaoNum > 0 {
					d.Seats = 1 << uint(p.GetSeatIndex())
					d.Times = fangpaoNum * points
					d.Chip = -unit * int64(d.Times)

					myBills := make([]Bill, room.NumSeat())
					b := &myBills[masterSeatId]
					b.Details = append(b.Details, d)
					room.Billing(myBills)
				}
				if meizimoNum > 0 {
					d.Seats = 1 << uint(p.GetSeatIndex())
					d.Times = meizimoNum * points
					d.Chip = -unit * int64(d.Times)

					myBills := make([]Bill, room.NumSeat())
					b := &myBills[masterSeatId]
					b.Details = append(b.Details, d)
					room.Billing(myBills)
				}

				myBills := make([]Bill, room.NumSeat())
				for i := 0; i < room.NumSeat(); i++ {
					other := room.GetPlayer(i)
					if zimoNum > 0 && p != other {
						d.Seats = 1 << uint(masterSeatId)
						d.Times = zimoNum * points
						d.Chip = -unit * int64(d.Times)
						b := &myBills[i]
						b.Details = append(b.Details, d)
					}
				}
				room.Billing(myBills)
			}
			// 奖马
			maima(p.GetSeatIndex(), mj.horses, OpJiangMa)

			// 买马
			for i := 0; i < room.NumSeat(); i++ {
				maima(i, mj.maima[i], OpMaiMa)
			}
			// 罚马
			maima(room.dealer.GetSeatIndex(), mj.fama, OpFaMa)
		}
		// 放炮胡
		if p.drawCard == -1 {
			addition2["JP"] = 0
		}
		// 自摸
		if p.drawCard != -1 {
			addition2["ZM"] = 0
		}
		// 连庄
		if t := p.continuousDealerTimes; t > 0 && room.CanPlay(OptLianZhuang) {
			addition2["LianZhuang"] = t
			times += t
		}

		detail.Chip = int64(chip)
		detail.Points = points
		detail.Times = times
		detail.Chip = -unit * int64(times)
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

	jiangmaSeatId := mj.jiangmaSeatId()
	room.Broadcast("BuyHorses", map[string]any{
		"Fama":          mj.fama,
		"Maima":         mj.maima,
		"Jiangma":       mj.horses,
		"JiangmaSeatId": jiangmaSeatId,
		"ResultSet":     horseResults,
	})
}

func (mj *chaoshanMahjong) GameOver() {
	for i := range mj.horses {
		mj.horses[i] = 0
	}
	for i := range mj.fama {
		mj.fama[i] = 0
	}
	for i := range mj.maima {
		for k := range mj.maima[i] {
			mj.maima[i][k] = 0
		}
	}
}

type chaoshanMahjongWorld struct {
}

func NewChaoshanMahjongWorld() *chaoshanMahjongWorld {
	return &chaoshanMahjongWorld{}
}

func (w *chaoshanMahjongWorld) NewRoom(id, subId int) *roomutils.Room {
	r := NewMahjongRoom(id, subId)
	r.SetNoPlay(OptBoom)
	r.SetNoPlay(OptMaGenGang)
	r.SetNoPlay(OptShengYiQuanBuPeng)
	r.SetNoPlay(OptBiHu)
	r.SetNoPlay(OptXiaoHu)
	r.SetNoPlay(OptShiBeiBuJiFen)
	r.SetNoPlay(OptLianZhuang)

	r.SetNoPlay("jiangma_2")
	r.SetNoPlay("jiangma_5")
	r.SetNoPlay("jiangma_8")

	r.SetNoPlay("maima_1")
	r.SetNoPlay("maima_2")

	r.SetNoPlay("fama_1")
	r.SetNoPlay("fama_2")

	r.SetNoPlay("fengding_5")
	r.SetNoPlay("fengding_10")

	r.SetNoPlay("seat_2")
	r.SetNoPlay("seat_3")
	r.SetPlay("seat_4")

	r.SetNoPlay(OptHuangZhuangBuHuangGang)
	r.SetNoPlay(OptJiHuBuNengChiHu)
	r.SetPlay(OptAbleRobKong)
	r.SetPlay(OptSevenPairs)
	r.SetPlay(OptShiSanYao)

	local := NewchaoshanMahjong()
	local.room = r
	r.localMahjong = local
	return r.Room
}

func (w *chaoshanMahjongWorld) GetName() string {
	return "chaoshanmj"
}

func (w *chaoshanMahjongWorld) NewPlayer() *service.Player {
	p := NewMahjongPlayer()
	p.Player = service.NewPlayer(p)
	p.localObj = &chaoshanObj{MahjongPlayer: p}
	return p.Player
}

// chaoshanObj 潮汕麻将玩家逻辑
type chaoshanObj struct {
	*MahjongPlayer
}

func (obj *chaoshanObj) OnDiscard() {
	room := obj.Room()
	dealer := room.dealer
	unit := room.Unit()

	genZhuang := true
	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		if len(p.melds) > 0 ||
			len(dealer.discardHistory) != 1 ||
			len(p.discardHistory) != 1 ||
			dealer.discardHistory[0] != p.discardHistory[0] {
			genZhuang = false
		}
	}
	// 跟庄
	if false && genZhuang {
		bills := make([]Bill, room.NumSeat())
		bill := &bills[dealer.GetSeatIndex()]
		for i := 0; i < room.NumSeat(); i++ {
			if p := room.GetPlayer(i); p != nil && p != dealer {
				detail := ChipDetail{
					Seats:   1 << uint(p.GetSeatIndex()),
					Operate: OpGenZhuang,
					Points:  1,
					Times:   1,
				}
				detail.Chip = -int64(detail.Times) * unit
				bill.Details = append(bill.Details, detail)
			}
		}
		room.Billing(bills)
	}
}

func (obj *chaoshanObj) IsAblePong() bool {
	room := obj.Room()
	if room.CanPlay(OptShengYiQuanBuPeng) &&
		room.NumSeat()+room.helper.ReserveCardNum >= room.CardSet().Count() {
		return false
	}
	return obj.MahjongPlayer.IsAblePong()
}

func (obj *chaoshanObj) IsAbleWin() bool {
	if !obj.MahjongPlayer.IsAbleWin() {
		return false
	}

	room := obj.Room()
	mj := room.localMahjong.(*chaoshanMahjong)
	copyCards := obj.copyCards()
	score, _ := mj.Score(copyCards, obj.melds)
	if room.CanPlay(OptJiHuBuNengChiHu) && obj.drawCard == -1 && score == PingHu {
		return false
	}
	// 自摸胡
	if !mj.isBigBoom && obj.drawCard == -1 {
		if chaoshanScoreList[score] < 20 {
			return false
		}
	}
	return true
}
