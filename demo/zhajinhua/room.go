package zhajinhua

import (
	"gofishing-game/internal/errcode"
	"gofishing-game/service"
	"math/rand"
	"third/cardutil"
	"third/gameutil"
	"time"

	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
	"github.com/guogeer/quasar/randutil"
	"github.com/guogeer/quasar/utils"
)

var (
	maxPayTime         = 60 * time.Second // 充值时间
	maxAutoTime        = 30 * time.Second
	systemAutoPlayTime = 1000 * time.Millisecond
)

const (
	OptMengpailunshu1 = iota + 1
	OptMengpailunshu2
	OptMengpailunshu3
	_ // 轮数占用
	_ // 轮数占用
)

const (
	OptLunshu10 = service.OptLunshu10
	OptLunshu20 = service.OptLunshu20
)

type ZhajinhuaRoom struct {
	*service.Room

	activePlayer *ZhajinhuaPlayer
	winner       *ZhajinhuaPlayer

	deadline time.Time
	helper   *cardutil.ZhajinhuaHelper

	allBet [64]int64
	// 本轮下注的筹码
	maxBet, currentChip            int64
	loop, lookLoopLimit, loopLimit int
	compareLoopLimit               int

	// 庄家座位
	dealerSeatId int
	chips        []int64
}

func (room *ZhajinhuaRoom) OnEnter(player *service.Player) {
	room.Room.OnEnter(player)

	comer := player.GameAction.(*ZhajinhuaPlayer)
	log.Infof("player %d enter room %d", comer.Id, room.Id)

	seatId := room.GetEmptySeat()
	if seatId != roomutils.NoSeat && comer.SeatId == roomutils.NoSeat {
		comer.RoomObj.SitDown(seatId)

		info := comer.GetUserInfo(false)
		room.Broadcast("SitDown", map[string]any{"Code": Ok, "Info": info}, comer.Id)
	}

	// 玩家重连
	data := map[string]any{
		"Status":        room.Status,
		"SubId":         room.GetSubId(),
		"Countdown":     room.GetShowTime(room.deadline),
		"CurrentLoop":   room.loop + 1,
		"LookLoopLimit": room.lookLoopLimit,
		"LoopLimit":     room.loopLimit,
	}
	if room.dealerSeatId >= 0 {
		data["Dealer"] = room.dealerSeatId
	}

	var seats []*ZhajinhuaUserInfo
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil {
			info := p.GetUserInfo(comer.Id == p.Id)
			seats = append(seats, info)
		}
	}
	data["SeatPlayers"] = seats

	// 玩家可能没座位
	comer.WriteJSON("GetRoomInfo", data)
	if room.Status == service.RoomStatusPlaying {
		comer.OnTurn()
	}
}

func (room *ZhajinhuaRoom) Leave(player *service.Player) ErrCode {
	ply := player.GameAction.(*ZhajinhuaPlayer)
	log.Debugf("player %d leave room %d", ply.Id, room.Id)
	return Ok
}

func (room *ZhajinhuaRoom) OnLeave(player *service.Player) {
	room.Room.OnLeave(player)
}

func (room *ZhajinhuaRoom) OnCreate() {
	room.dealerSeatId = -1

	room.loop = 0
	room.lookLoopLimit = 0
	if room.CanPlay(OptMengpailunshu1) {
		room.lookLoopLimit = 1
	} else if room.CanPlay(OptMengpailunshu2) {
		room.lookLoopLimit = 2
	} else if room.CanPlay(OptMengpailunshu3) {
		room.lookLoopLimit = 3
	}
	room.loopLimit = 8
	if room.CanPlay(OptLunshu10) {
		room.loopLimit = 10
	} else if room.CanPlay(OptLunshu20) {
		room.loopLimit = 20
	}
	// 最大轮数
	{
		n, ok := config.Int("zhajinhua_room", room.GetSubId(), "LoopLimit")
		if ok == true {
			room.loopLimit = int(n)
		}
	}
	// 比牌轮数
	{
		room.compareLoopLimit = 1
		n, ok := config.Int("zhajinhua_room", room.GetSubId(), "CompareLoopLimit")
		if ok == true {
			room.compareLoopLimit = int(n)
		}
	}

	// 闷牌轮数
	{
		n, ok := config.Int("zhajinhua_room", room.GetSubId(), "LookLoopLimit")
		if ok == true {
			room.lookLoopLimit = int(n)
		}
	}
	// 最大押注
	{
		n, ok := config.Int("zhajinhua_room", room.GetSubId(), "MaxBet")
		room.maxBet = 0
		if ok == true {
			room.maxBet = n
		}
	}

	room.Room.OnCreate()
}

func (room *ZhajinhuaRoom) Award() {
	guid := util.GUID()
	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		if p != nil && p.RoomObj.IsReady() {
			p.winGold = 0
		}
	}

	room.deadline = time.Now().Add(room.RestartTime())
	sec := room.GetShowTime(room.deadline)

	winner := room.winner
	details := make([]CompareResult, 0, 4)
	if winner == nil {
		var compareUsers, activeUsers int
		for i := 0; i < room.NumSeat(); i++ {
			if p := room.GetPlayer(i); p != nil && p.IsPlaying() {
				activeUsers++
			}
		}
		for i := 0; i < room.NumSeat(); i++ {
			seatId := (room.dealerSeatId + 1 + i) % room.NumSeat()
			if p := room.GetPlayer(seatId); p != nil && p.IsPlaying() {
				compareUsers++
				if winner == nil {
					winner = p
				} else {
					seats := []int{winner.SeatId, p.SeatId}
					detail := CompareResult{
						Seats:        seats,
						CompareSeats: seats,
					}
					if compareUsers+1 == activeUsers {
						resultA := CardResult{Cards: winner.cards}
						resultA.CardType, _ = room.helper.GetType(resultA.Cards)
						resultB := CardResult{Cards: p.cards}
						resultB.CardType, _ = room.helper.GetType(resultB.Cards)
						detail.CompareResults = []CardResult{resultA, resultB}
					}
					if room.helper.Less(p.cards, winner.cards) == false {
						winner = p
					}
					detail.Winner = winner.Id
					details = append(details, detail)
				}
			}
		}
		room.winner = winner
	}

	isAllRobot := true
	for i := 0; i < room.NumSeat(); i++ {
		user := room.StartGameUsers[i]
		if bet := room.allBet[i]; bet != 0 && user != nil && !user.IsRobot {
			isAllRobot = false
		}
	}
	for i := 0; !isAllRobot && i < room.NumSeat(); i++ {
		var balance int64 = -1
		if p := room.GetPlayer(i); p != nil {
			balance = p.Gold
		}
		user := room.StartGameUsers[i]
		if bet := room.allBet[i]; bet != 0 && user != nil {
			service.AddSomeItemLog(user.Id, []*Item{{Id: gameutil.ItemIdGold, Num: -bet, Balance: balance}}, guid, "user.zhajinhua_bet")
		}
	}
	winner.winGold = 0
	for _, gold := range room.allBet {
		winner.winGold += gold
	}

	if tax, ok := config.Float("Room", room.GetSubId(), "TaxPercent"); ok {
		winner.winGold = int64(float64(winner.winGold) * (100.0 - tax) / 100.0)
	}

	type UserDetail struct {
		UId       int
		Cards     []int
		CardType  int
		ExtraGold int64 `json:",omitempty"` // 返还的金币
	}
	users := make([]UserDetail, 0, room.NumSeat())
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil && p.RoomObj.IsReady() {
			typ, _ := room.helper.GetType(p.cards[:])
			detail := UserDetail{UId: p.Id, Cards: p.cards[:], CardType: typ, ExtraGold: p.extraGold}
			users = append(users, detail)
		}
	}
	room.Broadcast("Award", map[string]any{
		"Sec":            sec,
		"Winner":         winner.Id,
		"WinGold":        winner.winGold,
		"Users":          users,
		"CompareDetails": details,
	})

	winner.AddGold(winner.winGold, guid, "sum.zhajinhua_win")
	if isAllRobot == false {
		winner.AddGoldLog(winner.winGold, guid, "user.zhajinhua_win")
	}
	room.GameOver()
}

func (room *ZhajinhuaRoom) GameOver() {
	// 积分场最后一局
	if room.IsTypeScore() && room.ExistTimes+1 == room.LimitTimes {
		room.Broadcast("TotalAward", struct{}{})
	}
	room.Room.GameOver()

	room.winner = nil
	for i := range room.allBet {
		room.allBet[i] = 0
	}
	room.loop = 0
	room.currentChip = 0
	room.activePlayer = nil
}

func (room *ZhajinhuaRoom) CountActivePlayers() int {
	counter := 0
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil && p.IsPlaying() {
			counter++
		}
	}
	return counter
}

func (room *ZhajinhuaRoom) StartGame() {
	room.Room.StartGame()
	room.Status = service.RoomStatusPlaying
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil {
			p.isShow = false
			p.cards = nil
		}
	}
	// 初始化筹码
	room.chips = []int64{1, 2, 3, 4, 5}
	s, ok := config.String("zhajinhua_room", room.GetSubId(), "Chips")
	if ok == true {
		room.chips = nil
		chips := util.ParseIntSlice(s)
		if len(chips) > 0 && chips[0] > 0 {
			room.chips = chips
		}
	}

	dealerSeatId := room.dealerSeatId
	room.dealerSeatId = room.NextSeat(dealerSeatId)
	// choose dealer
	if host := GetPlayer(room.HostId); host != nil && dealerSeatId == -1 && host.Room() == room {
		room.dealerSeatId = host.SeatId
	}
	dealer := room.GetPlayer(room.dealerSeatId)
	if dealerSeatId != room.dealerSeatId {
		room.Broadcast("NewDealer", map[string]any{"UId": dealer.Id})
	}

	room.currentChip = room.Unit()

	activeSeats := make([]int, 0, 8)
	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		if p != nil && p.RoomObj.IsReady() {
			activeSeats = append(activeSeats, i)
		}
	}
	var samples []int64
	percent, _ := config.Float("zhajinhua_room", room.GetSubId(), "CardControlPercent")
	if randutil.IsPercentNice(percent) {
		s, _ := config.String("zhajinhua_room", room.GetSubId(), "CardSamples")
		samples = util.ParseIntSlice(s)
	}

	// start deal card
	start := rand.Intn(len(activeSeats))
	for i := range activeSeats {
		k := (start + i) % len(activeSeats)
		p := room.GetPlayer(activeSeats[k])
		if p != nil && p.RoomObj.IsReady() {
			cards := make([]int, room.helper.Size())
			if len(samples) > 1 {
				typ := randutil.Array(samples)
				table := room.CardSet().GetRemainingCards()
				if a := room.helper.Cheat(typ, table); a != nil {
					copy(cards, a)
					room.CardSet().Cheat(a...)
				}
			}
			if cards[0] == 0 {
				for k := range cards {
					cards[k] = room.CardSet().Deal()
				}
			}
			_, call, _ := p.Chips()
			p.cards = cards
			p.bet += call
			p.AddGold(-call, util.GUID(), "sum.zhajinhua_bet")
			room.allBet[p.SeatId] += call
		}
	}

	users := make([]StupidUser, 0, 8)
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil && p.IsPlaying() {
			users = append(users, StupidUser{UId: p.Id, SeatId: p.SeatId, Cards: p.cards, IsRobot: p.IsRobot})
		}
	}
	for _, p := range room.AllPlayers {
		data := map[string]any{"AllBet": room.allBet[:room.NumSeat()]}
		if p.IsRobot == true {
			data["Users"] = users
		}
		p.WriteJSON("StartDealCard", data)
	}

	log.Debug("start game", room.dealerSeatId)
	room.activePlayer = dealer
	room.Turn()
}

func (room *ZhajinhuaRoom) GetPlayer(seatId int) *ZhajinhuaPlayer {
	if seatId < 0 || seatId >= room.NumSeat() {
		return nil
	}
	if p := room.SeatPlayers[seatId]; p != nil {
		return p.GameAction.(*ZhajinhuaPlayer)
	}
	return nil
}

func (room *ZhajinhuaRoom) OnTakeAction() {
	loop := room.loop
	current := room.activePlayer
	if current == nil {
		return
	}
	nextId := room.NextSeat(current.SeatId)
	next := room.GetPlayer(nextId)

	var counter, allInUsers int
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil && p.IsPlaying() {
			counter++
			room.winner = p
			if p.isAllIn == true {
				allInUsers++
			}
		}
	}
	if counter > 1 {
		room.winner = nil
	}

	if current.loop == loop+1 && next.loop == loop+1 {
		room.NewRound()
	}
	// 金币场，两个玩家时，有人全压
	over := (counter == allInUsers)

	if room.winner != nil || room.loop >= room.loopLimit || over {
		room.Award()
	} else if current.loop == loop+1 {
		room.activePlayer = next
		room.Turn()
	}
}

// new round
func (room *ZhajinhuaRoom) NewRound() {
	room.loop++

	data := map[string]any{
		"Loop": room.loop,
	}
	room.Broadcast("NewRound", data)
}

func (room *ZhajinhuaRoom) maxAutoTime() time.Duration {
	d := maxAutoTime
	t, ok := config.Duration("zhajinhua_room", room.GetSubId(), "AutoDuration")
	if ok == true {
		d = t
	}
	return d
}

func (room *ZhajinhuaRoom) Turn() {
	current := room.activePlayer
	current.action = ActionNone
	room.deadline = time.Now().Add(room.maxAutoTime())
	current.AddTimer(service.TimerEventOperate, func() { current.TakeAction(-2) }, room.maxAutoTime())

	room.OnTurn()
	current.AutoPlay()
}

func (room *ZhajinhuaRoom) OnTurn() {
	for _, player := range room.AllPlayers {
		p := player.GameAction.(*ZhajinhuaPlayer)
		p.OnTurn()
	}
}

func (room *ZhajinhuaRoom) NextSeat(seatId int) int {
	for i := 0; i < room.NumSeat(); i++ {
		nextId := (seatId + i + 1) % room.NumSeat()
		next := room.GetPlayer(nextId)
		if next != nil && next.IsPlaying() {
			return next.SeatId
		}
	}
	return roomutils.NoSeat
}

func (room *ZhajinhuaRoom) IsAbleAllIn() bool {
	if len(room.chips) > 0 {
		return false
	}

	activeUsers := room.CountActivePlayers()
	if room.IsTypeScore() == false && activeUsers == 2 {
		return true
	}
	return false
}
