package zhajinhua

import (
	"gofishing-game/games/migrate/internal/cardrule"
	"gofishing-game/internal/gameutils"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
	"math/rand"
	"time"

	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
	"github.com/guogeer/quasar/utils/randutils"
)

var (
	maxPayTime         = 60 * time.Second // 充值时间
	maxAutoTime        = 30 * time.Second
	systemAutoPlayTime = 1000 * time.Millisecond
)

const (
	OptMengpailunshu1 = "menpailunshu_1"
	OptMengpailunshu2 = "menpailunshu_2"
	OptMengpailunshu3 = "menpailunshu_3"
)

const (
	OptLunshu10 = "lunshu_10"
	OptLunshu20 = "lunshu_20"
)

type ZhajinhuaRoom struct {
	*roomutils.Room

	activePlayer *ZhajinhuaPlayer
	winner       *ZhajinhuaPlayer

	helper *cardrule.ZhajinhuaHelper

	allBet [64]int64
	// 本轮下注的筹码
	maxBet, currentChip            int64
	loop, lookLoopLimit, loopLimit int
	compareLoopLimit               int

	// 庄家座位
	dealerSeatIndex int
	chips           []int64
}

func (room *ZhajinhuaRoom) OnEnter(player *service.Player) {
	comer := player.GameAction.(*ZhajinhuaPlayer)
	log.Infof("player %d enter room %d", comer.Id, room.Id)

	// 玩家重连
	data := map[string]any{
		"status":        room.Status,
		"subId":         room.SubId,
		"countdown":     room.Countdown(),
		"currentLoop":   room.loop + 1,
		"lookLoopLimit": room.lookLoopLimit,
		"loopLimit":     room.loopLimit,
	}
	if room.dealerSeatIndex >= 0 {
		data["dealer"] = room.dealerSeatIndex
	}

	var seats []*ZhajinhuaUserInfo
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil {
			info := p.GetUserInfo(comer.Id == p.Id)
			seats = append(seats, info)
		}
	}
	data["seatPlayers"] = seats

	// 玩家可能没座位
	comer.SetClientValue("roomInfo", data)
	if room.Status == 0 {
		comer.OnTurn()
	}
}

func (room *ZhajinhuaRoom) OnCreate() {
	room.dealerSeatIndex = -1

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
		n, ok := config.Int("zhajinhua_room", room.SubId, "loopLimit")
		if ok {
			room.loopLimit = int(n)
		}
	}
	// 比牌轮数
	{
		room.compareLoopLimit = 1
		n, ok := config.Int("zhajinhua_room", room.SubId, "compareLoopLimit")
		if ok {
			room.compareLoopLimit = int(n)
		}
	}

	// 闷牌轮数
	{
		n, ok := config.Int("zhajinhua_room", room.SubId, "lookLoopLimit")
		if ok {
			room.lookLoopLimit = int(n)
		}
	}
	// 最大押注
	{
		n, ok := config.Int("zhajinhua_room", room.SubId, "maxBet")
		room.maxBet = 0
		if ok {
			room.maxBet = n
		}
	}
}

func (room *ZhajinhuaRoom) Award() {
	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		if p != nil && roomutils.GetRoomObj(p.Player).IsReady() {
			p.winGold = 0
		}
	}

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
			seatId := (room.dealerSeatIndex + 1 + i) % room.NumSeat()
			if p := room.GetPlayer(seatId); p != nil && p.IsPlaying() {
				compareUsers++
				if winner == nil {
					winner = p
				} else {
					seats := []int{winner.GetSeatIndex(), p.GetSeatIndex()}
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
					if !room.helper.Less(p.cards, winner.cards) {
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
		user := room.StartGameUsers[i]
		if bet := room.allBet[i]; bet != 0 && user != nil {
			service.AddSomeItemLog(user.Id, []gameutils.Item{&gameutils.NumericItem{Id: gameutils.ItemIdGold, Num: -bet}}, "user.zhajinhua_bet")
		}
	}
	winner.winGold = 0
	for _, gold := range room.allBet {
		winner.winGold += gold
	}

	if tax, ok := config.Float("room", room.SubId, "taxPercent"); ok {
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
		if p := room.GetPlayer(i); p != nil && roomutils.GetRoomObj(p.Player).IsReady() {
			typ, _ := room.helper.GetType(p.cards[:])
			detail := UserDetail{UId: p.Id, Cards: p.cards[:], CardType: typ, ExtraGold: p.extraGold}
			users = append(users, detail)
		}
	}
	room.Broadcast("award", map[string]any{
		"countdown":      room.Countdown(),
		"winner":         winner.Id,
		"winGold":        winner.winGold,
		"users":          users,
		"compareDetails": details,
	})

	winner.BagObj().Add(gameutils.ItemIdGold, winner.winGold, "zhajinhua_win", service.WithNoItemLog())
	if !isAllRobot {
		service.AddSomeItemLog(winner.Id, []gameutils.Item{&gameutils.NumericItem{Id: gameutils.ItemIdGold, Num: winner.winGold}}, "user.zhajinhua_win")
	}
	room.GameOver()
}

func (room *ZhajinhuaRoom) GameOver() {
	// 积分场最后一局
	if room.IsTypeScore() && room.ExistTimes+1 == room.LimitTimes {
		room.Broadcast("totalAward", struct{}{})
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
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil {
			p.isShow = false
			p.cards = nil
		}
	}
	// 初始化筹码
	room.chips = []int64{1, 2, 3, 4, 5}
	config.Scan("zhajinhua_room", room.SubId, "chips", config.JSON(&room.chips))

	dealerSeatId := room.dealerSeatIndex
	room.dealerSeatIndex = room.NextSeat(dealerSeatId)
	// choose dealer
	if host := room.GetPlayer(room.HostSeatIndex()); host != nil && dealerSeatId == -1 && host.Room() == room {
		room.dealerSeatIndex = host.GetSeatIndex()
	}
	dealer := room.GetPlayer(room.dealerSeatIndex)
	if dealerSeatId != room.dealerSeatIndex {
		room.Broadcast("newDealer", map[string]any{"uid": dealer.Id})
	}

	room.currentChip = room.Unit()

	activeSeats := make([]int, 0, 8)
	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		if p != nil && roomutils.GetRoomObj(p.Player).IsReady() {
			activeSeats = append(activeSeats, i)
		}
	}
	var samples []int64
	percent, _ := config.Float("zhajinhua_room", room.SubId, "cardControlPercent")
	if randutils.IsPercentNice(percent) {
		config.Scan("zhajinhua_room", room.SubId, "cardSamples", config.JSON(&samples))
	}

	// start deal card
	start := rand.Intn(len(activeSeats))
	for i := range activeSeats {
		k := (start + i) % len(activeSeats)
		p := room.GetPlayer(activeSeats[k])
		if p != nil && roomutils.GetRoomObj(p.Player).IsReady() {
			cards := make([]int, room.helper.Size())
			if len(samples) > 1 {
				typ := randutils.Index(samples)
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
			p.BagObj().Add(gameutils.ItemIdGold, -call, "zhajinhua_bet", service.WithNoItemLog())
			room.allBet[p.GetSeatIndex()] += call
		}
	}

	users := make([]StupidUser, 0, 8)
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil && p.IsPlaying() {
			users = append(users, StupidUser{UId: p.Id, SeatIndex: p.GetSeatIndex(), Cards: p.cards, IsRobot: p.IsRobot})
		}
	}
	for _, p := range room.GetAllPlayers() {
		data := map[string]any{"allBet": room.allBet[:room.NumSeat()]}
		if p.IsRobot {
			data["users"] = users
		}
		p.WriteJSON("startDealCard", data)
	}

	log.Debug("start game", room.dealerSeatIndex)
	room.activePlayer = dealer
	room.Turn()
}

func (room *ZhajinhuaRoom) GetPlayer(seatIndex int) *ZhajinhuaPlayer {
	if seatIndex < 0 || seatIndex >= room.NumSeat() {
		return nil
	}
	if p := room.FindPlayer(seatIndex); p != nil {
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
	nextId := room.NextSeat(current.GetSeatIndex())
	next := room.GetPlayer(nextId)

	var counter, allInUsers int
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil && p.IsPlaying() {
			counter++
			room.winner = p
			if p.isAllIn {
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
		"loop": room.loop,
	}
	room.Broadcast("newRound", data)
}

func (room *ZhajinhuaRoom) maxAutoTime() time.Duration {
	d := maxAutoTime
	t, ok := config.Duration("zhajinhua_room", room.SubId, "autoDuration")
	if ok {
		d = t
	}
	return d
}

func (room *ZhajinhuaRoom) Turn() {
	current := room.activePlayer
	current.action = ActionNone
	current.operateTimer = current.TimerGroup.NewTimer(func() { current.TakeAction(-2) }, room.maxAutoTime())

	room.OnTurn()
	current.AutoPlay()
}

func (room *ZhajinhuaRoom) OnTurn() {
	for _, player := range room.GetAllPlayers() {
		p := player.GameAction.(*ZhajinhuaPlayer)
		p.OnTurn()
	}
}

func (room *ZhajinhuaRoom) NextSeat(seatId int) int {
	for i := 0; i < room.NumSeat(); i++ {
		nextId := (seatId + i + 1) % room.NumSeat()
		next := room.GetPlayer(nextId)
		if next != nil && next.IsPlaying() {
			return next.GetSeatIndex()
		}
	}
	return roomutils.NoSeat
}

func (room *ZhajinhuaRoom) IsAbleAllIn() bool {
	if len(room.chips) > 0 {
		return false
	}

	activeUsers := room.CountActivePlayers()
	if !room.IsTypeScore() && activeUsers == 2 {
		return true
	}
	return false
}
