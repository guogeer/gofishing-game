package sangong

import (
	"gofishing-game/games/migrate/internal/cardrule"
	"gofishing-game/internal/errcode"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
	"math/rand"
	"time"

	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
	"github.com/guogeer/quasar/utils"
)

const (
	OptFangzhudangzhuang = "房主当庄" // 房主当庄
	OptWuzhuang          = "无庄"   // 无庄
	OptZiyouqiangzhuang  = "自由抢庄" // 自由抢庄

	OptChouma = "筹码_%d" // 筹码1
)

const (
	RoomStatusLook = 100 + iota
	RoomStatusBet
	RoomStatusChooseDealer
)

var (
	maxAutoTime = 16 * time.Second
)

type UserDetail struct {
	Uid  int   `json:"uid,omitempty"`
	Gold int64 `json:"gold,omitempty"`
}

type SangongRoom struct {
	*roomutils.Room

	dealer      *SangongPlayer
	nextDealers []*SangongPlayer
	deadline    time.Time
	helper      *cardrule.SangongHelper

	isAbleStart, isAbleEnd bool // 房主开始游戏
	autoTimer              *utils.Timer
}

func (room *SangongRoom) OnEnter(player *service.Player) {
	comer := player.GameAction.(*SangongPlayer)
	log.Infof("player %d enter room %d", comer.Id, room.Id)

	// 玩家重连
	data := map[string]any{
		"status": room.Status,
		"subId":  room.SubId,
		"ts":     room.Countdown(),
	}

	var seats []*SangongPlayerInfo
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

	// 玩家可能没座位
	comer.SetClientValue("roomInfo", data)
}

func (room *SangongRoom) Leave(player *service.Player) errcode.Error {
	ply := player.GameAction.(*SangongPlayer)
	log.Debugf("player %d leave room %d", ply.Id, room.Id)
	return nil
}

func (room *SangongRoom) OnLeave(player *service.Player) {
	p := player.GameAction.(*SangongPlayer)
	if p == room.dealer {
		room.dealer = nil
	}
}

func (room *SangongRoom) OnCreate() {
	room.isAbleStart = false
}

func (room *SangongRoom) Award() {
	way := service.GetServerName()
	unit, _ := config.Int("room", room.SubId, "unit")

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
					relations = append(relations, Relation{Winner: winner.GetSeatIndex(), Loser: loser.GetSeatIndex(), Gold: gold})
				}
			}
			chips[i] = 0
		}
		for _, p := range readyPlayers {
			p.AddGold(p.winGold, way, service.WithNoItemLog())
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
			if gold > loser.NumGold() && !room.IsTypeScore() {
				gold = loser.NumGold()
			}

			winner.winGold += gold
			loser.winGold -= gold
			if winner == dealer {
				dealerWinGold += gold
			}
		}

		dealerLostGold := dealerWinGold - dealer.winGold
		if dealer.winGold+dealer.NumGold() < 0 && !room.IsTypeScore() {
			dealer.winGold = -dealer.NumGold()
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
			p.AddGold(p.winGold, way, service.WithNoItemLog())
		}
	}

	room.deadline = time.Now().Add(room.FreeDuration())
	// 显示其他三家手牌
	details := make([]UserDetail, 0, 8)
	for _, p := range room.readyPlayers() {
		details = append(details, UserDetail{Uid: p.Id, Gold: p.winGold})
	}
	room.Broadcast("award", map[string]any{"details": details, "ts": room.Countdown(), "relations": relations})

	room.GameOver()
}

func (room *SangongRoom) GameOver() {
	// room.Status = 0

	// 积分场最后一局
	if room.IsTypeScore() && room.ExistTimes+1 == room.LimitTimes {
		details := make([]UserDetail, 0, 8)
		for i := 0; i < room.NumSeat(); i++ {
			if p := room.GetPlayer(i); p != nil {
				details = append(details, UserDetail{Uid: p.Id, Gold: p.NumGold()})
			}
		}
		room.Broadcast("totalAward", map[string]any{"details": details})
	}
	room.Room.GameOver()
	utils.StopTimer(room.autoTimer)
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
		data := map[string]any{"uid": room.dealer.Id}
		if len(seats) > 0 {
			data["seats"] = seats
		}
		room.Broadcast("newDealer", data)
	}
	room.StartBetting()
}

func (room *SangongRoom) StartDealCard() {
	// 发牌
	room.deadline = time.Now().Add(maxAutoTime)
	for _, p := range room.readyPlayers() {
		for k := 0; k < len(p.cards); k++ {
			p.cards[k] = room.CardSet().Deal()
		}

		p.cardType, _ = room.helper.GetType(p.cards)
	}
	data := map[string]any{
		"ts": room.Countdown(),
	}
	room.Broadcast("startDealCard", data)
}

func (room *SangongRoom) AutoFinish() {
	room.Status = RoomStatusLook
	room.deadline = time.Now().Add(maxAutoTime)
	ts := room.Countdown()

	for _, player := range room.GetAllPlayers() {
		p := player.GameAction.(*SangongPlayer)
		data := map[string]any{
			"ts": ts,
		}
		if roomutils.GetRoomObj(p.Player).IsReady() {
			data["cards"] = p.cards
		}
		p.WriteJSON("startLookCard", data)
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
	room.Status = RoomStatusBet
	room.deadline = time.Now().Add(maxAutoTime)

	for _, player := range room.GetAllPlayers() {
		p := player.GameAction.(*SangongPlayer)
		data := map[string]any{
			"ts": room.Countdown(),
		}

		p.WriteJSON("startBetting", data)
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
		room.Status = RoomStatusChooseDealer
		room.deadline = time.Now().Add(maxAutoTime)
		room.Broadcast("startChooseDealer", map[string]any{
			"ts": room.Countdown()})
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
		if !p.IsDone() {
			return
		}
	}

	// 全部亮牌后，直接结算
	room.Broadcast("showAllCard", struct{}{})
	room.Award()
}

func (room *SangongRoom) GetPlayer(seatIndex int) *SangongPlayer {
	if seatIndex < 0 || seatIndex >= room.NumSeat() {
		return nil
	}
	if p := room.FindPlayer(seatIndex); p != nil {
		return p.GameAction.(*SangongPlayer)
	}
	return nil
}

func (room *SangongRoom) Timeout(f func()) {
	if room.CanPlay(roomutils.OptAutoPlay) {
		utils.StopTimer(room.autoTimer)
		room.autoTimer = utils.NewTimer(f, maxAutoTime)
	}
}
