package threedice

import (
	"container/list"
	"fmt"
	"gofishing-game/internal/errcode"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
	"math/rand"
	"time"

	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
	"github.com/guogeer/quasar/utils"
)

var (
	maxBetTime       = 16 * time.Second
	maxSyncTime      = 500 * time.Millisecond
	maxRobDealerTime = 4500 * time.Millisecond
	// 系统当庄
	systemDealerInfo = service.SimpleUserInfo{
		Nickname: "静静",
	}
)

// 摇色子结果集
var (
	baoziResultSet []int // 豹子结果集
	otherResultSet []int // 大或小结果集
)

func init() {
	helper := cardutils.NewThreeDiceHelper()
	for i := 1; i <= 6; i++ {
		for j := 1; j <= 6; j++ {
			for k := 1; k <= 6; k++ {
				typ := helper.GetType(i, j, k)
				if typ == cardutils.ThreeDiceBaozi {
					baoziResultSet = append(baoziResultSet, 100*i+10*j+k)
				} else {
					otherResultSet = append(otherResultSet, 100*i+10*j+k)
				}
			}
		}
	}
}

type ThreeDiceRoom struct {
	*roomutils.Room

	systemDealer *service.SimpleUserInfo
	dealer       *ThreeDicePlayer
	dealerGold   int64

	autoTimer     *utils.Timer
	deadline      time.Time
	tempDealer    int
	last          [64]int
	lasti         int
	robDealerList *list.List
	helper        *cardutils.ThreeDiceHelper

	areas, limit [2]int64
	syncTimer    *utils.Timer
}

func (room *ThreeDiceRoom) OnEnter(player *service.Player) {
	room.Room.OnEnter(player)

	comer := player.GameAction.(*ThreeDicePlayer)
	log.Infof("player %d enter room %d", comer.Id, room.Id)

	// 自动坐下
	seatId := room.GetEmptySeat()
	if comer.SeatIndex == roomutils.NoSeat && seatId != roomutils.NoSeat && !comer.IsRobot() {
		comer.SitDown(seatId)
	}

	// 玩家重连
	data := map[string]any{
		"Status":     room.Status,
		"SubId":      room.SubId,
		"Countdown":  room.Countdown(),
		"Chips":      room.Chips(),
		"History":    room.GetLastHistory(40),
		"Online":     len(room.AllPlayers),
		"BetUserNum": room.countBetUser(),
	}

	if room.Status == roomutils.RoomStatusPlaying {
		dealerInfo := room.systemDealer
		if p := room.dealer; p != nil {
			dealerInfo = p.SimpleInfo()
		}
		data["Dealer"] = dealerInfo
		data["DealerGold"] = room.dealerGold
		data["Areas"] = room.areas
		data["Limit"] = room.limit
	}
	if s, ok := config.String("threedice", "DealerRequiredGold", "Value"); ok {
		a := []int64{200000, 1000000} // 默认值
		for i, n := range util.ParseIntSlice(s) {
			if i < len(a) {
				a[i] = n
			}
		}
		data["DealerLimit"] = a
	}

	var seats []*ThreeDiceUserInfo
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

	comer.WriteJSON("GetRoomInfo", data)
}

func (room *ThreeDiceRoom) Leave(player *service.Player) errcode.Error {
	ply := player.GameAction.(*ThreeDicePlayer)
	log.Debugf("player %d leave room %d", ply.Id, room.Id)
	return nil
}

func (room *ThreeDiceRoom) OnLeave(player *service.Player) {
	room.Room.OnLeave(player)

	p := player.GameAction.(*ThreeDicePlayer)
	if p == room.dealer {
		room.dealer = nil
	}
}

func (room *ThreeDiceRoom) Award() {
	guid := util.GUID()
	way := "user." + service.GetServerName()

	baoziPercent := -0.1
	if room.dealer != nil {
		percent, ok := config.Float("threedice", "UserBaoziPercent", "Value")
		if ok {
			baoziPercent = percent
		}
	} else {
		percent, ok := config.Float("threedice", "SystemBaoziPercent", "Value")
		if ok {
			baoziPercent = percent
		}
	}

	var hash int
	if baoziPercent > 0 {
		resultSet := otherResultSet
		if util.Random().IsNice(int(float64(util.Random().Range()) * baoziPercent / 100)) {
			resultSet = baoziResultSet
		}
		hash = resultSet[rand.Intn(len(resultSet))]
	} else {
		for i := 0; i < 3; i++ {
			dice := rand.Intn(6) + 1
			hash = 10*hash + dice
		}
	}

	room.last[room.lasti] = hash
	room.lasti = (room.lasti + 1) % len(room.last)
	typ := room.helper.GetType(hash/100, hash/10%10, hash%10)
	tax, _ := config.Float("threedice", "SystemTax", "Value")

	var winAreaId = -1
	var dealerWinGold int64

	utils.StopTimer(room.autoTimer)
	room.autoTimer = utils.NewTimer(room.StartGame, room.RestartTime())
	room.deadline = time.Now().Add(room.RestartTime())

	sec := room.Countdown()
	if typ == cardutils.ThreeDiceXiao {
		winAreaId = 0
	} else if typ == cardutils.ThreeDiceDa {
		winAreaId = 1
	}
	// 大赢家
	type BigWinner struct {
		Info    *service.SimpleUserInfo
		WinGold int64
	}
	winner := &BigWinner{}
	for _, player := range room.AllPlayers {
		p := player.GameAction.(*ThreeDicePlayer)
		if p.areaId != -1 {
			gold := p.areas[p.areaId]
			if p.areaId != winAreaId {
				gold = -gold
			}

			dealerWinGold += -gold
			if gold > 0 {
				room.AutoBroadcast(p, gold)

				if winner.WinGold < gold {
					winner.WinGold = gold
					winner.Info = p.SimpleInfo()
				}

				gold = int64(float64(gold)*(1-tax)) + gold
			}
			p.winGold = gold
		}
	}
	data := map[string]any{
		"Sec":       sec,
		"Dices":     hash,
		"Type":      typ,
		"WinAreaId": winAreaId,
	}
	if winner.WinGold > 0 {
		data["BigWinner"] = winner
	}
	for _, player := range room.AllPlayers {
		p := player.GameAction.(*ThreeDicePlayer)

		gold := p.winGold
		data["MyWinGold"] = gold
		p.WriteJSON("Award", data)
		if gold > 0 {
			p.AddGold(gold, guid, way)
		}
	}

	if room.dealer != nil {
		room.AutoBroadcast(room.dealer, dealerWinGold)
		room.dealer.AddGold(int64(float64(dealerWinGold)*(1-tax)), guid, way)
	}

	room.dealer = nil
	for i := range room.areas {
		room.areas[i] = 0
	}
	room.GameOver()
}

func (room *ThreeDiceRoom) AutoBroadcast(p *ThreeDicePlayer, gold int64) {
	n, ok := config.Int("config", "AutoBroadcastMinGold", "Value")
	if ok && gold >= n {
		msg := fmt.Sprintf("恭喜%v赢得%d万金币", p.Nickname, gold/10000)
		service.Broadcast(0, "系统", 0, msg)
	}
}

func (room *ThreeDiceRoom) StartGame() {
	room.Room.StartGame()

	t := maxRobDealerTime
	if d, ok := config.Duration("threedice", "RobDealerTime", "Value"); ok {
		t = d
	}
	room.Status = roomutils.RoomStatusRobDealer
	room.deadline = time.Now().Add(t)

	min, max := room.dealerRequiredGold()
	room.dealerGold = rand.Int63n(max-min) + min
	room.Broadcast("StartRobDealer", map[string]any{"Sec": room.Countdown(), "MinGold": min, "MaxGold": max})
	utils.NewTimer(room.chooseDealer, t)
}

func (room *ThreeDiceRoom) chooseDealer() {
	var max int64

	dealerInfo := GetSystemDealer()
	dealers := make([]int, 0, 10)
	for e := room.robDealerList.Front(); e != nil; e = e.Next() {
		uid := e.Value.(int)
		p := GetPlayer(uid)
		if max == p.robDealerGold {
			dealers = append(dealers, uid)
		} else if max < p.robDealerGold {
			max = p.robDealerGold
			dealers = dealers[:0]
			dealers = append(dealers, uid)
		}
	}

	for {
		e := room.robDealerList.Back()
		if e == nil {
			break
		}
		room.robDealerList.Remove(e)
	}
	if n := len(dealers); n > 0 {
		dealerId := dealers[rand.Intn(n)]
		room.dealer = GetPlayer(dealerId)
		room.dealerGold = room.dealer.robDealerGold
		dealerInfo = room.dealer.SimpleInfo()
	}
	room.limit[0] = room.dealerGold
	room.limit[1] = room.dealerGold
	room.systemDealer = dealerInfo

	t := maxBetTime
	if d, ok := config.Duration("threedice", "BetTime", "Value"); ok {
		t = d
	}
	room.deadline = time.Now().Add(t)
	room.Broadcast("FinishRobDealer", map[string]any{
		"Dealer":     dealerInfo,
		"DealerGold": room.dealerGold,
		"Limit":      room.limit,
		"Sec":        room.Countdown(),
	})

	room.Status = roomutils.RoomStatusPlaying
	room.autoTimer = utils.NewTimer(room.Award, t)
}

func (room *ThreeDiceRoom) GetPlayer(seatId int) *ThreeDicePlayer {
	if seatId < 0 || seatId >= room.NumSeat() {
		return nil
	}
	if p := room.SeatPlayers[seatId]; p != nil {
		return p.GameAction.(*ThreeDicePlayer)
	}
	return nil
}

func (room *ThreeDiceRoom) Chips() []int64 {
	s, _ := config.String("threedice", "Chips", "Value")
	return util.ParseIntSlice(s)
}

func (room *ThreeDiceRoom) dealerRequiredGold() (int64, int64) {
	s, _ := config.String("threedice", "DealerRequiredGold", "Value")

	var temp [2]int64
	slice2 := util.ParseIntSlice(s)
	copy(temp[:], slice2)
	return temp[0], temp[1]
}

func (room *ThreeDiceRoom) OnBet() {
}

func (room *ThreeDiceRoom) GetLastHistory(n int) []int {
	var last []int
	N := len(room.last)
	if N == 0 {
		return last
	}
	for i := (N - n + room.lasti) % N; i != room.lasti; i = (i + 1) % N {
		d := room.last[i]
		if d > 0 {
			last = append(last, d)
		}
	}
	return last
}

func (room *ThreeDiceRoom) countBetUser() int {
	var counter int // 押注用户数
	for _, player := range room.AllPlayers {
		p := player.GameAction.(*ThreeDicePlayer)
		if p.areaId != -1 {
			counter++
		}
	}
	return counter
}

func (room *ThreeDiceRoom) Sync() {
	data := map[string]any{
		"Online":     len(room.AllPlayers),
		"BetUserNum": room.countBetUser(),
	}
	room.Broadcast("Sync", data)
}

func GetSystemDealer() *service.SimpleUserInfo {
	dealer := systemDealerInfo
	if n := config.Row("dice3_dealer"); n > 0 {
		rowId := config.RowId(rand.Intn(n))
		dealer.Nickname, _ = config.String("dice3_dealer", rowId, "Nickname")
	}
	if s, ok := config.String("threedice", "UserIcons", "Value"); ok {
		icons := util.ParseStrings(s)
		if n := len(icons); n > 0 {
			dealer.Icon = icons[rand.Intn(n)]
		}
	}
	return &dealer
}
