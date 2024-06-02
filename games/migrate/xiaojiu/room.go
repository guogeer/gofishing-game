package xiaojiu

import (
	"gofishing-game/internal/errcode"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
	"math/rand"
	"time"

	"github.com/guogeer/quasar/log"
	"github.com/guogeer/quasar/utils"
)

const (
	maxBetTime = 18 * time.Second
)

const (
	roomOptMingjiu         = "明九"      // 明九
	roomOptAnjiu           = "暗九"      // 暗九
	roomOptLunzhuang       = "轮庄"      // 轮庄
	roomOptSuijizhuang     = "随机庄"     // 随机庄
	roomOptFangzhuzhuang   = "房主庄"     // 房主庄
	roomOptDanrenxianzhu10 = "单人限注_10" // 单人限注10
	roomOptDanrenxianzhu20 = "单人限注_20" // 单人限注20
	roomOptDanrenxianzhu30 = "单人限注_30" // 单人限注30
	roomOptDanrenxianzhu50 = "单人限注_50" // 单人限注50
	roomOptZhuangjiabie10  = "蹩十"      // 蹩十
)

type UserDetail struct {
	UId  int   `json:"uId,omitempty"`
	Gold int64 `json:"gold,omitempty"`
	// Cards []int
}

type XiaojiuRoom struct {
	*roomutils.Room

	dealer                *XiaojiuPlayer
	continuousDealerTimes int // 连续段改装次数
	deadline              time.Time
	areas                 [3]int64
	cards                 [4][2]int

	autoTimer *utils.Timer
}

func (room *XiaojiuRoom) OnEnter(player *service.Player) {
	comer := player.GameAction.(*XiaojiuPlayer)
	log.Infof("player %d enter room %d", comer.Id, room.Id)

	// 玩家重连
	data := map[string]any{
		"status": room.Status,
		"subId":  room.SubId,
		"ts":     room.Countdown(),
		"chips":  room.Chips(),
	}

	if room.Status == roomutils.RoomStatusPlaying {
		data["areas"] = room.areas
		data["myAreas"] = comer.areas
		data["cards"] = room.cards
	}

	var seats []*XiaojiuUserInfo
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

	comer.SetClientValue("roomInfo", data)
}

func (room *XiaojiuRoom) Leave(player *service.Player) errcode.Error {
	ply := player.GameAction.(*XiaojiuPlayer)
	log.Debugf("player %d leave room %d", ply.Id, room.Id)
	return nil
}

func (room *XiaojiuRoom) OnLeave(player *service.Player) {
	p := player.GameAction.(*XiaojiuPlayer)
	if p == room.dealer {
		room.dealer = nil
	}
}

func (room *XiaojiuRoom) Award() {
	if n := room.GetPlayValue("必压"); n > 0 {
		areaId := rand.Intn(len(room.areas))
		for i := 0; i < room.NumSeat(); i++ {
			if p := room.GetPlayer(i); p != nil && p != room.dealer {
				var sum int64
				for _, v := range p.areas {
					sum += v
				}
				if sum < int64(n) {
					p.Bet(areaId, int64(n)-sum)
				}
			}
		}
	}

	way := "sum." + roomutils.GetServerName(room.SubId)
	room.deadline = time.Now().Add(room.FreeDuration())
	for i := range room.cards {
		for k := range room.cards[i] {
			if room.cards[i][k] == 0 {
				room.cards[i][k] = room.CardSet().Deal()
			}
		}
	}
	type CardResult struct {
		Type     int // 11：对子；1、点数
		Points   int
		Win      int  // -1：玩家输；0：平；1、玩家赢
		IsBieshi bool // 憋十
	}
	type UserResult struct {
		UId     int
		WinGold int64
		Areas   []int64
	}

	getPoints := func(c int) int {
		points := c & 0x0f
		if points == 0x0e {
			return 1
		}
		return points
	}
	getType := func(cards [2]int) (int, int) {
		if cards[0]&0x0f == cards[1]&0x0f {
			return 11, getPoints(cards[0])
		}
		return 1, (getPoints(cards[0]) + getPoints(cards[1])) % 10
	}

	dealerAreaId := len(room.cards) - 1
	resultSet := make([]CardResult, len(room.cards))
	dealerType, dealerPoints := getType(room.cards[dealerAreaId])
	dealerResult := CardResult{Type: dealerType}
	if dealerType == 1 && dealerPoints == 0 && room.CanPlay(roomOptZhuangjiabie10) {
		dealerPoints = 1
		dealerResult.IsBieshi = true
	}
	dealerResult.Points = dealerPoints
	resultSet[dealerAreaId] = dealerResult

	for i := 0; i+1 < len(room.cards); i++ {
		typ, points := getType(room.cards[i])
		result := CardResult{Type: typ, Points: points}
		if dealerType > typ {
			result.Win = -1
		} else if dealerType < typ {
			result.Win = 1
		} else {
			if points == 1 && dealerResult.IsBieshi {
				result.Win = 0
			} else if points <= dealerPoints {
				result.Win = -1
			} else if points > dealerPoints {
				result.Win = 1
			}
		}
		resultSet[i] = result
	}

	users := make([]UserResult, 0, 8)
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil && p != room.dealer {
			p.winGold = 0
			for k, gold := range p.areas {
				var userMultiples, dealerMultiples int64
				switch resultSet[k].Win {
				case -1:
					userMultiples, dealerMultiples = 0, 1
				case 0:
					userMultiples, dealerMultiples = 1, 0
				case 1:
					userMultiples, dealerMultiples = 2, -1
				}
				p.winGold += userMultiples * gold
				room.dealer.winGold += dealerMultiples * gold
			}
			users = append(users, UserResult{UId: p.Id, WinGold: p.winGold, Areas: p.areas[:]})
		}
	}
	users = append(users, UserResult{UId: room.dealer.Id, WinGold: room.dealer.winGold})

	data := map[string]any{
		"cards":     room.cards,
		"users":     users,
		"resultSet": resultSet,
		"ts":        room.Countdown(),
	}
	room.Broadcast("award", data)
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil {
			p.AddGold(p.winGold, way, service.WithNoItemLog())
		}
	}

	for i := range room.areas {
		room.areas[i] = 0
	}
	room.GameOver()
}

func (room *XiaojiuRoom) GameOver() {
	// 积分场最后一局
	details := make([]UserDetail, 0, 8)
	activeUsers := room.NumSeatPlayer()
	if room.TimesByLoop+1 == room.LimitTimes*room.TimesPerLoop*activeUsers {
		for i := 0; i < room.NumSeat(); i++ {
			if p := room.GetPlayer(i); p != nil {
				details = append(details, UserDetail{UId: p.Id, Gold: p.NumGold() - roomutils.GetRoomObj(p.Player).OriginGold})
			}
		}
		room.Broadcast("totalAward", map[string]any{"details": details})
	}

	room.Room.GameOver()

	utils.StopTimer(room.autoTimer)
	if room.ExistTimes < room.LimitTimes {
		room.autoTimer = utils.NewTimer(room.StartGame, room.FreeDuration())
	}
	for i := range room.cards {
		for k := range room.cards[i] {
			room.cards[i][k] = 0
		}
	}
}

func (room *XiaojiuRoom) OnCreate() {
	room.TimesPerLoop = 5
}

func (room *XiaojiuRoom) StartGame() {
	room.Room.StartGame()
	room.chooseDealer()
	if room.dealer != nil {
		room.continuousDealerTimes++
	}

	t := maxBetTime
	room.Status = roomutils.RoomStatusPlaying
	room.deadline = time.Now().Add(t)
	if room.CanPlay(roomOptMingjiu) { // 明九
		for i := range room.cards {
			room.cards[i][0] = room.CardSet().Deal()
		}
	}

	utils.StopTimer(room.autoTimer)
	room.autoTimer = utils.NewTimer(room.Award, t)
	room.Broadcast("startBetting", map[string]any{"ts": room.Countdown(), "cards": room.cards})
}

func (room *XiaojiuRoom) chooseDealer() {
	var dealerSeatIndex = roomutils.NoSeat
	var seats = make([]int, 0, 8)
	var host = room.GetPlayer(room.HostSeatIndex())
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil {
			seats = append(seats, p.GetSeatIndex())
		}
	}
	if room.CanPlay(roomOptSuijizhuang) { // 随机庄
		if room.dealer == nil || room.continuousDealerTimes >= room.TimesPerLoop {
			dealerSeatIndex = seats[rand.Intn(len(seats))]
		}
	} else if room.CanPlay(roomOptFangzhuzhuang) && host != nil {
		if room.dealer == nil {
			seats = seats[:0]
			dealerSeatIndex = host.GetSeatIndex()
		}
	} else { // 轮庄
		if room.dealer == nil {
			dealerSeatIndex = seats[0]
		}
		if room.dealer != nil && room.continuousDealerTimes >= room.TimesPerLoop {
			startSeatId := room.dealer.GetSeatIndex()
			for i := 0; i < room.NumSeat(); i++ {
				seatId := (startSeatId + i + 1) % room.NumSeat()
				if p := room.GetPlayer(seatId); p != nil {
					dealerSeatIndex = seatId
					break
				}
			}
		}
		seats = seats[:0]
	}
	if dealerSeatIndex != roomutils.NoSeat {
		room.continuousDealerTimes = 0
		room.dealer = room.GetPlayer(dealerSeatIndex)
		room.Broadcast("newDealer", map[string]any{"dealerId": room.dealer.Id, "seats": seats})
	}
}

func (room *XiaojiuRoom) GetPlayer(seatIndex int) *XiaojiuPlayer {
	if seatIndex < 0 || seatIndex >= room.NumSeat() {
		return nil
	}
	if p := room.FindPlayer(seatIndex); p != nil {
		return p.GameAction.(*XiaojiuPlayer)
	}
	return nil
}

func (room *XiaojiuRoom) Chips() []int64 {
	return []int64{1, 2, 5, 10, 20}
}

func (room *XiaojiuRoom) betLimitPerUser() int64 {
	var limit int64 = 10
	if room.CanPlay(roomOptDanrenxianzhu10) { // 单人限注10
		limit = 10
	} else if room.CanPlay(roomOptDanrenxianzhu20) { // 单人限注20
		limit = 20
	} else if room.CanPlay(roomOptDanrenxianzhu30) { // 单人限注30
		limit = 30
	} else if room.CanPlay(roomOptDanrenxianzhu50) { // 单人限注50
		limit = 50
	}
	return limit
}
