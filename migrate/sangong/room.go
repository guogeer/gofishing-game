package sangong

import (
	"gofishing-game/internal/errcode"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
	"math/rand"
	"third/cardutil"
	"time"

	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
	"github.com/guogeer/quasar/util"
)

const (
	OptNone              = iota
	OptFangzhudangzhuang // 房主当庄
	OptWuzhuang          // 无庄
	OptZiyouqiangzhuang  // 自由抢庄

	OptChouma1  // 筹码1
	OptChouma2  // 筹码2
	OptChouma3  // 筹码3
	OptChouma5  // 筹码5
	OptChouma8  // 筹码8
	OptChouma10 // 筹码10
	OptChouma20 // 筹码20
)

var (
	maxAutoTime = 16 * time.Second
)

type UserDetail struct {
	UId  int
	Gold int64
	// Cards []int
}

type SangongRoom struct {
	*service.Room

	dealer      *SangongPlayer
	nextDealers []*SangongPlayer
	deadline    time.Time
	helper      *cardutil.SangongHelper

	isAbleStart, isAbleEnd bool // 房主开始游戏
	autoTimer              *util.Timer
}

func (room *SangongRoom) OnEnter(player *service.Player) {
	room.Room.OnEnter(player)

	comer := player.GameAction.(*SangongPlayer)
	log.Infof("player %d enter room %d", comer.Id, room.Id)

	// 自动坐下
	seatId := room.GetEmptySeat()
	if comer.SeatId == roomutils.NoSeat && seatId != roomutils.NoSeat {
		// comer.SitDown()
		comer.RoomObj.SitDown(seatId)

		info := comer.GetUserInfo(false)
		room.Broadcast("SitDown", map[string]any{"Code": Ok, "Info": info}, comer.Id)
	}

	// 玩家重连
	data := map[string]any{
		"Status":    room.Status,
		"SubId":     room.SubId,
		"Countdown": room.Countdown(),
	}

	var seats []*SangongPlayerInfo
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil {
			info := p.GetUserInfo(comer.Id == p.Id)
			seats = append(seats, info)
		}
	}
	data["SeatPlayers"] = seats
	if room.dealer != nil {
		data["DealerId"] = room.dealer.Id
	}

	// 玩家可能没座位
	comer.WriteJSON("GetRoomInfo", data)
}

func (room *SangongRoom) Leave(player *service.Player) errcode.Error {
	ply := player.GameAction.(*SangongPlayer)
	log.Debugf("player %d leave room %d", ply.Id, room.Id)
	return nil
}

func (room *SangongRoom) OnLeave(player *service.Player) {
	room.Room.OnLeave(player)

	p := player.GameAction.(*SangongPlayer)
	if p == room.dealer {
		room.dealer = nil
	}
}

func (room *SangongRoom) OnCreate() {
	room.isAbleStart = false
	room.Room.OnCreate()
}

func (room *SangongRoom) Award() {
	guid := util.GUID()
	way := "user." + service.GetName()
	unit, _ := config.Int("Room", room.SubId, "Unit")

	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil {
			p.winGold = 0
		}
	}

	dealer := room.dealer
	readyPlayers := room.readyPlayers()

	type Relation struct {
		Winner, Loser int
		Gold          int64
	}
	relations := make([]Relation, 0, 16)
	// 无庄，相互比牌
	if room.CanPlay(OptWuzhuang) {
		// 排序，牌最大的玩家排在最前面
		for i := 0; i < len(readyPlayers); i++ {
			for k := 0; k+i+1 < len(readyPlayers); k++ {
				if room.helper.Less(readyPlayers[k].cards, readyPlayers[k+1].cards) {
					readyPlayers[k], readyPlayers[k+1] = readyPlayers[k+1], readyPlayers[k]
				}
			}
		}
		var chips = make([]int, len(readyPlayers))
		for i, p := range readyPlayers {
			chips[i] = p.chip
		}
		for i, winner := range readyPlayers {
			for k := len(readyPlayers) - 1; k > i; k-- {
				var gold int64

				loser := readyPlayers[k]
				if chips[i] <= chips[k] {
					gold = unit * int64(chips[i])
					chips[k] -= chips[i]
					chips[i] = 0
				} else {
					gold = unit * int64(chips[k])
					chips[i] -= chips[k]
					chips[k] = 0
				}
				if gold > 0 {
					winner.winGold += gold
					loser.winGold -= gold
					relations = append(relations, Relation{Winner: winner.SeatId, Loser: loser.SeatId, Gold: gold})
				}
			}
			chips[i] = 0
		}
		for _, p := range readyPlayers {
			p.AddGold(p.winGold, guid, way)
		}
	} else {
		// 默认和庄家比较
		var dealerWinGold int64
		for _, p := range readyPlayers {
			if p == dealer {
				continue
			}

			winner, loser := dealer, p
			if room.helper.Less(winner.cards, loser.cards) {
				winner, loser = loser, winner
			}

			chip := winner.chip
			if loser.chip > 0 {
				chip = loser.chip
			}

			gold := int64(chip) * unit
			if gold > loser.Gold && room.IsTypeScore() == false {
				gold = loser.Gold
			}

			winner.winGold += gold
			loser.winGold -= gold
			if winner == dealer {
				dealerWinGold += gold
			}
		}

		dealerLostGold := dealerWinGold - dealer.winGold
		if dealer.winGold+dealer.Gold < 0 && room.IsTypeScore() == false {
			dealer.winGold = -dealer.Gold
		}

		dealerRealLostGold := dealerWinGold - dealer.winGold
		for _, p := range readyPlayers {
			if p != dealer {
				gold := p.winGold
				if gold > 0 && dealerLostGold > 0 {
					gold = gold * dealerRealLostGold / dealerLostGold
				}
				p.winGold = gold
			}
			p.AddGold(p.winGold, guid, way)
		}
	}

	room.deadline = time.Now().Add(room.RestartTime())
	sec := room.Countdown()
	// 显示其他三家手牌
	details := make([]UserDetail, 0, 8)
	for _, p := range room.readyPlayers() {
		details = append(details, UserDetail{UId: p.Id, Gold: p.winGold})
	}
	room.Broadcast("Award", map[string]any{"Details": details, "Sec": sec, "Relations": relations})

	room.GameOver()
}

func (room *SangongRoom) GameOver() {
	// room.Status = 0

	// 积分场最后一局
	if room.IsTypeScore() && room.ExistTimes+1 == room.LimitTimes {
		details := make([]UserDetail, 0, 8)
		for i := 0; i < room.NumSeat(); i++ {
			if p := room.GetPlayer(i); p != nil {
				details = append(details, UserDetail{UId: p.Id, Gold: p.Gold})
			}
		}
		room.Broadcast("TotalAward", map[string]any{"Details": details})
	}
	room.Room.GameOver()
	util.StopTimer(room.autoTimer)
}

// 游戏中玩家
func (room *SangongRoom) readyPlayers() []*SangongPlayer {
	all := make([]*SangongPlayer, 0, 16)
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil && roomutils.GetRoomObj(p.Player).IsReady() {
			all = append(all, p)
		}
	}
	return all
}

func (room *SangongRoom) ChooseDealer() {
	nextDealers := room.nextDealers

	var seats []int
	if len(nextDealers) > 0 {
		room.dealer = nextDealers[rand.Intn(len(nextDealers))]

		for _, p := range nextDealers {
			seats = append(seats, p.GetSeatIndex())
		}
	}
	if room.dealer != nil {
		data := map[string]any{"UId": room.dealer.Id}
		if len(seats) > 0 {
			data["Seats"] = seats
		}
		room.Broadcast("NewDealer", data)
	}
	room.StartBetting()
}

func (room *SangongRoom) StartDealCard() {
	// 发牌
	room.deadline = time.Now().Add(maxAutoTime)
	sec := room.Countdown()
	for _, p := range room.readyPlayers() {
		for k := 0; k < len(p.cards); k++ {
			p.cards[k] = room.CardSet().Deal()
		}

		p.cardType, _ = room.helper.GetType(p.cards)
	}
	data := map[string]any{
		"Sec": sec,
	}
	room.Broadcast("StartDealCard", data)
}

func (room *SangongRoom) AutoFinish() {
	room.Status = service.RoomStatusLook
	room.deadline = time.Now().Add(maxAutoTime)
	sec := room.Countdown()

	for _, player := range room.AllPlayers {
		p := player.GameAction.(*SangongPlayer)
		data := map[string]any{
			"Sec": sec,
		}
		if roomutils.GetRoomObj(p.Player).IsReady() {
			data["Cards"] = p.cards
		}
		p.WriteJSON("StartLookCard", data)
	}

	room.Timeout(func() {
		for _, p := range room.readyPlayers() {
			p.Finish()
		}
	})
}

// 选择押注
func (room *SangongRoom) StartBetting() {
	// 没有庄家，如通比牛牛，直接选择摸牌
	room.Status = service.RoomStatusBet
	room.deadline = time.Now().Add(maxAutoTime)

	for _, player := range room.AllPlayers {
		p := player.GameAction.(*SangongPlayer)
		data := map[string]any{
			"Sec": room.Countdown(),
		}

		p.WriteJSON("StartBetting", data)
	}

	room.Timeout(func() {
		for _, p := range room.readyPlayers() {
			// 庄家不押注，默认选择1分
			if step := p.Step(); p != room.dealer {
				p.Bet(step)
			}
		}
	})
}

func (room *SangongRoom) OnBet() {
	// 除庄家以外的人都压注了
	for _, p := range room.readyPlayers() {
		if p != room.dealer && p.chip == -1 {
			return
		}
	}

	// OK
	room.AutoFinish()
}

func (room *SangongRoom) StartGame() {
	room.Room.StartGame()

	// 先发牌
	room.StartDealCard()
	// 自由抢庄
	if room.CanPlay(OptZiyouqiangzhuang) {
		room.Status = service.RoomStatusChooseDealer
		room.deadline = time.Now().Add(maxAutoTime)
		room.Broadcast("StartChooseDealer", map[string]any{
			"Sec": room.Countdown()})
		room.Timeout(func() {
			for _, p := range room.readyPlayers() {
				p.ChooseDealer(false)
			}
		})
		// 等待玩家选择抢庄
	} else if room.CanPlay(OptFangzhudangzhuang) {
		// 房主固定为庄家
		if host := room.GetPlayer(room.HostSeatIndex()); host != nil {
			room.nextDealers = []*SangongPlayer{host}
			room.ChooseDealer()
		}
	} else if room.CanPlay(OptWuzhuang) {
		room.StartBetting()
	}
}

func (room *SangongRoom) OnChooseDealer() {
	for _, p := range room.readyPlayers() {
		if p.robOrNot == -1 {
			return
		}
	}

	// OK
	room.nextDealers = nil
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

func (room *SangongRoom) OnFinish() {
	for _, p := range room.readyPlayers() {
		if p.IsDone() == false {
			return
		}
	}

	// 全部亮牌后，直接结算
	room.Broadcast("ShowAllCard", struct{}{})
	room.Award()
}

func (room *SangongRoom) GetPlayer(seatId int) *SangongPlayer {
	if seatId < 0 || seatId >= room.NumSeat() {
		return nil
	}
	if p := room.SeatPlayers[seatId]; p != nil {
		return p.GameAction.(*SangongPlayer)
	}
	return nil
}

func (room *SangongRoom) Timeout(f func()) {
	if room.CanPlay(service.OptAutoPlay) {
		util.StopTimer(room.autoTimer)
		room.autoTimer = util.NewTimer(f, maxAutoTime)
	}
}
