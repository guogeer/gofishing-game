package niuniu

import (
	"gofishing-game/service"
	"math/rand"
	"third/cardutil"
	. "third/errcode"
	"time"

	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
	"github.com/guogeer/quasar/util"
)

const (
	OptNiuNiuShangZhuang  = 1 + iota // 牛牛上庄
	OptGuDingShangZhuang             // 固定上庄
	OptZiYouShangZhuang              // 自由上庄
	OptMingPaiShangZhuang            // 明牌上庄
	OptTongBiNiuNiu                  // 通比牛牛

	OptWuXiaoNiu // 五小牛
	OptZhaDanNiu // 炸弹牛
	OptWuHuaNiu  // 五花牛

	OptFanBeiGuiZe1 // 牛牛X3 牛九X2 牛八X2
	OptFanBeiGuiZe2 // 牛牛X4 牛九X3 牛八X2 牛七X2

	OptDiZhu1_2
	OptDiZhu2_4
	OptDiZhu4_8

	OptXianJiaTuiZhu     // 闲家推注
	OptKaiShiJinZhiJinRu // 游戏开始后禁止进入

	OptDiZhu1 // 底分1
	OptDiZhu2 // 低分2
	OptDiZhu4 // 低分4

	OptShangZhuangDiFen100 // 上庄分数100
	OptShangZhuangDiFen150 // 上庄分数150
	OptShangZhuangDiFen200 // 上庄分数200

	// 最大抢庄倍数
	OptZuiDaQiangZhuang1
	OptZuiDaQiangZhuang2
	OptZuiDaQiangZhuang3
	OptZuiDaQiangZhuang4
	OptZuiDaQiangZhuang5

	OptDiZhu1_2_3_4_5

	// 2017-10-09 应耒阳地区要求
	OptSiHuaNiu     // 四小牛
	OptFanBeiGuiZe3 // 四小牛X5 五花牛X6 炸弹牛X7 五小牛X8
)

var (
	maxAutoTime = 16 * time.Second
)

type UserDetail struct {
	UId  int
	Gold int64
	// Cards []int
}

type NiuNiuRoom struct {
	*service.Room

	dealer      *NiuNiuPlayer
	nextDealers []*NiuNiuPlayer
	deadline    time.Time
	helper      *cardutil.NiuNiuHelper

	isAbleEnd bool // 房主开始游戏
	autoTimer *util.Timer
}

func (room *NiuNiuRoom) OnEnter(player *service.Player) {
	room.Room.OnEnter(player)

	comer := player.GameAction.(*NiuNiuPlayer)
	log.Infof("player %d enter room %d", comer.Id, room.Id)

	// 自动坐下
	seatId := room.GetEmptySeat()
	if comer.SeatId == service.NoSeat && seatId != service.NoSeat {
		// comer.SitDown()
		comer.RoomObj.SitDown(seatId)

		info := comer.GetUserInfo(false)
		room.Broadcast("SitDown", map[string]any{"Code": Ok, "Info": info}, comer.Id)
	}

	// 玩家重连
	data := map[string]any{
		"Status":    room.Status,
		"SubId":     room.GetSubId(),
		"Countdown": room.GetShowTime(room.deadline),
	}

	var seats []*NiuNiuPlayerInfo
	for i := 0; i < room.SeatNum(); i++ {
		if p := room.GetPlayer(i); p != nil {
			info := p.GetUserInfo(comer.Id == p.Id)
			seats = append(seats, info)
		}
	}
	data["SeatPlayers"] = seats
	if room.dealer != nil {
		data["DealerId"] = room.dealer.Id
	}
	if room.CanPlay(OptMingPaiShangZhuang) {
		data["MaxRobTimes"] = room.maxRobTimes()
	}

	// 玩家可能没座位
	comer.WriteJSON("GetRoomInfo", data)
}

func (room *NiuNiuRoom) Leave(player *service.Player) ErrCode {
	ply := player.GameAction.(*NiuNiuPlayer)
	log.Debugf("player %d leave room %d", ply.Id, room.Id)
	return Ok
}

func (room *NiuNiuRoom) OnLeave(player *service.Player) {
	room.Room.OnLeave(player)

	room.dealer = nil
	room.nextDealers = nil
}

func (room *NiuNiuRoom) OnCreate() {
	helper := room.helper
	if room.CanPlay(OptWuXiaoNiu) {
		helper.SetOption(cardutil.NNWuXiaoNiu)
	}
	if room.CanPlay(OptWuHuaNiu) {
		helper.SetOption(cardutil.NNWuHuaNiu)
	}
	if room.CanPlay(OptZhaDanNiu) {
		helper.SetOption(cardutil.NNZhaDanNiu)
	}
	if room.CanPlay(OptSiHuaNiu) {
		helper.SetOption(cardutil.NNSiHuaNiu)
	}
	room.Room.OnCreate()
}

func (room *NiuNiuRoom) getWeightTimes(weight int) int {
	allTimes := []int{
		1,             // 没牛
		1, 1, 1, 1, 1, // 1~5
		1, 2, 2, 3, 4, // 6~10
		8, // 五小牛
		6, // 炸弹牛
		5, // 五花牛
		4, // 四小牛
	}
	if room.CanPlay(OptFanBeiGuiZe3) {
		allTimes[11] = 8
		allTimes[12] = 7
		allTimes[13] = 6
		allTimes[14] = 5
	} else if room.CanPlay(OptFanBeiGuiZe1) {
		allTimes[7] = 1
		allTimes[8] = 2
		allTimes[9] = 2
		allTimes[10] = 3
	}
	return allTimes[weight]
}

func (room *NiuNiuRoom) maxRobTimes() int {
	userTimes := 0
	if room.CanPlay(OptZuiDaQiangZhuang1) {
		userTimes = 1
	} else if room.CanPlay(OptZuiDaQiangZhuang2) {
		userTimes = 2
	} else if room.CanPlay(OptZuiDaQiangZhuang3) {
		userTimes = 3
	} else if room.CanPlay(OptZuiDaQiangZhuang4) {
		userTimes = 4
	} else if room.CanPlay(OptZuiDaQiangZhuang5) {
		userTimes = 5
	}
	return userTimes
}

func (room *NiuNiuRoom) Award() {
	guid := util.GUID()
	way := "user." + service.GetName()
	unit := room.Unit()

	roomType := room.GetRoomType()
	readyPlayers := room.readyPlayers()
	// 默认和庄家比较
	dealer := room.dealer
	// 没有庄家的时候和最大的比
	if dealer == nil {
		for _, p := range readyPlayers {
			if dealer == nil || dealer.Less(p) {
				dealer = p
			}
		}
	}
	// 清理上局数据
	for i := 0; i < room.SeatNum(); i++ {
		if p := room.GetPlayer(i); p != nil {
			if p.lastWinGold > 0 {
				p.betInAddition = true
			}
			p.lastWinGold = 0
		}
	}

	// 牛牛上庄
	if room.CanPlay(OptNiuNiuShangZhuang) {
		for _, p := range readyPlayers {
			if p.weight == 10 {
				room.nextDealers = append(room.nextDealers, p)
			}
		}
		// 2017-8-28 1.牛牛上庄在几个玩家同时牛牛时，牛牛最大的做庄
		for i := 1; i < len(room.nextDealers); i++ {
			first, second := room.nextDealers[0], room.nextDealers[i]
			if first.Less(second) {
				room.nextDealers[0] = second
			}
		}
	}

	var dealerWinGold int64
	for _, p := range readyPlayers {
		if p == dealer {
			continue
		}

		winner, loser := dealer, p
		if winner.Less(loser) {
			winner, loser = loser, winner
		}

		robTimes := 1
		if dealer != nil && dealer.robTimes > 0 {
			robTimes = dealer.robTimes
		}

		betTimes := winner.betTimes
		if loser.betTimes > 0 {
			betTimes = loser.betTimes
		}

		userTimes := 1
		if room.CanPlay(OptDiZhu1) {
			userTimes = 1
		} else if room.CanPlay(OptDiZhu2) {
			userTimes = 2
		} else if room.CanPlay(OptDiZhu4) {
			userTimes = 4
		}

		if robTimes > 0 {
			userTimes *= robTimes
		}
		if betTimes > 0 {
			userTimes *= betTimes
		}

		weightTimes := room.getWeightTimes(winner.weight)
		gold := int64(weightTimes) * int64(userTimes) * unit
		if gold > loser.AliasGold() && roomType != service.RoomTypeScore {
			gold = loser.AliasGold()
		}

		winner.lastWinGold += gold
		loser.lastWinGold -= gold
		if winner == dealer {
			dealerWinGold += gold
		}
	}

	dealerLostGold := dealerWinGold - dealer.lastWinGold
	if dealer.lastWinGold+dealer.AliasGold() < 0 && roomType != service.RoomTypeScore {
		dealer.lastWinGold = -dealer.AliasGold()
	}

	dealerRealLostGold := dealerWinGold - dealer.lastWinGold
	for _, p := range readyPlayers {
		if p != dealer {
			gold := p.lastWinGold
			if gold > 0 && dealerLostGold > 0 {
				gold = gold * dealerRealLostGold / dealerLostGold
			}
			p.lastWinGold = gold
		}
		p.AddAliasGold(p.lastWinGold, guid, way)
	}

	room.deadline = time.Now().Add(room.RestartTime())
	sec := room.GetShowTime(room.deadline)
	// 显示其他三家手牌
	details := make([]UserDetail, 0, 8)
	for _, p := range room.readyPlayers() {
		details = append(details, UserDetail{UId: p.Id, Gold: p.lastWinGold})
	}
	room.Broadcast("Award", map[string]any{"Details": details, "Times": sec, "Sec": sec})

	// room.ExistTimes++
	room.GameOver()
}

func (room *NiuNiuRoom) GameOver() {
	// room.Status = service.RoomStatusFree

	// 积分场最后一局
	details := make([]UserDetail, 0, 8)
	if room.IsUserCreate() && room.ExistTimes+1 == room.LimitTimes {
		for i := 0; i < room.SeatNum(); i++ {
			if p := room.GetPlayer(i); p != nil {
				details = append(details, UserDetail{UId: p.Id, Gold: p.Gold - p.OriginGold})
			}
		}
		room.Broadcast("TotalAward", map[string]any{"Details": details})
	}
	room.Room.GameOver()

	// 固定当庄，超过三局、输完都可以提前下庄
	if room.CanPlay(OptGuDingShangZhuang) {
		var gold int64
		if room.CanPlay(OptShangZhuangDiFen100) {
			gold = 100
		} else if room.CanPlay(OptShangZhuangDiFen150) {
			gold = 150
		} else if room.CanPlay(OptShangZhuangDiFen200) {
			gold = 200
		}
		if room.ExistTimes >= 3 || (gold > 0 && gold+room.dealer.AliasGold() <= 0) {
			room.isAbleEnd = true
			room.dealer.WriteJSON("EndGameOrNot", struct{}{})
		}
	}

	util.StopTimer(room.autoTimer)
}

// 游戏中玩家
func (room *NiuNiuRoom) readyPlayers() []*NiuNiuPlayer {
	all := make([]*NiuNiuPlayer, 0, 16)
	for i := 0; i < room.SeatNum(); i++ {
		if p := room.GetPlayer(i); p != nil && p.RoomObj.IsReady() {
			all = append(all, p)
		}
	}
	return all
}

func (room *NiuNiuRoom) ChooseDealer() {
	choices := room.nextDealers
	if room.dealer == nil && len(choices) == 0 {
		choices = room.readyPlayers()
	}

	var seats []int
	if len(choices) > 0 {
		room.dealer = choices[rand.Intn(len(choices))]

		for _, p := range choices {
			seats = append(seats, p.SeatId)
		}
	}
	data := map[string]any{"UId": room.dealer.Id}
	if len(seats) > 0 {
		data["Seats"] = seats
	}
	room.Broadcast("NewDealer", data)
	room.nextDealers = nil
	room.StartBetting()
}

func (room *NiuNiuRoom) StartDealCard() {
	// 发牌
	for _, p := range room.readyPlayers() {
		for k := 0; k < len(p.cards); k++ {
			p.cards[k] = room.CardSet().Deal()
		}

		p.expectWeight, p.expectCards = room.helper.Weight(p.cards)
	}
	data := map[string]any{
		"Sec": 0,
	}
	if room.CanPlay(OptMingPaiShangZhuang) {
		// 部分明牌
		type dealCard struct {
			UId   int
			Cards []int
		}

		// 兼容
		users := make([]dealCard, 0, 8)
		readyPlayers := room.readyPlayers()
		for _, p := range readyPlayers {
			cards := make([]int, 5)
			users = append(users, dealCard{UId: p.Id, Cards: cards})
		}
		for i, p := range readyPlayers {
			user := &users[i]
			for k := 0; k+1 < len(p.cards); k++ {
				user.Cards[k] = p.cards[k]
			}
			data["Parts"] = users
			data["Cards"] = user.Cards
			p.WriteJSON("StartDealCard", data)

			for k := 0; k+1 < len(p.cards); k++ {
				user.Cards[k] = 0
			}
		}
		return
	}
	room.Broadcast("StartDealCard", data)
}

func (room *NiuNiuRoom) AutoChooseTriCards() {
	d := maxAutoTime
	if t, ok := config.Duration("config", "NiuniuLookCards", "Value"); ok {
		d = t
	}
	room.Status = service.RoomStatusLook
	room.deadline = time.Now().Add(d)

	sec := room.GetShowTime(room.deadline)
	for _, player := range room.AllPlayers {
		p := player.GameAction.(*NiuNiuPlayer)
		data := map[string]any{
			"Sec": sec,
		}
		if p.RoomObj.IsReady() {
			data["Weight"] = p.expectWeight
			data["Cards"] = p.cards
			data["Tricards"] = p.expectCards[:3]
		}
		p.WriteJSON("StartLookCard", data)
	}

	room.Timeout(func() {
		for _, p := range room.readyPlayers() {
			var tri [3]int
			if true || room.IsTypeScore() {
				copy(tri[:], p.expectCards[:3])
			}
			p.ChooseTriCards(tri)
		}
	})
}

// 选择押注
func (room *NiuNiuRoom) StartBetting() {
	d := maxAutoTime
	if t, ok := config.Duration("config", "NiuniuBetTime", "Value"); ok {
		d = t
	}
	// 没有庄家，如通比牛牛，直接选择摸牌
	room.Status = service.RoomStatusBet
	room.deadline = time.Now().Add(d)

	for _, player := range room.AllPlayers {
		p := player.GameAction.(*NiuNiuPlayer)
		data := map[string]any{
			"Sec":   room.GetShowTime(room.deadline),
			"Chips": p.Chips(),
		}

		p.WriteJSON("StartBetting", data)
	}

	room.Timeout(func() {
		for _, p := range room.readyPlayers() {
			// 庄家不押注，默认选择1分
			if chips := p.Chips(); p != room.dealer {
				chip := chips[0]
				if n := p.autoTimes; n > 0 {
					chip = n
				}
				p.Bet(chip)
			}
		}
	})
}

func (room *NiuNiuRoom) OnBet() {
	// 除庄家以外的人都压注了
	for _, p := range room.readyPlayers() {
		if p != room.dealer && p.betTimes == -1 {
			return
		}
	}

	// OK
	room.AutoChooseTriCards()
}

func (room *NiuNiuRoom) StartGame() {
	room.Room.StartGame()

	// 先发牌
	room.StartDealCard()

	if room.CanPlay(OptNiuNiuShangZhuang) {
		room.ChooseDealer()
	} else if room.CanPlay(OptGuDingShangZhuang) {
		// 房主固定为庄家
		if host := GetPlayer(room.HostId); host != nil && host.Room() == room {
			room.nextDealers = []*NiuNiuPlayer{host}
		}
		room.ChooseDealer()
	} else if room.CanPlay(OptMingPaiShangZhuang) {
		d := maxAutoTime
		if t, ok := config.Duration("config", "NiuniuRobDealerTime", "Value"); ok {
			d = t
		}
		room.Status = service.RoomStatusRobDealer
		room.deadline = time.Now().Add(d)
		sec := room.GetShowTime(room.deadline)
		room.Broadcast("StartRobDealer", map[string]any{
			"Sec":   sec,
			"Times": room.maxRobTimes(),
		})
		// 开始选择倍数
		room.Timeout(func() {
			for _, p := range room.readyPlayers() {
				// 默认不抢庄
				p.DoubleAndRob(0)
			}
		})
		// 等待加倍抢庄
	} else if room.CanPlay(OptTongBiNiuNiu) {
		room.nextDealers = nil
		room.StartBetting()
		// 自由抢庄
	} else if room.CanPlay(OptZiYouShangZhuang) {
		d := maxAutoTime
		if t, ok := config.Duration("config", "NiuniuChooseDealerTime", "Value"); ok {
			d = t
		}
		room.Status = service.RoomStatusChooseDealer
		room.deadline = time.Now().Add(d)
		room.Broadcast("StartChooseDealer", map[string]any{
			"Sec": room.GetShowTime(room.deadline)})
		room.Timeout(func() {
			for _, p := range room.readyPlayers() {
				p.ChooseDealer(false)
			}
		})
		// 等待玩家选择抢庄
	}
}

func (room *NiuNiuRoom) OnChooseDealer() {
	for _, p := range room.readyPlayers() {
		if p.robOrNot == -1 {
			return
		}
	}

	// OK
	for _, p := range room.readyPlayers() {
		if p.robOrNot == 1 {
			room.nextDealers = append(room.nextDealers, p)
		}
	}
	// 没人抢庄
	if len(room.nextDealers) == 0 {
		room.nextDealers = room.readyPlayers()
	}

	// room.StartDealCard()
	room.ChooseDealer()
}

func (room *NiuNiuRoom) OnDoubleAndRob() {
	for _, p := range room.readyPlayers() {
		if p.robTimes == -1 {
			return
		}
	}

	// OK
	times := 0
	for _, p := range room.readyPlayers() {
		if times < p.robTimes {
			times = p.robTimes
		}
	}

	var seats []int
	for _, p := range room.readyPlayers() {
		if times == p.robTimes {
			seats = append(seats, p.SeatId)
		}
	}

	// 没人抢庄
	if times == 0 {
		times = 1
	}
	seatId := seats[rand.Intn(len(seats))]
	room.dealer = room.GetPlayer(seatId)
	room.dealer.robTimes = times
	room.Broadcast("NewDealer", map[string]any{"Seats": seats, "UId": room.dealer.Id, "Times": times})
	// room.AutoShow()
	room.StartBetting()
}

func (room *NiuNiuRoom) OnChooseTriCards() {
	for _, p := range room.readyPlayers() {
		if p.weight == -1 {
			return
		}
	}

	// 全部亮牌后，直接结算
	room.Broadcast("ShowAllCard", struct{}{})
	room.Award()
}

func (room *NiuNiuRoom) GetPlayer(seatId int) *NiuNiuPlayer {
	if seatId < 0 || seatId >= room.SeatNum() {
		return nil
	}
	if p := room.SeatPlayers[seatId]; p != nil {
		return p.GameAction.(*NiuNiuPlayer)
	}
	return nil
}

func (room *NiuNiuRoom) Timeout(f func()) {
	d := room.deadline.Sub(time.Now())
	if room.CanPlay(service.OptAutoPlay) {
		util.StopTimer(room.autoTimer)
		room.autoTimer = util.NewTimer(f, d)
	}
}
