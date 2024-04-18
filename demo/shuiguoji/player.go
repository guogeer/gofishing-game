package shuiguoji

// 可以连线的水果机
// Guogeer 2018-02-08

import (
	"gofishing-game/service"
	"strings"

	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
	"github.com/guogeer/quasar/randutil"
	"github.com/guogeer/quasar/script"
	"github.com/guogeer/quasar/utils"
)

const (
	gPointX = 5
	gPointY = 3
)

const (
	// 香蕉-西瓜-橄榄-葡萄-橙子-铃铛-樱桃-BAR-WILD-BONUS-7
	itemBanana = iota
	itemWaterMelon
	itemOlive
	itemGrape
	itemOrange
	itemBell
	itemCherry
	itemBar
	itemWild
	itemBonus
	item7
	AllItemNum
)

var gAllLines = [][][]int{
	{},
	{{0, 1}, {1, 1}, {2, 1}, {3, 1}, {4, 1}}, // line 1
	{{0, 0}, {1, 0}, {2, 0}, {3, 0}, {4, 0}}, // line 2
	{{0, 2}, {1, 2}, {2, 2}, {3, 2}, {4, 2}}, // line 3
	{{0, 0}, {1, 1}, {2, 2}, {3, 1}, {4, 0}}, // line 4
	{{0, 2}, {1, 1}, {2, 0}, {3, 1}, {4, 2}}, // line 5
	{{0, 0}, {1, 0}, {2, 1}, {3, 2}, {4, 2}}, // line 6
	{{0, 2}, {1, 2}, {2, 1}, {3, 0}, {4, 0}}, // line 7
	{{0, 1}, {1, 2}, {2, 1}, {3, 0}, {4, 1}}, // line 8
	{{0, 1}, {1, 0}, {2, 1}, {3, 2}, {4, 1}}, // line 9
}

var gAllMultiples = [][]int{
	{0, 0, 1, 3, 10, 75},      // 香蕉
	{0, 0, 0, 3, 10, 85},      // 西瓜
	{0, 0, 0, 15, 40, 250},    // 橄榄
	{0, 0, 0, 25, 50, 400},    // 葡萄
	{0, 0, 0, 30, 70, 550},    // 橙子
	{0, 0, 0, 35, 80, 650},    // 铃铛
	{0, 0, 0, 45, 100, 800},   // 樱桃
	{0, 0, 0, 75, 175, 1250},  // BAR
	{0, 0, 0, 0, 0, 1250},     // WILD
	{0, 0, 0, 25, 50, 400},    // BONUS
	{0, 0, 0, 100, 200, 1750}, // 7
}

var gAllFreeTimes = [][]int{
	{0, 0, 0, 0, 0, 0},   // 香蕉
	{0, 0, 0, 0, 0, 0},   // 西瓜
	{0, 0, 0, 0, 0, 0},   // 橄榄
	{0, 0, 0, 0, 0, 0},   // 葡萄
	{0, 0, 0, 0, 0, 0},   // 橙子
	{0, 0, 0, 0, 0, 0},   // 铃铛
	{0, 0, 0, 0, 0, 0},   // 樱桃
	{0, 0, 0, 0, 0, 0},   // BAR
	{0, 0, 0, 0, 0, 0},   // WILD
	{0, 0, 0, 5, 10, 20}, // BONUS
	{0, 0, 0, 0, 0, 0},   // 7
}

var defaultSamples = "330-88-43-58-72-38-28-25-15-30-30,280-88-43-58-72-38-28-25-15-30-30,260-88-43-58-72-38-28-25-15-30-40,205-88-43-58-72-38-28-25-5-30-30,66-88-43-58-72-38-28-25-5-30-30"
var gHeaders = []string{"", "", "", "Line777Percent", "Line7777Percent", "Line77777Percent", ""}

// 玩家信息
type shuiguojiUserInfo struct {
	service.UserInfo
	FreeTimes int
	SeatId    int
	Line      int
	Chip      int64
}

type shuiguojiPlayer struct {
	*service.Player

	guid         string
	chip         int64
	line         int
	freeTimes    int
	totalWinGold int64
}

func (ply *shuiguojiPlayer) BeforeLeave() {
	ply.freeTimes = 0
	ply.totalWinGold = 0
}

func (ply *shuiguojiPlayer) Bet(line int, chip int64) {
	room := ply.Room()
	log.Debugf("player %d bet %d line %d", ply.Id, chip, line)

	guid := util.GUID()
	oldFreeTimes := ply.freeTimes
	if ply.freeTimes > 0 {
		ply.freeTimes--
		line, chip, guid = ply.line, ply.chip, ply.guid
	}
	if line <= 0 || line >= len(gAllLines) {
		return
	}

	isValidChip := false
	s, _ := config.String("entertainment", room.GetSubId(), "Chips")
	chips := util.ParseIntSlice(s)
	for _, c := range chips {
		if chip == c {
			isValidChip = true
		}
	}
	gold := int64(line) * chip
	if oldFreeTimes <= 0 && (!isValidChip || gold > ply.Gold) {
		return
	}

	// OK
	ply.chip, ply.line, ply.guid = chip, line, guid
	way := service.ItemWay{Way: "sys.sgj_bet", SubId: room.GetSubId()}.String()
	if oldFreeTimes == 0 {
		ply.AddGold(-gold, util.GUID(), way)
	}

	var cards, clientCards [gPointX][gPointY]int
	var colSamples [gPointX][AllItemNum]int

	ss := defaultSamples
	check := ply.Room().GetInvisiblePrizePool().Check()
	if check == 0 {
		ss1, _ := config.String("entertainment", room.GetSubId(), "CardSamples")
		if len(util.ParseStrings(ss)) == len(util.ParseStrings(ss1)) {
			ss = ss1
		}
	}
	ss = strings.Replace(ss, "-", "|", -1)
	for i, s := range util.ParseStrings(ss) {
		s = strings.Replace(s, "|", ",", -1)
		if i < len(colSamples) {
			sum := 0
			s = strings.Replace(s, "|", "-", -1)
			sample := util.ParseIntSlice(s)
			for _, n := range sample {
				sum += int(n)
			}
			validSum := 1000
			if sum < validSum {
				validSum = sum
			}

			for k := range sample {
				colSamples[i][k] = int(sample[k]) * validSum / sum
			}
		}
	}
	var addFreeTimes int
	var tax, winGold, winPrize, losePrize int64

	cheatCards := make([]int, 0, 16)
	validCards := make([]int, 0, 1000)
	for retry := 100; retry > 0; retry-- {
		cheatCards = cheatCards[:0]
		room.CardSet().Shuffle()
		for x := range cards {
			validCards = validCards[:0]
			for c, n := range colSamples[x] {
				for k := 0; k < n; k++ {
					validCards = append(validCards, c)
				}
			}
			randutil.Shuffle(validCards)
			for k := 0; k < gPointY; k++ {
				cheatCards = append(cheatCards, validCards[k])
			}
		}
		room.CardSet().MoveBack(cheatCards)
		room.CardSet().MoveFront(cheatCards...)
		for x := range cards {
			for y := range cards[x] {
				cards[x][y] = room.CardSet().Deal()
			}
		}

		for k := 0; k <= line; k++ {
			n := 0
			points := gAllLines[k]
			for _, point := range points {
				c := cards[point[0]][point[1]]
				if c == itemWild && c == cards[points[0][0]][points[0][1]] {
					n++
				}
			}
		}
		tax, winGold, winPrize, losePrize, addFreeTimes = 0, 0, 0, 0, 0
		var prizePool = room.GetPrizePool().Add(0)
		for i := 1; i <= line; i++ {
			var c, n int
			for k := 0; k <= len(gAllLines[i]); k++ {
				cell := -1
				if k < len(gAllLines[i]) {
					point := gAllLines[i][k]
					cell = cards[point[0]][point[1]]
				}
				if k == 0 {
					c = cell
				}
				if c == itemWild && !(cell == itemBonus || cell == item7) {
					c = cell
				}
				if cell == itemWild && !(c == itemBonus || c == item7) {
					cell = c
				}
				if c == cell {
					n++
				}
				if c != cell {
					if k == n {
						winGold += int64(gAllMultiples[c][n]) * chip
						if key := gHeaders[n]; c == item7 && len(key) > 0 {
							percent, _ := config.Float("shuiguoji_pool", chip, key)
							winPrize += int64(float64(prizePool) * percent / 100)
						}
					}
					c, n = cell, 1
				}
			}
		}
		// 免费次数
		var bonus int
		for x := range cards {
			old := bonus
			for y := range cards[x] {
				if cards[x][y] == itemBonus {
					bonus++
					break
				}
			}
			if x+1 == len(cards) || old == bonus {
				addFreeTimes += gAllFreeTimes[itemBonus][bonus]
				bonus = 0
			}
		}
		warnLine, _ := config.Int("entertainment", room.GetSubId(), "WarningLine")
		if winGold+winPrize-losePrize > warnLine {
			continue
		}

		simpleInfo := ply.GetSimpleInfo(0)
		user := &PrizePoolUser{
			SimpleUserInfo: *simpleInfo,
			Prize:          winPrize,
		}
		prizePoolRank.update(user)

		ply.totalWinGold += winGold
		if check < 0 && gold > winGold {
			break
		}
		if check > 0 && gold < winGold {
			break
		}
		if check == 0 {
			break
		}
	}
	oldPrizePool := room.GetPrizePool().Add(0)
	if winPrize > losePrize+oldPrizePool {
		winPrize = losePrize + oldPrizePool
	}
	if gold < winGold {
		percent, _ := config.Float("Room", room.GetSubId(), "TaxPercent")
		tax = int64(float64(winGold-gold) * percent / 100)
		percent, _ = config.Float("entertainment", room.GetSubId(), "PrizePoolPercent")
		losePrize = int64(float64(winGold-gold) * percent / 100)
	}
	if winPrize > 0 {
		largs := map[string]any{
			"UId":      ply.Id,
			"WinPrize": winPrize,
			"Rank":     0,
			"Nickname": ply.Nickname,
			"SubId":    room.GetSubId(),
		}
		script.Call("room.lua", "notify_prize_pool", largs)
	}

	ply.freeTimes += addFreeTimes
	room.GetPrizePool().Add(losePrize - winPrize)
	for x := range clientCards {
		for y := range clientCards[x] {
			clientCards[x][y] = cards[x][len(cards[x])-1-y]
		}
	}
	data := map[string]any{
		"Code":      Ok,
		"Chip":      chip,
		"Line":      line,
		"Cards":     clientCards,
		"WinGold":   winGold,
		"FreeTimes": ply.freeTimes,
		"WinPrize":  winPrize,
		"LosePrize": losePrize,
	}
	if oldFreeTimes == 1 {
		data["TotalGold"] = ply.totalWinGold
	}
	ply.RoomObj.BetGold += gold
	ply.RoomObj.WinGold += winGold + winPrize - losePrize - tax
	ply.Room().GetInvisiblePrizePool().Add(winGold + winPrize - gold - tax)
	ply.WriteJSON("Bet", data)
	way = service.ItemWay{Way: "sys.sgj_win", SubId: room.GetSubId()}.String()
	ply.AddGold(winGold+winPrize-losePrize-tax, guid, way)
	if ply.freeTimes == 0 {
		ply.totalWinGold = 0
	}
	// 结算后立即开始下一局
	room.GameOver()
	room.StartGame()
}

func (ply *shuiguojiPlayer) GetUserInfo(otherId int) *shuiguojiUserInfo {
	info := &shuiguojiUserInfo{}
	info.UserInfo = ply.GetInfo(otherId)
	info.SeatId = ply.SeatId
	info.Line = ply.line
	info.Chip = ply.chip
	info.FreeTimes = ply.freeTimes
	return info
}

func (ply *shuiguojiPlayer) Room() *shuiguojiRoom {
	if room := ply.RoomObj.CardRoom(); room != nil {
		return room.(*shuiguojiRoom)
	}
	return nil
}
