package niuniu

import (
	"gofishing-game/internal/errcode"
	"gofishing-game/internal/gameutils"
	"gofishing-game/migrate/internal/cardrule"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
	"math/rand"
	"time"

	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
)

const (
	OptNiuNiuShangZhuang  = "niuNiuShangZhuang"  // 牛牛上庄
	OptGuDingShangZhuang  = "guDingShangZhuang"  // 固定上庄
	OptZiYouShangZhuang   = "ziYouShangZhuang"   // 自由上庄
	OptMingPaiShangZhuang = "mingPaiShangZhuang" // 明牌上庄
	OptTongBiNiuNiu       = "tongBiNiuNiu"       // 通比牛牛

	OptWuXiaoNiu = "wuXiaoNiu" // 五小牛
	OptZhaDanNiu = "zhaDanNiu" // 炸弹牛
	OptWuHuaNiu  = "wuHuaNiu"  // 五花牛

	OptFanBeiGuiZe1 = "fanBeiGuiZe1" // 牛牛X3 牛九X2 牛八X2
	OptFanBeiGuiZe2 = "fanBeiGuiZe2" // 牛牛X4 牛九X3 牛八X2 牛七X2

	OptDiZhu1_2 = "diZhu1_2"
	OptDiZhu2_4 = "diZhu2_4"
	OptDiZhu4_8 = "diZhu4_8"

	OptXianJiaTuiZhu     = "xianJiaTuiZhu"     // 闲家推注
	OptKaiShiJinZhiJinRu = "kaiShiJinZhiJinRu" // 游戏开始后禁止进入

	OptDiZhu1 = "diZhu1" // 底分1
	OptDiZhu2 = "diZhu2" // 低分2
	OptDiZhu4 = "diZhu4" // 低分4

	OptShangZhuangDiFen100 = "shangZhuangDiFen100" // 上庄分数100
	OptShangZhuangDiFen150 = "shangZhuangDiFen150" // 上庄分数150
	OptShangZhuangDiFen200 = "shangZhuangDiFen200" // 上庄分数200

	// 最大抢庄倍数
	OptZuiDaQiangZhuang1 = "zuiDaQiangZhuang1"
	OptZuiDaQiangZhuang2 = "zuiDaQiangZhuang2"
	OptZuiDaQiangZhuang3 = "zuiDaQiangZhuang3"
	OptZuiDaQiangZhuang4 = "zuiDaQiangZhuang4"
	OptZuiDaQiangZhuang5 = "zuiDaQiangZhuang5"

	OptDiZhu1_2_3_4_5 = "diZhu1_2_3_4_5"

	// 2017-10-09 应耒阳地区要求
	OptSiHuaNiu     = "siHuaNiu"     // 四小牛
	OptFanBeiGuiZe3 = "fanBeiGuiZe3" // 四小牛X5 五花牛X6 炸弹牛X7 五小牛X8
)

const (
	RoomStatusBet = 100 + iota
	RoomStatusLook
	RoomStatusRobDealer
	RoomStatusChooseDealer
)

var (
	maxAutoTime = 16 * time.Second
)

type UserDetail struct {
	Uid  int   `json:"uid,omitempty"`
	Gold int64 `json:"gold,omitempty"`
}

type NiuNiuRoom struct {
	*roomutils.Room

	dealer      *NiuNiuPlayer
	nextDealers []*NiuNiuPlayer
	helper      *cardrule.NiuNiuHelper
	isAbleEnd   bool // 房主开始游戏
}

func (room *NiuNiuRoom) OnEnter(player *service.Player) {
	comer := player.GameAction.(*NiuNiuPlayer)
	log.Infof("player %d enter room %d", comer.Id, room.Id)

	// 玩家重连
	data := map[string]any{
		"status":    room.Status,
		"subId":     room.SubId,
		"countdown": room.Countdown(),
	}

	var seats []*NiuNiuPlayerInfo
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil {
			info := p.GetUserInfo(comer.Id == p.Id)
			seats = append(seats, info)
		}
	}
	data["seatPlayers"] = seats
	if room.dealer != nil {
		data["dealerId"] = room.dealer.Id
	}
	if room.CanPlay(OptMingPaiShangZhuang) {
		data["maxRobTimes"] = room.maxRobTimes()
	}

	// 玩家可能没座位
	comer.WriteJSON("getRoomInfo", data)
}

func (room *NiuNiuRoom) Leave(player *service.Player) errcode.Error {
	ply := player.GameAction.(*NiuNiuPlayer)
	log.Debugf("player %d leave room %d", ply.Id, room.Id)
	return nil
}

func (room *NiuNiuRoom) OnLeave(player *service.Player) {
	room.dealer = nil
	room.nextDealers = nil
}

func (room *NiuNiuRoom) OnCreate() {
	helper := room.helper
	if room.CanPlay(OptWuXiaoNiu) {
		helper.SetOption(cardrule.NNWuXiaoNiu)
	}
	if room.CanPlay(OptWuHuaNiu) {
		helper.SetOption(cardrule.NNWuHuaNiu)
	}
	if room.CanPlay(OptZhaDanNiu) {
		helper.SetOption(cardrule.NNZhaDanNiu)
	}
	if room.CanPlay(OptSiHuaNiu) {
		helper.SetOption(cardrule.NNSiHuaNiu)
	}
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
	way := "user." + service.GetServerName()
	unit := room.Unit()

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
	for i := 0; i < room.NumSeat(); i++ {
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
		if room.IsTypeNormal() && !loser.BagObj().IsEnough(gameutils.ItemIdGold, gold) {
			gold = loser.BagObj().NumItem(gameutils.ItemIdGold)
		}

		winner.lastWinGold += gold
		loser.lastWinGold -= gold
		if winner == dealer {
			dealerWinGold += gold
		}
	}

	dealerLostGold := dealerWinGold - dealer.lastWinGold
	if room.IsTypeNormal() && dealer.BagObj().NumItem(gameutils.ItemIdGold)+dealer.lastWinGold < 0 {
		dealer.lastWinGold = -dealer.BagObj().NumItem(gameutils.ItemIdGold)
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
		p.BagObj().Add(gameutils.ItemIdGold, p.lastWinGold, way)
	}

	// 显示其他三家手牌
	details := make([]UserDetail, 0, 8)
	for _, p := range room.readyPlayers() {
		details = append(details, UserDetail{Uid: p.Id, Gold: p.lastWinGold})
	}
	room.Broadcast("award", map[string]any{"Details": details, "countdown": room.Countdown()})
	room.GameOver()
}

func (room *NiuNiuRoom) GameOver() {
	// room.Status = 0

	// 积分场最后一局
	details := make([]UserDetail, 0, 8)
	if room.IsTypeScore() && room.ExistTimes+1 == room.LimitTimes {
		for i := 0; i < room.NumSeat(); i++ {
			if p := room.GetPlayer(i); p != nil {
				details = append(details, UserDetail{Uid: p.Id, Gold: p.BagObj().NumItem(gameutils.ItemIdGold) - roomutils.GetRoomObj(p.Player).OriginGold})
			}
		}
		room.Broadcast("totalAward", map[string]any{"details": details})
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
		if room.ExistTimes >= 3 || (gold > 0 && gold+room.dealer.BagObj().NumItem(gameutils.ItemIdGold) <= 0) {
			room.isAbleEnd = true
			room.dealer.WriteJSON("endGameOrNot", struct{}{})
		}
	}
}

// 游戏中玩家
func (room *NiuNiuRoom) readyPlayers() []*NiuNiuPlayer {
	all := make([]*NiuNiuPlayer, 0, 16)
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil && roomutils.GetRoomObj(p.Player).IsReady() {
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
			seats = append(seats, p.GetSeatIndex())
		}
	}
	data := map[string]any{"uid": room.dealer.Id}
	if len(seats) > 0 {
		data["seats"] = seats
	}
	room.Broadcast("newDealer", data)
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
		"countdown": 0,
	}
	if room.CanPlay(OptMingPaiShangZhuang) {
		// 部分明牌
		type dealCard struct {
			Uid   int   `json:"uid,omitempty"`
			Cards []int `json:"cards,omitempty"`
		}

		// 兼容
		users := make([]dealCard, 0, 8)
		readyPlayers := room.readyPlayers()
		for _, p := range readyPlayers {
			cards := make([]int, 5)
			users = append(users, dealCard{Uid: p.Id, Cards: cards})
		}
		for i, p := range readyPlayers {
			user := &users[i]
			for k := 0; k+1 < len(p.cards); k++ {
				user.Cards[k] = p.cards[k]
			}
			data["parts"] = users
			data["cards"] = user.Cards
			p.WriteJSON("startDealCard", data)

			for k := 0; k+1 < len(p.cards); k++ {
				user.Cards[k] = 0
			}
		}
		return
	}
	room.Broadcast("startDealCard", data)
}

func (room *NiuNiuRoom) AutoChooseTriCards() {
	d, _ := config.Duration("config", "NiuniuLookCards", "Value", maxAutoTime)
	room.SetCountdown(func() {
		for _, p := range room.readyPlayers() {
			var tri [3]int
			if true || room.IsTypeScore() {
				copy(tri[:], p.expectCards[:3])
			}
			p.ChooseTriCards(tri)
		}
	}, d)

	room.Status = RoomStatusLook
	for _, player := range room.GetAllPlayers() {
		p := player.GameAction.(*NiuNiuPlayer)
		data := map[string]any{
			"countdown": room.Countdown(),
		}
		if roomutils.GetRoomObj(p.Player).IsReady() {
			data["weight"] = p.expectWeight
			data["cards"] = p.cards
			data["tricards"] = p.expectCards[:3]
		}
		p.WriteJSON("startLookCard", data)
	}
}

// 选择押注
func (room *NiuNiuRoom) StartBetting() {
	d, _ := config.Duration("config", "NiuniuBetTime", "Value", maxAutoTime)
	room.SetCountdown(func() {
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
	}, d)
	// 没有庄家，如通比牛牛，直接选择摸牌
	room.Status = RoomStatusBet

	for _, player := range room.GetAllPlayers() {
		p := player.GameAction.(*NiuNiuPlayer)
		data := map[string]any{
			"countdown": room.Countdown(),
			"chips":     p.Chips(),
		}

		p.WriteJSON("startBetting", data)
	}
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
		if host := room.GetPlayer(room.HostSeatIndex()); host != nil && host.Room() == room {
			room.nextDealers = []*NiuNiuPlayer{host}
		}
		room.ChooseDealer()
	} else if room.CanPlay(OptMingPaiShangZhuang) {
		d, _ := config.Duration("config", "NiuniuRobDealerTime", "Value", maxAutoTime)
		room.Broadcast("StartRobDealer", map[string]any{
			"countdown": room.Countdown(),
			"times":     room.maxRobTimes(),
		})
		// 开始选择倍数
		room.SetCountdown(func() {
			for _, p := range room.readyPlayers() {
				// 默认不抢庄
				p.DoubleAndRob(0)
			}
		}, d)
		room.Status = RoomStatusRobDealer
		// 等待加倍抢庄
	} else if room.CanPlay(OptTongBiNiuNiu) {
		room.nextDealers = nil
		room.StartBetting()
		// 自由抢庄
	} else if room.CanPlay(OptZiYouShangZhuang) {
		d, _ := config.Duration("config", "NiuniuChooseDealerTime", "Value", maxAutoTime)
		room.Status = RoomStatusChooseDealer
		room.Broadcast("startChooseDealer", map[string]any{
			"countdown": room.Countdown(),
		})
		room.SetCountdown(func() {
			for _, p := range room.readyPlayers() {
				p.ChooseDealer(false)
			}
		}, d)
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
			seats = append(seats, p.GetSeatIndex())
		}
	}

	// 没人抢庄
	if times == 0 {
		times = 1
	}
	seatId := seats[rand.Intn(len(seats))]
	room.dealer = room.GetPlayer(seatId)
	room.dealer.robTimes = times
	room.Broadcast("NewDealer", map[string]any{"Seats": seats, "uid": room.dealer.Id, "Times": times})
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
	room.Broadcast("showAllCard", struct{}{})
	room.Award()
}

func (room *NiuNiuRoom) GetPlayer(seatIndex int) *NiuNiuPlayer {
	if seatIndex < 0 || seatIndex >= room.NumSeat() {
		return nil
	}
	if p := room.FindPlayer(seatIndex); p != nil {
		return p.GameAction.(*NiuNiuPlayer)
	}
	return nil
}
