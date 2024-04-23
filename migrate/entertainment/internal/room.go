package internal

import (
	"container/list"
	"gofishing-game/internal/cardutils"
	"gofishing-game/internal/gameutils"
	"gofishing-game/migrate/entertainment/utils"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
	"math/rand"
	"sort"
	"time"

	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
	"github.com/guogeer/quasar/randutil"
	"github.com/guogeer/quasar/script"
	"github.com/guogeer/quasar/util"
)

const (
	syncTime = 1500 * time.Millisecond
)

var (
	systemRate              = randutil.FromPercent(50.0) // 默认50%的胜率
	isNextTurnSystemControl = false                      // 下一把系统作弊
)

func RandInArray(a []int) int {
	return randutil.Index(a)
}

func GetPlayer(id int) *entertainmentPlayer {
	if p := service.GetPlayer(id); p != nil {
		return p.GameAction.(*entertainmentPlayer)
	}
	return nil
}

func sortArrayValues(a []int) []int {
	table := make(map[int]bool)
	for _, v := range a {
		table[v] = true
	}

	values := make([]int, 0, 8)
	for v := range table {
		values = append(values, v)
	}
	sort.IntSlice(values).Sort()
	return values
}

type entertainmentDeal struct {
	Type int // 兼容，三个字段同一个含义

	Times        int
	PrizePercent float64 // 奖池瓜分比例
	Prize        int64   // 奖金
	Cards        []int
}

// 押注的游戏，如万人场、水果机，骰子场
type entertainmentGame interface {
	OnEnter(*entertainmentPlayer)

	// 开牌
	StartDealCard()
	Cheat(int) []int
	winPrizePool([]int) float64
}

type entertainmentHelper interface {
	count(cards []int) (int, int)
	Less(fromCards, toCards []int) bool
}

// 升序
type entertainmentAsc struct {
	array  []entertainmentDeal
	helper entertainmentHelper
}

func (asc *entertainmentAsc) Len() int {
	return len(asc.array)
}

func (asc *entertainmentAsc) Swap(i, j int) {
	a := asc.array
	a[i], a[j] = a[j], a[i]
}

func (asc *entertainmentAsc) Less(i, j int) bool {
	a := asc.array
	return asc.helper.Less(a[i].Cards, a[j].Cards)
}

type entertainmentRoom struct {
	*roomutils.Room

	betAreas     []int64 // 各区域押注
	userBetAreas []int64 // 各区域玩家押注
	chips        []int64 // 筹码
	last         [64]int // 历史记录
	lasti        int     // 历史记录索引
	userAreaNum  int     // 玩家押注区域数量

	visiblePrizePool int64 // 明池

	dealer            *entertainmentPlayer
	dealerQueue       *list.List
	deadline          time.Time
	delayCancelDealer bool // 自动下庄

	robSeat           int // 抢座
	entertainmentGame entertainmentGame

	autoTimer         *time.Timer
	deals, cheatDeals []entertainmentDeal
	helper            entertainmentHelper
	dealerLoop        int // 当庄轮数
	cheatWinPercent   float64
	multipleSamples   []int

	invisiblePrizePool utils.InvisiblePrizePool // 暗池
	prizePool          utils.PrizePool          // 奖池
}

func (room *entertainmentRoom) OnEnter(player *service.Player) {
	comer := player.GameAction.(*entertainmentPlayer)

	log.Infof("player %d enter room %d", comer.Id, room.Id)

	minDealerGold, _ := config.Int("entertainment", room.SubId, "MinDealerGold")
	forceCancelDealerGold, _ := config.Int("entertainment", room.SubId, "ForceCancelDealerGold")
	percent, _ := config.Float("entertainment", room.SubId, "AllUserBetPercent")
	loopLimit, _ := config.Int("entertainment", room.SubId, "DealerLoopLimit")
	// 玩家重连
	prize := room.GetPrizePool().Add(0)
	lastPrize := room.GetPrizePool().LastPrize()
	rank := room.GetPrizePool().Rank
	data := map[string]any{
		"status": room.Status,
		"subId":  room.SubId,
		"chips":  room.chips,
		// 奖池
		"prizePool":             prize,
		"lastPrize":             lastPrize,
		"rank":                  rank,
		"countdown":             room.Countdown(),
		"minDealerGold":         minDealerGold,
		"forceCancelDealerGold": forceCancelDealerGold,
		"allUserBetPercent":     percent,
		"currentDealerLoop":     room.dealerLoop,
		"dealerLoopLimit":       loopLimit,

		"robSeat":      room.robSeat,
		"myBetAreas":   comer.betAreas,
		"roomBetAreas": room.betAreas,
		"dealer":       0,
	}

	// 庄家ID
	if room.dealer != nil {
		data["dealer"] = room.dealer.GetUserInfo(comer.Id)
	}
	// 当前排队上庄前10位
	data["DealerQueue"] = comer.dealerQueue()

	// 座位上的玩家
	var seats []*userInfo
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil {
			info := p.GetUserInfo(comer.Id)
			seats = append(seats, info)
		}
	}
	data["seatPlayers"] = seats
	if roomutils.GetRoomObj(comer.Player).GetSeatIndex() == roomutils.NoSeat {
		data["personInfo"] = comer.GetUserInfo(comer.Id)
	}
	comer.SetClientValue("roomInfo", data)

	p := player.GameAction.(*entertainmentPlayer)
	room.entertainmentGame.OnEnter(p)
}

func (room *entertainmentRoom) OnBet(area int, gold int64) {
	room.betAreas[area] += gold
}

func (room *entertainmentRoom) StartGame() {
	room.Room.StartGame()
	if room.userBetAreas == nil {
		room.userBetAreas = make([]int64, len(room.betAreas))
	}
	if room.cheatDeals == nil {
		room.cheatDeals = make([]entertainmentDeal, len(room.deals))
		for i := range room.cheatDeals {
			room.cheatDeals[i].Cards = make([]int, len(room.deals[i].Cards))
		}
	}

	config.Scan("entertainment", room.SubId, "Chips", &room.chips)

	log.Debugf("room %d start game", room.Id)

	// 清理金币不够上庄的玩家
	minDealerGold, _ := config.Int("entertainment", room.SubId, "MinDealerGold")
	for e := room.dealerQueue.Front(); e != nil; {
		p := e.Value.(*entertainmentPlayer)

		e = e.Next()
		if !p.BagObj().IsEnough(gameutils.ItemIdGold, minDealerGold) {
			p.CancelDealer()
		}
	}
	//  推选庄家
	if front := room.dealerQueue.Front(); room.dealer == nil && front != nil {
		room.dealer = front.Value.(*entertainmentPlayer)
		room.dealer.dealerGold = room.dealer.dealerLimitGold
		if room.dealer.dealerGold > room.dealer.BagObj().NumItem(gameutils.ItemIdGold) {
			room.dealer.dealerGold = room.dealer.BagObj().NumItem(gameutils.ItemIdGold)
		}
		if room.dealer.dealerGold == 0 {
			room.dealer.dealerGold = room.dealer.BagObj().NumItem(gameutils.ItemIdGold)
		}
		// 庄家有座位需要先站立
		if roomutils.GetRoomObj(room.dealer.Player).GetSeatIndex() != roomutils.NoSeat {
			roomutils.GetRoomObj(room.dealer.Player).SitUp()
		}
		// room.dealer.RoomObj.IsVisible = true
		room.Broadcast("newDealer", map[string]any{
			"info": room.dealer.GetUserInfo(room.dealer.Id),
		})
		// 2018-01-25 上庄后，队列暂时不清除庄家
		// room.dealerQueue.Remove(front)
		// room.dealer.applyElement = nil
	}

	// 是否有空位
	if room.GetEmptySeat() == roomutils.NoSeat {
		seatId := roomutils.NoSeat
		for i := 0; i < room.NumSeat(); i++ {
			if p := room.GetPlayer(i); p != nil {
				if seatId == roomutils.NoSeat || p.BagObj().NumItem(gameutils.ItemIdGold) < room.GetPlayer(seatId).BagObj().NumItem(gameutils.ItemIdGold) {
					seatId = i
				}
			}
		}
		if seatId != roomutils.NoSeat {
			room.robSeat = seatId
		}
	}

	room.Broadcast("startGame", map[string]any{
		"coutdown": room.Countdown(),
		"robSeat":  room.robSeat,
	})
}

type awardArgs struct {
	SubId      int `json:"subId,omitempty"`
	AreaNum    int `json:"areaNum,omitempty"`
	TotalTimes int `json:"totalTimes,omitempty"`
	Level      int `json:"level,omitempty"`
	Top        int `json:"top,omitempty"`
}

func (room *entertainmentRoom) Award() {
	// 合并机器人的押注日志
	var totalRobotBet int64
	var guid = util.GUID()
	var subId = room.SubId
	var warningLine int64
	config.Scan("entertainment", subId, "WarningLine", &warningLine)
	for _, player := range room.GetAllPlayers() {
		p := player.GameAction.(*entertainmentPlayer)

		totalBet := p.totalBet()
		if p.IsRobot == true {
			totalRobotBet += totalBet
		}
		service.AddSomeItemLog(0, []gameutils.Item{&gameutils.NumericItem{Id: gameutils.ItemIdGold, Num: -totalRobotBet}}, service.GetServerId()+"robot_bet")
	}

	deals := room.deals
	areaNum := len(room.betAreas)
	dealerAreaId := areaNum

	retry, maxRetry := 0, 100
	ipp := room.GetInvisiblePrizePool()
	check := ipp.Check()
	for retry = 0; retry < maxRetry; retry++ {
		room.CardSet().Shuffle()
		for i := range room.deals {
			for k := range room.deals[i].Cards {
				room.deals[i].Cards[k] = 0
				room.cheatDeals[i].Cards[k] = 0
			}
		}
		room.entertainmentGame.StartDealCard()

		var faildeals int
		for i := 0; i < len(deals); i++ {
			cards := deals[i].Cards
			if cards[0] == 0 {
				faildeals++
			}
		}
		if retry < maxRetry/10*7 && retry < maxRetry-1 && faildeals > 0 {
			continue
		}

		// 按照暗池发牌
		totalBet := room.totalUserBet()
		if room.IsSystemDealer() && totalBet != 0 {
			k := dealerAreaId
			gold := room.visiblePrizePool
			if false && gold > 0 {
				k = rand.Intn(len(room.betAreas))
			}
			if isNextTurnSystemControl {
				multipleSamples := room.multipleSamples
				multiples := randutil.Array(multipleSamples)
				cards := room.entertainmentGame.Cheat(multiples)
				if cards != nil {
					copy(room.cheatDeals[k].Cards, cards)
					log.Infof("prize pool control area %d cards %v", k, cards)
				}
			}
			isNextTurnSystemControl = false
		}
		// 部分未发牌的区域，随机发牌
		for i := 0; i < len(deals); i++ {
			cards := deals[i].Cards
			if cards[0] > 0 {
				continue
			}

			for k := 0; k < len(cards); k++ {
				cards[k] = room.CardSet().Deal()
			}
			log.Warn("try rand cards", cards)
		}

		// 系统当庄作弊
		percent := 0.0
		if room.IsSystemDealer() {
			var times, prizeNum int
			var levels []float64
			config.Scan("entertainment", subId, "SystemDealerWinPercent,Lv5WinPercent", &percent, &levels)
			for _, deal := range room.deals {
				pct := room.entertainmentGame.winPrizePool(deal.Cards)
				_, t := room.helper.count(deal.Cards)
				times += t
				if pct > 0 {
					prizeNum++
				}
			}
			args := &awardArgs{
				SubId:      subId,
				TotalTimes: times,
				AreaNum:    areaNum,
			}
			script.Call("room.lua", "fix_room_award", args)
			if args.Level >= len(levels) {
				args.Level = len(levels) - 1
			}
			if args.Level >= 0 && prizeNum == 0 {
				percent = levels[args.Level]
			}
		}
		if !room.IsSystemDealer() {
			config.Scan("config", subId, "betGameUserDealerWinPercent", &percent)
		}
		if percent != 0 {
			asc := &entertainmentAsc{array: room.deals, helper: room.helper}
			utils.HelpDealer(asc, percent+room.cheatWinPercent)
		}

		for i := range room.cheatDeals {
			if room.cheatDeals[i].Cards[0] > 0 && room.helper.Less(room.deals[i].Cards, room.cheatDeals[i].Cards) {
				copy(room.deals[i].Cards, room.cheatDeals[i].Cards)
			}
		}

		// 测试用例
		testSample := cardutils.GetSample()
		if testSample != nil {
			room.CardSet().Shuffle()
			for i := 0; i < len(deals); i++ {
				cards := room.deals[i].Cards
				for k := 0; k < len(cards); k++ {
					cards[k] = room.CardSet().Deal()
				}
			}
		}

		// 发牌后
		prizeAreas := 0
		isRetry := false
		totalPrize := room.GetPrizePool().Add(0)
		minBet, _ := config.Int("entertainment", room.SubId, "MinPrizePoolBet")
		for i := range room.deals {
			pct := room.entertainmentGame.winPrizePool(room.deals[i].Cards)
			if pct > 100.0 {
				pct = 100.0
			}

			typ, multiples := room.helper.count(room.deals[i].Cards)
			room.deals[i].Type = typ
			room.deals[i].Times = multiples
			room.deals[i].PrizePercent = pct
			room.deals[i].Prize = int64(pct / 100.0 * float64(totalPrize))
			if deals[i].Prize > 0 && minBet > 0 && i < areaNum {
				if total := room.totalBet(); total < minBet {
					isRetry = true
				}
			}
			// 系统当庄不开奖池
			if i == areaNum && room.dealer == nil {
				room.deals[i].Prize = 0
			}
			// 没人押注奖池不开奖
			if i < areaNum && room.betAreas[i] == 0 {
				room.deals[i].Prize = 0
			}
			if room.deals[i].Prize > 0 {
				prizeAreas++
			}
		}
		if prizeAreas > 1 {
			isRetry = true
		}

		for i := 0; i < areaNum; i++ {
			if room.helper.Less(deals[dealerAreaId].Cards, deals[i].Cards) == false {
				deals[i].Times = -deals[dealerAreaId].Times
			}
		}

		var systemWinGold int64
		for i := range room.betAreas {
			times := int64(room.deals[i].Times)
			if room.IsSystemDealer() {
				systemWinGold += -times * room.userBetAreas[i]
			} else {
				systemWinGold += times * (room.betAreas[i] - room.userBetAreas[i])
			}
		}
		// 测试用例
		if testSample != nil {
			break
		}

		if isRetry == true {
			continue
		}
		// log.Debug("=========", check, systemWinGold)
		if !ipp.IsValid(-systemWinGold) {
			continue
		}
		// log.Debug("========= ok", check, systemWinGold)
		if check < 0 && systemWinGold >= 0 {
			break
		}
		if check > 0 && systemWinGold <= 0 {
			break
		}
		if check == 0 {
			break
		}
	}
	// 增加牌型日志观察，监控系统
	log.Infof("try %d times, deal cards result:", retry)
	for _, deal := range room.deals {
		log.Info(deal.Cards)
	}

	var lastPrize int64
	for _, deal := range room.deals {
		lastPrize += deal.Prize
	}
	if lastPrize > 0 {
		room.GetPrizePool().ClearRank()
		room.GetPrizePool().SetLastPrize(lastPrize)
	}

	bitMap := 0
	for i := 0; i < areaNum; i++ {
		if deals[i].Times < 0 {
			bitMap |= 1 << uint(i)
		}
	}
	room.last[room.lasti] = bitMap
	room.lasti = (room.lasti + 1) % len(room.last)
	// 结算前，强制同步一次桌面筹码
	room.Sync()

	// 庄家收入
	var dealerWinGold, dealerLoseGold int64
	type Bill struct {
		uid               int
		bet               int64
		total             int64
		prize             int64
		isDealer          bool
		isRobot           bool
		areas, prizeAreas []int64
	}
	// 玩家赢或输
	var bills = make([]*Bill, 0, 64)
	for _, player := range room.GetAllPlayers() {
		p := player.GameAction.(*entertainmentPlayer)

		bill := &Bill{
			uid:        p.Id,
			isRobot:    p.IsRobot,
			areas:      make([]int64, areaNum),
			prizeAreas: make([]int64, areaNum),
		}
		for k, gold := range p.betAreas {
			doubleGold := int64(deals[k].Times) * gold
			bill.total += doubleGold
			bill.bet += gold // 无论输赢，押注的筹码都得算上
			bill.areas[k] = doubleGold
			// 奖池
			if sum := room.betAreas[k]; sum > 0 {
				f := float64(gold) / float64(sum) * float64(deals[k].Prize)
				bill.prizeAreas[k] = int64(f)
				bill.prize += bill.prizeAreas[k]
			}
		}

		total := bill.total
		//  玩家输的金币不能超过所有金币
		if sum := p.BagObj().NumItem(gameutils.ItemIdGold) + p.totalBet(); total+bill.prize+sum < 0 {
			total = -sum - bill.prize
		}

		if total > 0 {
			dealerLoseGold += total
		} else {
			dealerWinGold += -total
		}
		bill.total = total
		bills = append(bills, bill)
	}

	type Area struct {
		SeatId int
		Area   int
		Gold   int64
	}

	var areas []Area
	var totalTax int64
	var betAreas = make([]int64, areaNum)

	// 玩家当庄
	dealerRealLoseGold := dealerLoseGold
	dealerBill := &Bill{
		total:      dealerWinGold - dealerRealLoseGold,
		isDealer:   true,
		areas:      make([]int64, areaNum),
		prizeAreas: make([]int64, areaNum),
	}
	if room.dealer != nil {
		prize := deals[dealerAreaId].Prize
		gold := dealerWinGold + room.dealer.dealerGold + prize
		if dealerLoseGold > gold {
			dealerRealLoseGold = gold
		}
		dealerBill.uid = room.dealer.Id
		dealerBill.isRobot = room.dealer.IsRobot
		dealerBill.prize = prize
		dealerBill.total = dealerWinGold - dealerRealLoseGold
	}
	bills = append(bills, dealerBill)

	dealerWinGold = dealerWinGold - dealerRealLoseGold + dealerBill.prize
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil {
			for k, gold := range p.betAreas {
				if gold != 0 {
					areas = append(areas, Area{SeatId: i, Area: k, Gold: gold})
					betAreas[k] += gold
				}
			}
		}
	}
	// 无座玩家押注
	for k, gold := range betAreas {
		if sub := room.betAreas[k] - gold; sub > 0 {
			areas = append(areas, Area{SeatId: roomutils.NoSeat, Area: k, Gold: sub})
		}
	}

	type seatInfo struct {
		SeatId int
		Gold   int64
		Prize  int64 `json:",omitempty"`
	}

	// 没有座位的玩家金币
	var details []seatInfo
	var noSeatGold, addPrizePool int64
	var taxPercent, prizePoolPercent, robotPercent float64
	config.Scan("room", room.SubId, "taxPercent", &taxPercent)
	config.Scan("config", room.SubId,
		"prizePoolPercent,RTPrizePoolPercent",
		&prizePoolPercent, &robotPercent,
	)
	largs := &awardArgs{
		SubId:   subId,
		AreaNum: areaNum,
	}
	ranklist := service.NewRankList(nil, largs.Top)

	scale := float64(dealerRealLoseGold) / float64(dealerLoseGold)
	for _, bill := range bills {
		uid := bill.uid
		total := bill.total
		prize := bill.prize
		if bill.bet == 0 && !bill.isDealer {
			continue
		}
		// 赢取金币
		if total > 0 && !bill.isDealer {
			total = int64(float64(total) * scale)
		}
		bill.total = total

		percent := prizePoolPercent
		if bill.isRobot == true {
			percent = robotPercent
		}
		var tax1, prize1, tax2, prize2 int64
		if total > 0 {
			tax1 = int64(taxPercent * float64(total) / 100)
			prize1 = int64(percent * float64(total) / 100)
			tax2 = int64(0 * float64(prize) / 100)
			prize2 = int64(0 * float64(prize) / 100)
			totalTax += tax1 + tax2
			addPrizePool += prize1 + prize2
			total -= tax1 + prize1
			prize -= tax2 + prize2
		}

		if base := service.GetPlayer(uid); base != nil {
			p := base.GameAction.(*entertainmentPlayer)
			p.winGold += total + prize + bill.bet
			p.winPrize += prize - prize1
			p.fakeGold += bill.total + bill.prize
			roomutils.GetRoomObj(p.Player).WinGold += total + prize + bill.bet
			room.GetPrizePool().UpdateRank(p.SimpleInfo(0), bill.prize)

			var add int64
			var sub = bill.bet
			for k := range bill.areas {
				if bill.areas[k] < 0 {
					add += bill.areas[k]
				} else {
					sub += -bill.areas[k]
					dealerBill.areas[k] += -bill.areas[k]
				}
			}
			for k := range bill.areas {
				if !bill.isDealer && add > 0 && bill.areas[k] > 0 {
					f := float64(bill.areas[k]) / float64(add) * float64(bill.total+sub)
					bill.areas[k] = int64(f + 0.5)
					dealerBill.areas[k] -= bill.areas[k]
					bill.areas[k] += bill.prizeAreas[k]
				}
			}
			copy(p.fakeAreas, bill.areas)

			if rankgold := bill.total + bill.prize; rankgold > 0 {
				ranklist.Update(base.GetSimpleInfo(0), rankgold)
			}
			if roomutils.GetRoomObj(p.Player).GetSeatIndex() == roomutils.NoSeat {
				noSeatGold += bill.total + bill.prize
			} else {
				seat := seatInfo{
					SeatId: roomutils.GetRoomObj(p.Player).GetSeatIndex(),
					Gold:   bill.total + bill.prize,
					Prize:  bill.prize,
				}
				details = append(details, seat)
			}
		}
	}
	if room.dealer != nil {
		copy(room.dealer.fakeAreas, dealerBill.areas)
	}
	// 暗池控制牌型
	var userWinGold = -room.totalUserBet()
	for _, player := range room.GetAllPlayers() {
		p := player.GameAction.(*entertainmentPlayer)
		if p.IsRobot == false {
			userWinGold += p.winGold - p.winPrize
		}
	}
	ipp.Add(userWinGold) // 暗池
	oldPrize := room.GetPrizePool().Add(0)
	newPrize := room.GetPrizePool().Add(addPrizePool - lastPrize) // 奖池
	if oldPrize > newPrize {
		for rankid, rankuser := range room.GetPrizePool().Rank {
			largs := map[string]any{
				"UId":      rankuser.Id,
				"Nickname": rankuser.Nickname,
				"SubId":    room.SubId,
				"WinPrize": rankuser.Prize,
				"Rank":     rankid,
			}
			script.Call("room.lua", "notify_prize_pool", largs)
		}
	}

	if noSeatGold != 0 {
		details = append(details, seatInfo{SeatId: roomutils.NoSeat, Gold: noSeatGold})
	}

	type PersonInfo struct {
		Gold  int64
		Areas []int64
	}

	script.Call("room.lua", "change_award_cards", subId, deals)
	// 摇骰子
	dice1, dice2 := rand.Intn(6)+1, rand.Intn(6)+1
	for _, player := range room.GetAllPlayers() {
		p := player.GameAction.(*entertainmentPlayer)
		response := map[string]any{
			"tax":        totalTax,
			"deal":       deals,
			"dices":      dice1*10 + dice2,
			"details":    details,
			"betAreas":   areas,
			"personInfo": PersonInfo{Gold: p.fakeGold, Areas: p.fakeAreas},
			"dealer":     PersonInfo{Gold: dealerWinGold, Areas: dealerBill.areas},
			"countdown":  room.Countdown(),
			"top":        ranklist.Top(),
		}
		if newPrize != oldPrize {
			response["PrizePool"] = newPrize
		}
		if lastPrize > 0 {
			response["LastPrize"] = lastPrize
			response["Rank"] = room.GetPrizePool().Rank
		}
		p.WriteJSON("Award", response)
	}

	var totalRobotAward int64
	var awardWay = service.ItemWay{
		Way:   "sys." + service.GetServerId() + "_award",
		SubId: room.SubId,
	}.String()
	var tempAwardWay = service.ItemWay{
		Way:   "sum." + service.GetServerId() + "_award",
		SubId: room.SubId,
	}.String()
	for _, player := range room.GetAllPlayers() {
		p := player.GameAction.(*entertainmentPlayer)
		way := awardWay
		if p.IsRobot == true {
			robot = p
			way = tempAwardWay
			totalRobotAward += p.winGold
		}
		p.AddGold(p.winGold, guid, way)
	}
	if robot != nil {
		robot.AddGoldLog(-totalRobotBet, guid, betWay)
		robot.AddGoldLog(totalRobotAward, guid, awardWay)
	}

	room.GameOver()
}

func (room *entertainmentRoom) GameOver() {
	// 玩家当庄
	if room.dealer != nil {
		room.dealerLoop++
		// 玩家自动下庄或金币不足或当庄轮数超过限制

		var loop int
		var limit int64
		config.Scan("entertainment", room.SubId, "DealerLoopLimit,ForceCancelDealerGold", &loop, &limit)
		if room.dealer.dealerGold < limit || room.delayCancelDealer || (loop > 0 && room.dealerLoop >= loop) {
			room.dealer.CancelDealer()
		}
	}

	for _, player := range room.GetAllPlayers() {
		p := player.GameAction.(*entertainmentPlayer)
		if p.totalBet() == 0 {
			p.continuousBetTimes = 0
		} else {
			p.continuousBetTimes++
		}
	}
	room.Room.GameOver()

	for i := range room.betAreas {
		room.betAreas[i] = 0
	}
	for i := range room.userBetAreas {
		room.userBetAreas[i] = 0
	}

	room.cheatWinPercent = 0
	room.robSeat = roomutils.NoSeat
	room.CardSet().Shuffle()
}

func (room *entertainmentRoom) OnTime() {
	room.Sync()
	util.NewTimer(room.OnTime, syncTime)
}

func (room *entertainmentRoom) Sync() {
	data := map[string]any{
		"onlines":  len(room.GetAllPlayers()),
		"betAreas": room.betAreas[:],
	}
	room.Broadcast("sync", data)
}

func (room *entertainmentRoom) GetLast(n int) []int {
	var last []int
	N := len(room.last)
	for i := (N - n + room.lasti) % N; i != room.lasti; i = (i + 1) % N {
		d := room.last[i]
		if d >= 0 {
			last = append(last, d)
		}
	}
	// 反转
	for i := 0; 2*i < len(last); i++ {
		k := len(last) - 1 - i
		last[i], last[k] = last[k], last[i]
	}
	return last
}

func (room *entertainmentRoom) GetPlayer(seatId int) *entertainmentPlayer {
	if seatId < 0 || seatId >= room.NumSeat() {
		return nil
	}
	if p := room.GetPlayer(seatId); p != nil {
		return p.GameAction.(*entertainmentPlayer)
	}
	return nil
}

func (room *entertainmentRoom) totalBet() int64 {
	var sum int64
	for _, gold := range room.betAreas {
		sum += gold
	}
	return sum
}

func (room *entertainmentRoom) totalUserBet() int64 {
	var sum int64
	for _, gold := range room.userBetAreas {
		sum += gold
	}
	return sum
}

func (room *entertainmentRoom) IsSystemDealer() bool {
	return room.dealer == nil || room.dealer.IsRobot
}

func (room *entertainmentRoom) GetInvisiblePrizePool() *utils.InvisiblePrizePool {
	return &room.invisiblePrizePool
}

func (room *entertainmentRoom) GetPrizePool() *utils.PrizePool {
	return &room.prizePool
}
