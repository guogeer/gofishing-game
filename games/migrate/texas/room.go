package texas

import (
	"gofishing-game/games/migrate/internal/cardrule"
	"gofishing-game/internal/errcode"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
	"time"

	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
	"github.com/guogeer/quasar/utils"
)

var (
	systemFailTime = 10 * time.Second
	systemAutoTime = 500 * time.Millisecond
)

type Relation struct {
	PotId     int
	SeatIndex int
	Gold      int64
}

type TexasRoomInfo struct {
	Id, SubId   int
	FrontBlind  int64
	SmallBlind  int64
	BigBlind    int64
	MinBankroll int64
	MaxBankroll int64
	ActiveUsers int
}

type TexasRoom struct {
	*roomutils.Room

	activePlayer *TexasPlayer
	winner       *TexasPlayer

	deadline time.Time
	helper   *cardrule.TexasHelper

	potId int

	allBlind [64]int64 // 本轮的押注
	allPot   [64]int64 // 奖池

	cards []int // 公共牌
	raise int64 // 加注

	// 升盲
	delayAddBlind bool
	blindLoop     int

	frontBlind, bigBlind, smallBlind int64
	nextSmallBlind, nextBigBlind     int64

	tempDealerSeat, dealerSeat, bigBlindSeat, smallBlindSeat int // 注意，这里是大盲、小盲位置

	continuousLoop int // 连续玩的局数
	tournament     *TournamentCopy
}

func (room *TexasRoom) OnEnter(player *service.Player) {
	comer := player.GameAction.(*TexasPlayer)
	log.Infof("player %d enter room %d", comer.Id, room.Id)

	// 玩家重连
	data := map[string]any{
		"status": room.Status,
		"subId":  room.SubId,
		"ts":     room.Countdown(),
	}
	if room.dealerSeat >= 0 {
		data["dealerSeat"] = room.dealerSeat
	}
	if room.smallBlindSeat >= 0 {
		data["smallBlindSeat"] = room.smallBlindSeat
	}
	if room.bigBlindSeat >= 0 {
		data["bigBlindSeat"] = room.bigBlindSeat
	}
	if room.smallBlind > 0 {
		data["smallBlind"] = room.smallBlind
	}
	if room.bigBlind >= 0 {
		data["bigBlind"] = room.bigBlind
	}
	// 断线重连增加奖池
	if room.Status == roomutils.RoomStatusPlaying {
		data["pots"] = room.allPot[:room.potId+1]
	}

	var seats []*TexasUserInfo
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil {
			info := p.GetUserInfo(comer.Id == p.Id)
			seats = append(seats, info)
		}
	}
	data["seatPlayers"] = seats
	if comer.GetSeatIndex() == roomutils.NoSeat {
		data["personInfo"] = comer.GetUserInfo(true)
	}

	// 玩家可能没座位
	comer.SetClientValue("roomInfo", data)
	if room.Status == roomutils.RoomStatusPlaying {
		comer.OnTurn()
	}
}

func (room *TexasRoom) Leave(player *service.Player) errcode.Error {
	ply := player.GameAction.(*TexasPlayer)
	log.Debugf("player %d leave room %d", ply.Id, room.Id)
	return nil
}

func (room *TexasRoom) OnLeave(player *service.Player) {
	counter := 0
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil {
			counter++
		}
	}
	if counter < 2 {
		room.dealerSeat = -1
		room.smallBlindSeat = -1
		room.bigBlindSeat = -1
	}

	counter = 0
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil && roomutils.GetRoomObj(p.Player).ContinuousPlayTimes > 0 {
			counter++
		}
	}
	if counter < 2 {
		room.continuousLoop = 0
	}
}

func (room *TexasRoom) OnCreate() {
	room.tempDealerSeat = -1
	// room.Room.OnCreate()

	if !room.IsTypeTournament() {
		subId := room.SubId
		room.smallBlind, _ = config.Int("texasroom", subId, "smallBlind")
		room.bigBlind, _ = config.Int("texasroom", subId, "bigBlind")
	}
}

func (room *TexasRoom) Award() {
	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		if p != nil && p.IsPlaying() {
			p.winGold = 0
		}
	}
	// 自动亮牌
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil {
			p.AutoPlay()
		}
	}

	helper := room.helper
	room.deadline = time.Now().Add(room.FreeDuration())
	sec := room.Countdown()

	relations := make([]Relation, 0, 16)
	for potId := 0; potId <= room.potId; potId++ {
		var winners = make([]*TexasPlayer, 0, 1)
		if p := room.winner; p != nil {
			winners = append(winners, p)
		} else {
			var maxMatch []int
			for k := 0; k < room.NumSeat(); k++ {
				p := room.GetPlayer(k)
				if p != nil && p.IsPlaying() && p.potId >= potId {
					_, match := p.match()
					if maxMatch == nil || helper.Less(maxMatch, match) {
						maxMatch = match
					}
				}
			}

			for k := 0; k < room.NumSeat(); k++ {
				p := room.GetPlayer(k)
				if p != nil && p.IsPlaying() && p.potId >= potId {
					_, match := p.match()
					if helper.Equal(maxMatch, match) {
						winners = append(winners, p)
					}
				}
			}
		}
		gold := room.allPot[potId] / int64(len(winners))
		for _, p := range winners {
			p.winGold += gold
			relations = append(relations, Relation{SeatIndex: p.GetSeatIndex(), PotId: potId, Gold: gold})
		}
	}
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil {
			p.AddBankroll(p.winGold)
		}
	}
	// 显示其他三家手牌
	type UserDetail struct {
		UId      int
		Cards    [2]int
		Match    []int
		CardType int
		Gold     int64
	}
	users := make([]UserDetail, 0, 16)
	folders := make([]UserDetail, 0, 16)
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil {
			typ, match := p.match()
			detail := UserDetail{
				UId:      p.Id,
				Gold:     p.winGold,
				Cards:    p.cards,
				Match:    match,
				CardType: typ,
			}
			if p.IsPlaying() {
				users = append(users, detail)
			}
			if p.action == ActionFold && p.isShow {
				folders = append(folders, detail)
			}
		}
	}
	room.Broadcast("award", map[string]any{
		"sec":       sec,
		"users":     users,
		"folders":   folders,
		"relations": relations,
	})

	room.continuousLoop++
	room.GameOver()
}

func (room *TexasRoom) GameOver() {
	// 积分场最后一局
	if room.IsTypeScore() && room.ExistTimes+1 == room.LimitTimes {
		room.Broadcast("totalAward", struct{}{})
	}
	room.Room.GameOver()

	subId := room.SubId
	minBankroll, _ := config.Int("texasroom", subId, "minBankroll")
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil && p.bankroll == 0 && p.NumGold() < minBankroll {
			p.SitUp()
		}
	}
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil {
			p.initBankroll()
		}
	}

	room.winner = nil
	for i := 0; i < len(room.allBlind); i++ {
		room.allBlind[i] = 0
	}
	for i := 0; i < len(room.allPot); i++ {
		room.allPot[i] = 0
	}
	room.potId = 0
	room.cards = room.cards[:0]
	room.activePlayer = nil

	if room.IsTypeTournament() {
		cp := room.Tournament()
		users := make([]*TournamentUser, 0, 16)
		for i := 0; i < room.NumSeat(); i++ {
			if p := room.GetPlayer(i); p != nil && roomutils.GetRoomObj(p.Player).IsReady() {
				user := cp.Users[p.Id]
				user.Gold = p.bankroll
				users = append(users, user)
			}
		}
		cp.UpdateRank(users)
		cp.MergeRoom(room.Room)

		if room.delayAddBlind {
			room.smallBlind = room.nextSmallBlind
			room.bigBlind = room.nextBigBlind
			room.delayAddBlind = false
			room.OnAddBlind()
		}

		for i := 0; i < room.NumSeat(); i++ {
			if p := room.GetPlayer(i); p != nil && p.bankroll == 0 {
				if cp.IsAbleRebuy(room.blindLoop) || cp.IsAbleAddon(room.blindLoop) {
					p.failTimer = utils.NewTimer(p.Fail, systemFailTime)
				} else {
					p.Fail()
				}
			}
		}
	}
}

func (room *TexasRoom) StartGame() {
	room.Room.StartGame()

	// choose dealer
	room.dealerSeat = room.NextSeat(room.tempDealerSeat)
	if host := room.GetPlayer(room.HostSeatIndex()); host != nil && room.tempDealerSeat == -1 {
		room.dealerSeat = host.GetSeatIndex()
	}
	// save dealer
	room.tempDealerSeat = room.dealerSeat
	// small blind
	room.smallBlindSeat = room.NextSeat(room.dealerSeat)
	// big blind
	room.bigBlindSeat = room.NextSeat(room.smallBlindSeat)

	// only two players, no dealer
	var counter int
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil && p.IsPlaying() {
			counter++
		}
	}
	if counter == 2 {
		room.dealerSeat = room.smallBlindSeat
	}

	sb := room.GetPlayer(room.smallBlindSeat)
	bb := room.GetPlayer(room.bigBlindSeat)

	sb.totalBlind = room.smallBlind
	// 两个人时默认下大盲注
	if counter == 2 {
		sb.totalBlind = room.bigBlind
	}
	bb.totalBlind = room.bigBlind
	if sb.totalBlind > sb.bankroll {
		sb.totalBlind = sb.bankroll
	}
	if bb.totalBlind > bb.bankroll {
		bb.totalBlind = bb.bankroll
	}
	sb.AddBankroll(-sb.totalBlind)
	bb.AddBankroll(-bb.totalBlind)

	room.raise = 2 * room.bigBlind
	room.allBlind[sb.GetSeatIndex()] = sb.totalBlind
	room.allBlind[bb.GetSeatIndex()] = bb.totalBlind

	counter = 0 // 统计老玩家数量
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil && p.IsPlaying() && roomutils.GetRoomObj(p.Player).ContinuousPlayTimes > 0 {
			counter++
		}
	}
	subId := room.SubId
	minReadyNum, _ := config.Int("texasroom", subId, "bigBlind")
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil && p.IsPlaying() {
			var gold int64
			// 房间老玩家人数不少于开局人数，新玩家自动压大盲注
			t := roomutils.GetRoomObj(p.Player).ContinuousPlayTimes
			if counter >= int(minReadyNum) && t == 0 && room.continuousLoop > 0 {
				gold = room.bigBlind
				if p == bb {
					gold = 0
				}
				if p == sb {
					gold = room.bigBlind - room.smallBlind
				}
			}
			if gold >= p.bankroll {
				gold = p.bankroll
				p.action = ActionAllIn
			}

			p.totalBlind += gold
			p.AddBankroll(-gold)
			room.allBlind[p.GetSeatIndex()] += gold
		}
	}

	// start deal card
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil && p.IsPlaying() {
			for k := 0; k < len(p.cards); k++ {
				p.cards[k] = room.CardSet().Deal()
			}
		}
	}

	for _, player := range room.GetAllPlayers() {
		data := map[string]any{
			"smallBlindSeat": room.smallBlindSeat,
			"bigBlindSeat":   room.bigBlindSeat,
			"allBlind":       room.allBlind[:room.NumSeat()],
		}
		p := player.GameAction.(*TexasPlayer)
		if p.IsPlaying() {
			data["cards"] = p.cards
		}
		if room.dealerSeat >= 0 {
			data["dealerSeat"] = room.dealerSeat
		}
		p.WriteJSON("startPlaying", data)
	}

	if seatId := room.NextSeat(room.bigBlindSeat); seatId == roomutils.NoSeat {
		room.NewRound()
	} else {
		room.activePlayer = room.GetPlayer(seatId)
		room.Turn()
	}
}

func (room *TexasRoom) GetPlayer(seatIndex int) *TexasPlayer {
	if seatIndex < 0 || seatIndex >= room.NumSeat() {
		return nil
	}
	if p := room.FindPlayer(seatIndex); p != nil {
		return p.GameAction.(*TexasPlayer)
	}
	return nil
}

func (room *TexasRoom) OnTakeAction() {
	// first step
	// others fold except one
	var counter int
	var winner *TexasPlayer
	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		if p != nil && p.IsPlaying() {
			counter++
			winner = p
		}
	}

	if counter < 2 {
		room.winner = winner
	}
	log.Debug("take action", counter, winner.Id)

	act := room.activePlayer
	room.activePlayer = nil
	act.AutoPlay() // 亮牌

	nextId := room.NextSeat(act.GetSeatIndex())
	next := room.GetPlayer(nextId)
	// 加注
	raise := 2 * act.totalBlind
	if act.action == ActionRaise {
		room.raise = raise
	}
	// 全压
	if act.action == ActionAllIn && room.raise < raise {
		room.raise = raise
	}

	round := true
	maxBlind := maxInArray(room.allBlind[:])
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil && p.IsPlaying() {
			if p.action == ActionNone || (p.action != ActionAllIn && p.totalBlind < maxBlind) {
				round = false
			}
		}
	}
	if room.winner != nil || next == nil || next == act || round {
		room.NewRound()
	} else {
		room.activePlayer = next
		room.Turn()
	}
}

// new round
func (room *TexasRoom) NewRound() {
	room.raise = 2 * room.bigBlind

	dealNum := 1
	cardNum := len(room.cards)
	if cardNum == 0 {
		dealNum = 3
	}

	activeUsers := 0
	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		if p != nil && p.IsPlaying() && p.action != ActionAllIn {
			activeUsers++
		}
	}

	activeSeat := room.NextSeat(room.tempDealerSeat)
	if activeUsers < 2 || activeSeat == roomutils.NoSeat {
		dealNum = cap(room.cards) - len(room.cards)
	}
	if cardNum >= cap(room.cards) || room.winner != nil {
		dealNum = 0
	}
	for i := 0; i < dealNum; i++ {
		c := room.CardSet().Deal()
		room.cards = append(room.cards, c)
	}

	relations := make([]Relation, 0, 32)
	// 弃牌玩家
	for k := 0; k < room.NumSeat(); k++ {
		bet := room.allBlind[k]
		if p := room.GetPlayer(k); p != nil && bet > 0 && p.action == ActionFold {
			room.allBlind[k] = 0
			room.allPot[room.potId] += bet

			p.potId = room.potId
			relations = append(relations, Relation{PotId: room.potId, SeatIndex: k, Gold: bet})
		}
	}

	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil {
			p.totalBlind = 0
			p.action = ActionNone
		}
	}

	tempNum := len(relations)
	for {
		minSeatId := roomutils.NoSeat
		for k := 0; k < room.NumSeat(); k++ {
			if room.allBlind[k] > 0 {
				if minSeatId == roomutils.NoSeat || room.allBlind[minSeatId] > room.allBlind[k] {
					minSeatId = k
				}
			}
		}
		if minSeatId == roomutils.NoSeat {
			break
		}
		if len(relations) > tempNum {
			room.potId++
		}
		minBlind := room.allBlind[minSeatId]
		for k := 0; k < room.NumSeat(); k++ {
			if room.allBlind[k] > 0 {
				room.allBlind[k] -= minBlind
				room.allPot[room.potId] += minBlind
				if p := room.GetPlayer(k); p != nil {
					p.potId = room.potId
				}
				relations = append(relations, Relation{PotId: room.potId, SeatIndex: k, Gold: minBlind})
			}
		}
	}

	log.Debug("new round", dealNum, room.allPot[:room.potId+1])
	for _, player := range room.GetAllPlayers() {
		p := player.GameAction.(*TexasPlayer)
		data := map[string]any{
			"cards":     room.cards,
			"pots":      room.allPot[:room.potId+1],
			"relations": relations,
		}

		if p.IsPlaying() {
			typ, match := p.match()
			data["match"] = match
			data["cardType"] = typ
		}
		p.WriteJSON("newRound", data)
	}
	if room.winner != nil || cardNum == cap(room.cards) || activeUsers < 2 {
		room.Award()
	} else {
		room.activePlayer = room.GetPlayer(activeSeat)
		room.Turn()
	}
}

func (room *TexasRoom) Turn() {
	act := room.activePlayer
	room.deadline = time.Now().Add(room.maxAutoTime())
	act.operateTimer = utils.NewTimer(func() { act.TakeAction(-2) }, room.maxAutoTime())
	room.OnTurn()

	act.autoTimer = utils.NewTimer(act.AutoPlay, room.systemAutoTime())
}

func (room *TexasRoom) OnTurn() {
	for _, player := range room.GetAllPlayers() {
		p := player.GameAction.(*TexasPlayer)
		p.OnTurn()
	}
}

func (room *TexasRoom) NextSeat(seatId int) int {
	for i := 0; i < room.NumSeat(); i++ {
		nextId := (seatId + i + 1) % room.NumSeat()
		next := room.GetPlayer(nextId)
		if next != nil && next.IsPlaying() && next.bankroll > 0 {
			return next.GetSeatIndex()
		}
	}
	return roomutils.NoSeat
}

// 升盲
func (room *TexasRoom) AddBlind(smallBlind, bigBlind, frontBlind int64) {
	switch room.Status {
	case 0:
		room.smallBlind = smallBlind
		room.bigBlind = bigBlind
		room.blindLoop++
	case roomutils.RoomStatusPlaying:
		room.nextSmallBlind = smallBlind
		room.nextBigBlind = bigBlind
		room.delayAddBlind = true
	}
	room.OnAddBlind()
}

func (room *TexasRoom) OnAddBlind() {
	room.Broadcast("addBlind", map[string]any{
		"smallBlind":    room.smallBlind,
		"bigBlind":      room.bigBlind,
		"delayAddBlind": room.delayAddBlind,
	})
}

func (room *TexasRoom) systemAutoTime() time.Duration {
	d := systemAutoTime
	if t, ok := config.Duration("config", "texasAutoPlayDuration", "value"); ok {
		d = t
	}
	return d
}

func (room *TexasRoom) maxAutoTime() time.Duration {
	d := systemAutoTime
	if t, ok := config.Duration("config", "texasTimeoutDuration", "value"); ok {
		d = t
	}
	return d
}
