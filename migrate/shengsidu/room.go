package shengsidu

import (
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
	"math/rand"
	"third/cardutil"

	"github.com/guogeer/quasar/log"
	"github.com/guogeer/quasar/util"

	// "third/pb"
	// "third/rpc"
	"time"
	// "golang.org/x/net/context"
)

const (
	_ = 10 + iota
	OptXianshipai
	OptBuxianshipai
	OptZhuangjiaxianchu
	OptHeitaosanxianchu
	OptHeitaosanbichu
	OptBixuguan
	OptKebuguan
	OptHongtaoshizhaniao
	OptMeilunfangpian3xianchu // 每轮方片3先出
	OptMeilunyingjiaxianchu   // 每轮赢家先出
)

var (
	maxAutoTime    = 1500 * time.Millisecond
	maxOperateTime = 16 * time.Second
)

type Bill struct {
	// 炸弹数
	Boom int `json:",omitempty"`
	// 剩余牌数
	CardNum  int   `json:",omitempty"`
	Cards    []int `json:",omitempty"`
	Gold     int64
	Spring   bool `json:",omitempty"` // 春天
	Hearts10 bool `json:",omitempty"` // 扎鸟
}

type ShengsiduRoom struct {
	*service.Room

	helper *cardutil.ShengsiduHelper

	dealer, nextDealer  *ShengsiduPlayer
	discardPlayer       *ShengsiduPlayer
	expectDiscardPlayer *ShengsiduPlayer
	diamonds3Player     *ShengsiduPlayer // 摸到方片3的玩家
	hearts10Player      *ShengsiduPlayer // 摸到红桃10玩家
	winPlayer           *ShengsiduPlayer

	autoTime time.Time
}

func (room *ShengsiduRoom) OnEnter(player *service.Player) {
	room.Room.OnEnter(player)

	comer := player.GameAction.(*ShengsiduPlayer)
	log.Infof("player %d enter room %d", comer.Id, room.Id)

	// 自动坐下
	seatId := room.GetEmptySeat()
	if player.SeatId == roomutils.NoSeat && seatId != roomutils.NoSeat {
		// comer.SitDown()
		comer.RoomObj.SitDown(seatId)

		info := comer.GetUserInfo(false)
		room.Broadcast("SitDown", map[string]any{"Code": Ok, "Info": info}, comer.Id)
	}

	// 玩家重连
	data := map[string]any{
		"Status":    room.Status,
		"SubId":     room.SubId,
		"Countdown": room.GetShowTime(room.autoTime),
	}

	var seats []*ShengsiduUserInfo
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil {
			info := p.GetUserInfo(comer.Id == p.Id)
			seats = append(seats, info)
		}
	}
	data["SeatPlayers"] = seats

	// 玩家可能没座位
	comer.WriteJSON("GetRoomInfo", data)

	if room.Status != service.RoomStatusFree {
		room.OnTurn()
	}
}

func (room *ShengsiduRoom) Leave(player *service.Player) ErrCode {
	ply := player.GameAction.(*ShengsiduPlayer)
	log.Debugf("player %d leave room %d", ply.Id, room.Id)
	return Ok
}

func (room *ShengsiduRoom) OnLeave(player *service.Player) {
	room.Room.OnLeave(player)

	room.nextDealer = nil
}

func (room *ShengsiduRoom) OnCreate() {
	room.CardSet().Recover(cardutil.GetAllCards()...)
	room.Room.OnCreate()
}

func (room *ShengsiduRoom) StartGame() {
	room.Room.StartGame()

	// 选庄家
	if room.nextDealer != nil {
		room.dealer = room.nextDealer
	}
	room.nextDealer = nil

	// 发牌
	room.StartDealCard()
	// 黑桃三当庄
	if p := room.diamonds3Player; room.dealer == nil && p != nil {
		room.dealer = p
	}
	// 房主当庄
	if host := GetPlayer(room.HostId); room.dealer == nil && host != nil && host.Room() == room {
		room.dealer = host
	}
	// 随机
	if room.dealer == nil {
		seatId := rand.Intn(room.NumSeat())
		room.dealer = room.GetPlayer(seatId)
	}
	room.Broadcast("NewDealer", map[string]any{"UId": room.dealer.Id})

	if p := room.diamonds3Player; p != nil {
		room.expectDiscardPlayer = p
	} else {
		room.expectDiscardPlayer = room.dealer
	}
	room.Turn()
}

func (room *ShengsiduRoom) Award() {
	if room.CanPlay(OptMeilunyingjiaxianchu) {
		room.nextDealer = room.winPlayer
	}

	guid := util.GUID()
	way := service.GetName()
	unit := room.Unit()

	winPlayer := room.winPlayer
	winSeatId := winPlayer.SeatId
	winPlayer.winTimes++

	springNum := 0
	bills := make([]Bill, room.NumSeat())
	bills[winSeatId].Hearts10 = (room.hearts10Player == winPlayer)
	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		bills[i].Cards = p.GetSortedCards()
		// 有炸弹
		if t := p.boomTimes; t > 0 && false {
			for k := 0; k < room.NumSeat(); k++ {
				if other := room.GetPlayer(k); p != other {
					gold := int64(t) * 10 * unit
					bills[k].Gold -= gold
					bills[p.SeatId].Gold += gold
				}
			}
		}

		bill := &bills[i]
		bill.Boom = p.boomTimes
		if p != winPlayer {
			cards := p.GetSortedCards()

			cardNum := len(cards)
			bill.CardNum = cardNum
			bill.Boom = p.boomTimes

			bill.Spring = (p.discardNum == 0)          // 春天
			bill.Hearts10 = (room.hearts10Player == p) // 红桃10扎鸟
			if bill.Spring {
				springNum++
			}
		}
	}

	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		bill := &bills[i]
		if p != nil {
			loseGold := unit
			if bill.Spring {
				loseGold <<= 1
			}
			if springNum+1 == room.NumSeat() {
				loseGold <<= 1
			}
			// 红桃10扎鸟
			if bills[winSeatId].Hearts10 ||
				bill.Hearts10 {
				loseGold <<= 1
			}
			bills[i].Gold -= loseGold
			bills[winSeatId].Gold += loseGold
		}
	}
	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		gold := bills[i].Gold
		if p.maxWinGold < gold {
			p.maxWinGold = gold
		}
		p.AddGold(gold, guid, way)
	}
	if springNum > 0 {
		if room.NumSeat() > 2 && springNum+1 < room.NumSeat() {
			for i := 0; i < room.NumSeat(); i++ {
				p := room.GetPlayer(i)
				if p != room.winPlayer {
					p.addition["BeiBanTong"] += 1
				}
			}
			if p := room.winPlayer; p != nil {
				p.addition["TongBieJia"] += 1
			}
		}
		if room.NumSeat() == 2 && springNum+1 == room.NumSeat() {
			for i := 0; i < room.NumSeat(); i++ {
				p := room.GetPlayer(i)
				if p != room.winPlayer {
					p.addition["BeiQuanTong"] += 1
				}
			}
			if p := room.winPlayer; p != nil {
				p.addition["TongBieJia"] += 1
			}
		}
		if room.NumSeat() > 2 && springNum+1 == room.NumSeat() {
			if p := room.winPlayer; p != nil {
				p.addition["YiTongTianXia"] += 1
			}
		}
	}
	// 统别家，旧版拼错
	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		if p != room.winPlayer {
			p.addition["TongBeiJia"] = p.addition["TongBieJia"]
		}
	}

	// room.Status = service.RoomStatusFree
	room.autoTime = time.Now().Add(120 * time.Second)
	sec := room.GetShowTime(room.autoTime)
	room.Broadcast("Award", map[string]any{"Details": bills, "Times": sec, "Sec": sec})

	room.GameOver()
}

func (room *ShengsiduRoom) GameOver() {
	if room.IsTypeScore() && room.ExistTimes+1 == room.LimitTimes {
		type TotalAwardInfo struct {
			// 炸弹数
			Boom int
			// 赢的局数
			WinTimes int
			// 最大赢取金币
			MaxWinGold int64
			Gold       int64
			Addition   map[string]int
		}

		// 积分场最后一局
		details := make([]TotalAwardInfo, 0, 8)
		for i := 0; i < room.NumSeat(); i++ {
			if p := room.GetPlayer(i); p != nil {
				details = append(details, TotalAwardInfo{
					Boom:       p.totalBoomTimes,
					WinTimes:   p.winTimes,
					MaxWinGold: p.maxWinGold,
					Gold:       p.Gold,
					Addition:   p.addition,
				})
			}
		}
		room.Broadcast("TotalAward", map[string]any{"Details": details})
	}
	room.Room.GameOver()

	room.dealer = nil
	room.discardPlayer = nil
	room.expectDiscardPlayer = nil

	room.diamonds3Player = nil
	room.hearts10Player = nil
	room.winPlayer = nil
}

func (room *ShengsiduRoom) StartDealCard() {
	// 发牌
	helper := room.helper
	room.autoTime = time.Now().Add(maxOperateTime)
	sec := room.GetShowTime(room.autoTime)

	diamonds := 0x02 // 方片2
	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		for k := 0; k < 13; k++ {
			c := room.CardSet().Deal()
			if c == -1 {
				break
			}
			p.cards[c]++

			isPioneer := false
			if room.ExistTimes == 0 { // 方片3
				isPioneer = true
			}
			if room.CanPlay(OptMeilunfangpian3xianchu) {
				isPioneer = true
			}
			if isPioneer && c&0xf0 == diamonds&0xf0 && helper.Value(c) <= helper.Value(diamonds) {
				diamonds = c
				room.diamonds3Player = p
			}
			if c == 0x2a && room.CanPlay(OptHongtaoshizhaniao) { // 红桃10
				room.hearts10Player = p
			}
		}
		log.Debug("start deal card", p.GetSortedCards())
	}
	if p := room.diamonds3Player; p != nil {
		p.forceDiscardCard = diamonds
	}

	data := map[string]any{
		"Sec": sec,
	}
	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		data["Cards"] = p.GetSortedCards()
		p.WriteJSON("StartDealCard", data)
	}
}

func (room *ShengsiduRoom) GetPlayer(seatId int) *ShengsiduPlayer {
	if seatId < 0 || seatId >= room.NumSeat() {
		return nil
	}
	if p := room.SeatPlayers[seatId]; p != nil {
		return p.GameAction.(*ShengsiduPlayer)
	}
	return nil
}

func (room *ShengsiduRoom) Turn() {
	// 没人可以出牌，选取出牌的下家当庄家
	current := room.expectDiscardPlayer
	if p := room.discardPlayer; p != nil {
		cards := p.GetSortedCards()
		next := room.GetPlayer((current.SeatId + 1) % room.NumSeat())
		// 最后出的炸弹才给钱
		if len(cards) == 0 || p == next {
			typ, _, _ := room.helper.GetType(p.action)
			if typ == cardutil.ShengsiduZhadan {
				p.boomTimes++
				p.totalBoomTimes++
			}
		}
		if len(cards) == 0 {
			room.winPlayer = p
			room.Award()
			return
		}

		room.expectDiscardPlayer = next
		// 新的一轮
		if p == next {
			room.discardPlayer = nil
		}
		next.AutoPlay()
	} else {
		current.AutoPlay()
	}
	room.OnTurn()
}

func (room *ShengsiduRoom) OnTurn() {
	current := room.expectDiscardPlayer

	data := map[string]any{
		"UId": current.Id,
		"Sec": room.GetShowTime(room.autoTime),
	}
	if c := current.forceDiscardCard; c > 0 {
		data["ForceCards"] = []int{c}
	}
	if p := room.discardPlayer; p == nil {
		data["NewLoop"] = true
	} else {
		if ans := room.helper.Match(current.GetSortedCards(), p.action); len(ans) == 0 {
			data["Pass"] = true
		}
	}
	room.Broadcast("Turn", data)
}
