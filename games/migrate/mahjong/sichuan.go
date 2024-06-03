package mahjong

import (
	"gofishing-game/games/migrate/internal/cardrule"
	"gofishing-game/internal/cardutils"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
	"slices"
	"strings"

	"github.com/guogeer/quasar/config"
)

func init() {
	{
		w := NewSichuanWorld("xlch")
		service.AddWorld(w)
		AddHandlers(w.GetName())

		var cards []int
		for _, c := range []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 21, 22, 23, 24, 25, 26, 27, 28, 29, 41, 42, 43, 44, 45, 46, 47, 48, 49} {
			cards = append(cards, c, c, c, c)
		}
		cardutils.AddCardSystem(w.GetName(), cards)
	}
	{
		w := NewSichuanWorld("szdd")
		service.AddWorld(w)
		AddHandlers(w.GetName())

		var cards []int
		for _, c := range []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 21, 22, 23, 24, 25, 26, 27, 28, 29, 41, 42, 43, 44, 45, 46, 47, 48, 49} {
			cards = append(cards, c, c, c, c)
		}
		cardutils.AddCardSystem(w.GetName(), cards)
	}
}

type SichuanMahjong struct {
	room *MahjongRoom

	copyScoreList, scoreList       map[int]int
	copyAdditionList, additionList map[string]int
	LimitPoints                    int
}

// 金币模式
func NewSichuanMahjong() *SichuanMahjong {
	sc := &SichuanMahjong{
		copyScoreList: map[int]int{
			PingHu:          1,
			DuiDuiHu:        2,
			QingYiSe:        4,
			DaiYaoJiu:       4,
			QiDui:           4,
			JinGouDiao:      4,
			QingDui:         8,
			JiangDui:        8,
			LongQiDui:       16,
			QingQiDui:       16,
			QingYaoJiu:      16,
			JiangJinGouDiao: 16,
			QingJinGouDiao:  16,
			QingLongQiDui:   32,
			TianHu:          32,
			DiHu:            32,
			ShiBaLuoHan:     64,
			QingShiBaLuoHan: 256,
		},
		copyAdditionList: map[string]int{
			"自摸":  1,
			"根":   1,
			"杠上花": 1,
			"抢杠胡": 1,
			"杠上炮": 1,
		},
	}
	return sc
}

// 积分模式
func NewSichuanMahjongEx() *SichuanMahjong {
	sc := &SichuanMahjong{
		copyScoreList: map[int]int{
			PingHu:        1,
			DuiDuiHu:      2,
			QingYiSe:      4,
			QiDui:         4,
			LongQiDui:     8,
			QingDui:       16,
			QingLongQiDui: 32,
			DaiYaoJiu:     8,
			JiangDui:      8,
			QingQiDui:     8,
			JiangQiDui:    16,
			MenQing:       2,
			DuanYaoJiu:    2,
		},
		copyAdditionList: map[string]int{
			"自摸":   1, // 自摸
			"根":    1, // 根
			"抢杠胡":  1, // 抢杠胡
			"杠上花":  1, // 杠上花
			"杠上炮":  1, // 杠上炮
			"妙手回春": 1, // 妙手回春，最后一张牌自摸
			"金钩钓":  1, // 金钩钓
			"海底捞月": 1, // 海底捞月，最后一张牌接炮
			"天胡":   3, // 天胡
			"地胡":   2, // 地胡
		},
	}
	return sc
}

type XlchMahjong struct {
	*SichuanMahjong
}

type XzddMahjong struct {
	*SichuanMahjong
}

func (sc *SichuanMahjong) GetPoints(score int) int {
	points := sc.scoreList[score]
	return points
}

func (sc *SichuanMahjong) GetAddition(name string) int {
	unit := sc.additionList[name]
	return unit
}

func (sc *SichuanMahjong) SetPoints(score, points int) {
	sc.scoreList[score] = points
}

func (sc *SichuanMahjong) SetAddition(name string, unit int) {
	sc.additionList[name] = unit
}

func (sc *SichuanMahjong) OnCreateRoom() {
	room := sc.room
	sc.scoreList = map[int]int{}
	sc.additionList = map[string]int{}
	for k, v := range sc.copyScoreList {
		sc.scoreList[k] = v
	}
	for k, v := range sc.copyAdditionList {
		sc.additionList[k] = v
	}

	// log.Debug("room reset")
	room.SetPlay(OptDianGangHuaZiMo)
	if room.CanPlay(OptDianGangHuaFangPao) {
		room.SetNoPlay(OptDianGangHuaZiMo)
	}

	if !room.CanPlay(OptTianDiHu) {
		sc.SetAddition("天胡", 0)
		sc.SetAddition("地胡", 0)
	}
	if !room.CanPlay(OptMenQingZhongZhang) {
		sc.SetPoints(MenQing, 0)
		sc.SetPoints(DuanYaoJiu, 0)
	}
	if !room.CanPlay(OptYaoJiuJiangDui) {
		sc.SetPoints(JiangQiDui, 0)
		sc.SetPoints(JiangDui, 0)
		sc.SetPoints(DaiYaoJiu, 0)
	}
}

func (sc *SichuanMahjong) OnEnter(comer *MahjongPlayer) {
	room := sc.room
	// 正在游戏中
	if room.Status == roomStatusExchangeTriCards {
		data := map[string]any{"ts": room.deadline.Unix(), "triCards": comer.defaultTriCards}
		if comer.triCards[0] > 0 {
			data["myTriCards"] = comer.triCards
		}
		var others []int
		for i := 0; i < room.NumSeat(); i++ {
			if other := room.GetPlayer(i); other != comer && other.triCards[0] > 0 {
				others = append(others, i)
			}
		}
		if len(others) > 0 {
			data["others"] = others
		}
		comer.WriteJSON("startExchangeTriCards", data)
	}
	if room.Status == roomStatusChooseColor {
		data := map[string]any{"ts": room.deadline.Unix(), "color": comer.defaultColor, "myColor": comer.discardColor}
		if comer.discardColor != -1 {
			data["myColor"] = comer.discardColor
		}
		var others []int
		for i := 0; i < room.NumSeat(); i++ {
			if other := room.GetPlayer(i); comer != other && other.discardColor != -1 {
				others = append(others, i)
			}
		}
		if len(others) > 0 {
			data["others"] = others
		}

		comer.WriteJSON("startChooseColor", data)
	}
}

func (sc *SichuanMahjong) OnReady() {
	room := sc.room

	room.StartDealCard()

	// 换三张
	if room.CanPlay(OptHuanSanZhang) {
		room.StartExchangeTriCards()
	} else if room.CanPlay(OptChooseColor) {
		room.StartChooseColor()
	} else {
		room.dealer.OnDraw()
	}
}

func (sc *SichuanMahjong) OnWin() {
	room := sc.room
	// 结算
	unit := room.Unit()
	kongPlayer := room.kongPlayer
	discardPlayer := room.discardPlayer
	additionId, effectSeatId, failSeatId := -1, -1, -1

	var bills = make([]Bill, room.NumSeat())
	var moveKong = make([]Bill, room.NumSeat()) // 呼叫转移
	for _, p := range room.winPlayers {
		copyCards := p.copyCards()
		score, points := sc.Score(copyCards[:], p.melds)
		// log.Info("+++++++++++++++++++++++++++++++++++", score, points)

		// 另计番
		addition2 := map[string]int{}
		var score2, points2 int
		// 庄家摸第一张牌胡牌，天胡
		if p == room.dealer && len(p.drawHistory) == 1 && p.drawCard != -1 {
			if t := sc.GetPoints(TianHu); t > 0 {
				score2, points2 = TianHu, t
			}
			if t := sc.GetAddition("天胡"); t > 0 {
				addition2["天胡"] = t
			}
		}
		// 庄家打第一张牌闲家胡牌，地胡
		if p != room.dealer && p.discardNum == 0 && len(room.dealer.drawHistory) == 1 && room.dealer == room.discardPlayer {
			if t := sc.GetPoints(DiHu); t > 0 {
				score2, points2 = DiHu, t
			}
			if t := sc.GetAddition("地胡"); t > 0 {
				addition2["地胡"] = t
			}
		}

		// 门清
		menQing := (p.drawCard != -1)
		for _, m := range p.melds {
			if m.Type != cardrule.MeldInvisibleKong {
				menQing = false
			}
		}
		if t := sc.GetPoints(MenQing); menQing && t > 0 {
			score2, points2 = MenQing, t
		}
		if points < points2 {
			score = score2
		}
		addition2["根"] = sc.CountGen(score, copyCards[:], p.melds)

		if t := sc.GetAddition("金钩钓"); t > 0 && len(p.melds) == 4 {
			addition2["金钩钓"] = t
		}
		if t := sc.GetAddition("海底捞月"); t > 0 && p.drawCard == -1 && room.CardSet().Count() == 0 {
			addition2["海底捞月"] = t
		}
		if t := sc.GetAddition("妙手回春"); t > 0 && p.drawCard != -1 && room.CardSet().Count() == 0 {
			addition2["妙手回春"] = t
		}

		// detail := ChipChip{SeatIndex: p.GetSeatIndex(), Operate: cardrule.OperateWin, Score: score}
		detail := ChipDetail{Seats: 1 << uint(p.GetSeatIndex()), Operate: cardrule.OperateWin, Chip: int64(score)}
		// 接跑
		if p.drawCard == -1 {
			// 胡牌
			if discardPlayer != nil {
				failSeatId = discardPlayer.GetSeatIndex()
			}
			// 抢杠胡，别人杠牌，没人出牌
			if kongPlayer != nil && kongPlayer != p && discardPlayer == nil {
				failSeatId = kongPlayer.GetSeatIndex()
				addition2["抢杠胡"] = 1
			}
			// 杠上炮
			if kongPlayer != nil && discardPlayer == kongPlayer {
				failSeatId = discardPlayer.GetSeatIndex()
				addition2["杠上炮"] = 1

				// 额外赔偿杠所得
				effectSeatId = p.GetSeatIndex()
				bill := &moveKong[failSeatId]
				bill.Details = append(bill.Details, ChipDetail{Seats: 1 << uint(p.GetSeatIndex()), Chip: -kongPlayer.lastKong.Chip, Operate: cardrule.OperateMoveKong, Times: 1})
			}
			detail.Addition2 = addition2
			bill := &bills[failSeatId]
			bill.Details = append(bill.Details, detail)
		} else {
			boomId := -1
			// 一个策划一个坑，杠上开花维佳计自摸，严程不计
			// 杠上开花计自摸
			if kongPlayer == p {
				addition2["杠上花"] = 1
				// 点杠花算点炮
				if boom := p.lastKong.other; boom != nil && room.CanPlay(OptDianGangHuaFangPao) {
					boomId = boom.GetSeatIndex()
				}
			}
			// 自摸加底
			if room.CanPlay(OptZiMoJiaDi) {
				addition2["自摸"] = 0
				addition2["自摸加底"] = 0
			} else {
				addition2["自摸"] = 1
			}

			// 天胡、地胡不算自摸
			if score == TianHu || score == DiHu {
				addition2["自摸加底"] = 0
			}

			for k := 0; k < room.NumSeat(); k++ {
				add := map[string]int{}
				for k, v := range addition2 {
					add[k] = v
				}
				other := room.GetPlayer(k)
				if p == other || other.leaveGame {
					continue
				}
				// 点杠花点炮一个人出钱
				if boomId != -1 && boomId != other.GetSeatIndex() {
					continue
				}
				detail.Addition2 = add
				bill := &bills[other.GetSeatIndex()]
				bill.Details = append(bill.Details, detail)
			}
		}
	}

	// 算番
	for i := 0; i < len(bills); i++ {
		bill := &bills[i]
		for j := 0; j < len(bill.Details); j++ {
			detail := &bill.Details[j]

			total := 0
			// addition := detail.Addition
			addition2 := detail.Addition2
			for _, t := range addition2 {
				total += t
			}
			if _, ok := addition2["自摸"]; ok {
				additionId = ZiMo
			}
			if _, ok := addition2["杠上炮"]; ok {
				additionId = GangShangPao
			}
			if _, ok := addition2["杠上花"]; ok {
				additionId = GangShangHua
			}

			points := sc.GetPoints(int(detail.Chip))
			times := points * int(1<<(uint(total)))
			// 自摸加底
			if _, ok := addition2["自摸加底"]; ok {
				times++
				delete(addition2, "自摸加底")
			}

			if times > sc.LimitPoints && sc.LimitPoints > 0 {
				times = sc.LimitPoints
			}

			gold := unit * int64(times)
			detail.Times, detail.Chip, detail.Addition2 = times, -gold, addition2
		}
	}
	room.Billing(bills)

	var result []ChipResult
	for seatId, bill := range bills {
		if len(bill.Details) > 0 {
			result = append(result, ChipResult{SeatIndex: seatId, Chip: bill.Sum()})
		}
	}
	var wins []int
	for _, p := range room.winPlayers {
		wins = append(wins, p.Id)
	}
	room.Broadcast("compute", map[string]any{"operate": cardrule.OperateWin, "addition": additionId, "winPlayers": wins, "winCard": room.lastCard, "result": result})

	if effectSeatId != -1 {
		result = nil
		room.Billing(moveKong[:])
		var result []ChipResult
		for seatId, bill := range moveKong {
			if len(bill.Details) > 0 {
				result = append(result, ChipResult{SeatIndex: seatId, Chip: bill.Sum()})
			}
		}
		room.Broadcast("compute", map[string]any{"operate": cardrule.OperateMoveKong, "result": result})
	}

	// 破产玩家可提前离开游戏
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil && p.isBust {
			p.leaveGame = true
		}
	}

	for i, p := range room.winPlayers {
		c := room.lastCard
		p.rhHistory = append(p.rhHistory, c)

		if i > 0 {
			c += 1000
		}
		p.winHistory = append(p.winHistory, c)
	}
}

func (h *SichuanMahjong) Award() {
	room := h.room
	// 查花猪
	var b = make([]bool, room.NumSeat())
	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		if !p.leaveGame && HasColor(roomutils.GetServerName(room.SubId), p.handCards, p.discardColor) {
			b[p.GetSeatIndex()] = true
		}
	}
	var gold int64
	var unit = room.Unit()
	var bills = make([]Bill, room.NumSeat())
	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		if !b[p.GetSeatIndex()] {
			continue
		}
		times := 16
		if times > h.LimitPoints && h.LimitPoints > 0 {
			times = h.LimitPoints
		}
		gold = unit * int64(times)
		bill := &bills[p.GetSeatIndex()]
		for k := 0; k < room.NumSeat(); k++ {
			other := room.GetPlayer(k)
			if other.leaveGame || b[other.GetSeatIndex()] || p == other {
				continue
			}
			detail := ChipDetail{Seats: 1 << uint(other.GetSeatIndex()), Chip: -gold, Operate: cardrule.OperateHuaZhu, Times: times}
			bill.Details = append(bill.Details, detail)
		}
	}
	room.Billing(bills)

	var maxPoints = make([]int, room.NumSeat())
	bills = make([]Bill, room.NumSeat())
	// 查大叫
	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		if p.leaveGame || p.isWin || b[p.GetSeatIndex()] {
			continue
		}
		opts := p.CheckWin()
		if len(opts) == 0 {
			maxPoints[p.GetSeatIndex()] = -1
		} else {
			for _, w := range opts {
				if maxPoints[p.GetSeatIndex()] < w.Points {
					maxPoints[p.GetSeatIndex()] = w.Points
				}
			}
		}
	}

	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		if maxPoints[p.GetSeatIndex()] != -1 {
			continue
		}
		for k := 0; k < room.NumSeat(); k++ {
			other := room.GetPlayer(k)
			if points := maxPoints[other.GetSeatIndex()]; points > 0 {
				other.totalTimes["大叫"]++
				bill := &bills[p.GetSeatIndex()]
				times := points
				if times > h.LimitPoints && h.LimitPoints > 0 {
					times = h.LimitPoints
				}
				gold = unit * int64(times)
				detail := ChipDetail{Seats: 1 << uint(other.GetSeatIndex()), Chip: -gold, Operate: cardrule.OperateDaJiao, Times: times}
				bill.Details = append(bill.Details, detail)
			}
		}
	}
	room.Billing(bills)

	// 刮风下雨返还
	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		if p.isWin || p.leaveGame {
			continue
		}

		bills = make([]Bill, room.NumSeat())
		// 花猪、未听牌
		if HasColor(roomutils.GetServerName(room.SubId), p.handCards, p.discardColor) || p.CheckWin() == nil {
			bill := &bills[p.GetSeatIndex()]
			for k, g := range p.kongChip {
				other := room.GetPlayer(k)
				if g >= 0 || other.leaveGame {
					continue
				}
				detail := ChipDetail{Operate: cardrule.OperateBackKong, Times: 1, Seats: 1 << uint(k), Chip: g}
				// bill.Chip -= g
				bill.Details = append(bill.Details, detail)
			}
		}
		room.Billing(bills)
	}

	room.expectDiscardPlayer = nil
	room.expectKongPlayer = nil
	room.expectPongPlayer = nil
	room.expectWinPlayers = map[int]*MahjongPlayer{}
}

func (h *SichuanMahjong) GameOver() {
}

func (sc *SichuanMahjong) CountGen(score int, cards []int, melds []cardrule.Meld) int {
	allCards := cardutils.GetCardSystem(roomutils.GetServerName(sc.room.SubId)).GetAllCards()
	var all [MaxCard]int
	for _, c := range allCards {
		all[c] = cards[c]
	}
	for _, m := range melds {
		switch m.Type {
		case cardrule.MeldTriplet:
			all[m.Card] += 3
		case cardrule.MeldStraightKong, cardrule.MeldBentKong, cardrule.MeldInvisibleKong:
			all[m.Card] += 4
		}
	}
	gen := 0
	for _, c := range allCards {
		if all[c] == 4 {
			gen++
		}
	}
	// 部分番型已经包括根，需要减去
	switch score {
	case TianHu, DiHu:
		gen = 0
	case LongQiDui, QingLongQiDui:
		gen -= 1
	case ShiBaLuoHan, QingShiBaLuoHan:
		gen -= 4
	}
	return gen
}

func (sc *SichuanMahjong) Score(cards []int, melds []cardrule.Meld) (int, int) {
	var pairNum, pair2Num, kongNum, color int
	for _, c := range cardutils.GetCardSystem(roomutils.GetServerName(sc.room.SubId)).GetAllCards() {
		pairNum += cards[c] / 2
		pair2Num += cards[c] / 4
		if cards[c] > 0 {
			color = color | int(1<<uint(c/10))
		}
	}
	for _, m := range melds {
		color = color | int(1<<uint(m.Card/10))
	}

	// 清一色
	var isSameColor bool
	if color&(color-1) == 0 {
		isSameColor = true
	}

	for _, c := range cardutils.GetCardSystem(roomutils.GetServerName(sc.room.SubId)).GetAllCards() {
		kongNum += cards[c] / 4
	}
	for _, meld := range melds {
		if meld.Type == cardrule.MeldStraightKong || meld.Type == cardrule.MeldBentKong || meld.Type == cardrule.MeldInvisibleKong {
			kongNum++
		}
	}

	scores := make(map[int]bool)
	// 七对
	if pairNum == 7 {
		scores[QiDui] = true
		if pair2Num > 0 {
			scores[LongQiDui] = true
		}
		if isSameColor {
			scores[QingQiDui] = true
		}
		if pair2Num > 0 && isSameColor {
			scores[QingLongQiDui] = true
		}
		if CountCardsByValue(roomutils.GetServerName(sc.room.SubId), cards, melds, 2, 5, 8) == 14 {
			scores[JiangQiDui] = true
		}
	}

	if isSameColor {
		scores[QingYiSe] = true
	}

	if len(melds) == 4 && pairNum == 1 {
		scores[JinGouDiao] = true
		if CountCardsByValue(roomutils.GetServerName(sc.room.SubId), cards, nil, 2, 5, 8) == 2 && CountMeldsByValue(melds, 2, 5, 8) == 4 {
			scores[JiangJinGouDiao] = true
		}
		if isSameColor {
			scores[QingJinGouDiao] = true
		}
		if kongNum == 4 {
			scores[ShiBaLuoHan] = true
		}
		if kongNum == 4 && isSameColor {
			scores[QingShiBaLuoHan] = true
		}
	}

	room := sc.room
	for _, pair := range cardutils.GetCardSystem(roomutils.GetServerName(room.SubId)).GetAllCards() {
		if cards[pair] < 2 {
			continue
		}
		cards[pair] -= 2
		for _, opt := range room.helper.Split(cards) {
			// 顺子
			seq := CountMeldsByType(melds, cardrule.MeldSequence)
			seq += CountMeldsByType(opt.Melds, cardrule.MeldSequence)
			if seq > 0 {
				scores[PingHu] = true
			} else {
				scores[DuiDuiHu] = true
				if isSameColor {
					scores[QingDui] = true
				}
			}
			count19 := CountMeldsByValue(melds, 1, 9)
			count19 += CountMeldsByValue(opt.Melds, 1, 9)
			if count19 == 4 && IsSameValue(pair, 1, 9) {
				scores[DaiYaoJiu] = true
				if isSameColor {
					scores[QingYaoJiu] = true
				}
			}
			count258 := CountMeldsByValue(melds, 2, 5, 8)
			count258 += CountMeldsByValue(opt.Melds, 2, 5, 8)
			if seq == 0 && count258 == 4 && IsSameValue(pair, 2, 5, 8) {
				scores[JiangDui] = true
			}
		}
		cards[pair] += 2
	}
	// 断幺九
	if CountCardsByValue(roomutils.GetServerName(sc.room.SubId), cards, melds, 1, 9) == 0 {
		scores[DuanYaoJiu] = true
	}

	var bestScoreId, bestPoints int
	for k := range scores {
		if points := sc.scoreList[k]; bestPoints < points {
			bestScoreId = k
			bestPoints = points
		}
	}

	// 另计番
	if t := sc.GetAddition("另计番"); scores[JinGouDiao] && t > 0 {
		bestPoints = bestPoints << uint(t)
	}
	if gen := sc.CountGen(bestScoreId, cards, melds); gen > 0 {
		bestPoints = bestPoints << uint(gen)
	}
	return bestScoreId, bestPoints
}

// 血流成河
func (xlch *XlchMahjong) OnWin() {
	room := xlch.room
	xlch.SichuanMahjong.OnWin()
	// 下家摸牌
	room.delayDuration += maxDelayAfterWin
	if room.CountBustPlayers() == 0 {
		room.Turn()
	} else {
		room.bustTimeout = room.Turn
	}
	room.delayDuration -= maxDelayAfterWin
}

// 血战到底
// 血战到底胡牌时增加等待玩家破产
func (xzdd *XzddMahjong) OnWin() {
	room := xzdd.room
	xzdd.SichuanMahjong.OnWin()

	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil && p.isWin {
			p.leaveGame = true
		}
	}

	// 下家摸牌
	if room.CountBustPlayers() == 0 {
		room.Turn()
	} else {
		room.bustTimeout = room.Turn
	}
	// room.Turn()
}

type SichuanWorld struct {
	name string
}

func NewSichuanWorld(name string) *SichuanWorld {
	return &SichuanWorld{name: name}
}

func (w *SichuanWorld) GetName() string {
	return w.name
}

func (w *SichuanWorld) NewRoom(subId int) *roomutils.Room {
	r := NewMahjongRoom(subId)
	r.SetPlay(OptZiMoJiaFan)
	r.SetNoPlay(OptDianGangHuaFangPao) // 点杠花放炮
	r.SetPlay(OptDianGangHuaZiMo)      // 默认点杠花自摸
	r.SetPlay(OptTianDiHu)

	r.SetNoPlay(OptZiMoJiaDi)
	r.SetNoPlay(OptHuanSanZhang)
	r.SetNoPlay(OptMenQingZhongZhang)
	r.SetNoPlay(OptYaoJiuJiangDui)
	r.SetNoPlay(OptHuanSanZhang)

	// 换三张场次
	tags, _ := config.String("room", subId, "tags")
	if slices.Index(strings.Split(tags, ","), "exchange_tri_cards") >= 0 {
		r.SetPlay(OptHuanSanZhang)
	}
	r.SetPlay(OptBoom)
	r.SetPlay(OptChooseColor)
	r.SetPlay(OptCostAfterKong)
	r.SetPlay(OptStraightKong2)
	r.SetPlay(OptAbleRobKong)
	r.SetPlay(OptSevenPairs)

	sc := NewSichuanMahjong()
	if r.IsTypeScore() {
		sc = NewSichuanMahjongEx()
	}
	sc.room = r

	// log.Debug(w.GetName())
	r.localMahjong = &XlchMahjong{SichuanMahjong: sc}
	if slices.Index(strings.Split(tags, ","), "xzdd") >= 0 {
		r.localMahjong = &XzddMahjong{SichuanMahjong: sc}
	}
	return r.Room
}

func (w *SichuanWorld) NewPlayer() *service.Player {
	p := NewMahjongPlayer()
	p.Player = service.NewPlayer(p)
	p.localObj = p

	return p.Player
}
