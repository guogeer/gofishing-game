package paodekuai

import (
	"gofishing-game/internal/errcode"
	"gofishing-game/service"
	"math/rand"
	"third/cardutil"

	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"

	// "third/pb"
	// "third/rpc"
	"time"

	"github.com/guogeer/quasar/utils"
	// "golang.org/x/net/context"
)

const (
	OptXianshipai = iota + 10
	OptBuxianshipai
	OptZhuangjiaxianchu
	OptHeitaosanxianchu
	OptHeitaosanbichu
	OptBixuguan
	OptKebuguan
	OptHongtaoshizhaniao
	OptCard15
	OptCard16
	OptShoulunheitaosanxianchu // 首轮黑桃3先出
	OptSidaisan                // 四带三
	OptSandaidui               // 三带对
	OptMeilunheitaosanbichu    // 每轮黑桃3必出
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

type PaodekuaiRoom struct {
	*service.Room

	helper *cardutil.PaodekuaiHelper

	dealer, nextDealer  *PaodekuaiPlayer
	discardPlayer       *PaodekuaiPlayer
	expectDiscardPlayer *PaodekuaiPlayer
	spades3Player       *PaodekuaiPlayer // 摸到黑桃三的玩家
	hearts10Player      *PaodekuaiPlayer // 摸到红桃10玩家
	winPlayer           *PaodekuaiPlayer

	autoTime time.Time
}

func (room *PaodekuaiRoom) OnEnter(player *service.Player) {
	room.Room.OnEnter(player)

	comer := player.GameAction.(*PaodekuaiPlayer)
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
		"SubId":     room.GetSubId(),
		"Countdown": room.GetShowTime(room.autoTime),
	}

	var seats []*PaodekuaiUserInfo
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

func (room *PaodekuaiRoom) Leave(player *service.Player) ErrCode {
	ply := player.GameAction.(*PaodekuaiPlayer)
	log.Debugf("player %d leave room %d", ply.Id, room.Id)
	return Ok
}

func (room *PaodekuaiRoom) OnLeave(player *service.Player) {
	room.Room.OnLeave(player)

	room.nextDealer = nil
}

func (room *PaodekuaiRoom) OnCreate() {
	room.CardSet().Recover(cardutil.GetAllCards()...)
	if room.CanPlay(OptCard15) {
		room.CardSet().Remove(0xf0, 0xf1, 0x02, 0x12, 0x22, 0x0e, 0x1e, 0x2e, 0x0d)
	} else {
		room.CardSet().Remove(0xf0, 0xf1, 0x02, 0x12, 0x22, 0x0e)
	}
	room.Room.OnCreate()

	helper := room.helper
	helper.Sidaisan = false
	helper.Sandaidui = false
	if room.CanPlay(OptSidaisan) {
		helper.Sidaisan = true
	}
	if room.CanPlay(OptSandaidui) {
		helper.Sandaidui = true
	}
}

func (room *PaodekuaiRoom) StartGame() {
	room.Room.StartGame()

	// 选庄家
	if room.nextDealer != nil {
		room.dealer = room.nextDealer
	}
	room.nextDealer = nil

	// 发牌
	room.StartDealCard()
	// 黑桃三当庄
	if p := room.spades3Player; room.dealer == nil && p != nil {
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

	if p := room.spades3Player; p != nil {
		room.expectDiscardPlayer = p
		p.forceDiscardCard = 0x33 // 黑桃三
	} else {
		room.expectDiscardPlayer = room.dealer
	}
	room.Turn()
}

func (room *PaodekuaiRoom) Award() {
	if room.CanPlay(OptMeilunheitaosanbichu) == false {
		room.nextDealer = room.winPlayer
	}

	guid := util.GUID()
	way := service.GetName()
	unit, _ := config.Int("Room", room.GetSubId(), "Unit")

	winPlayer := room.winPlayer
	winSeatId := winPlayer.SeatId
	winPlayer.winTimes++

	bills := make([]Bill, room.NumSeat())
	bills[winSeatId].Hearts10 = (room.hearts10Player == winPlayer)
	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		bills[i].Cards = p.GetSortedCards()
		// 有炸弹
		if t := p.boomTimes; t > 0 {
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

			if cardNum == 1 {
				cardNum = 0
			}
			loseGold := int64(cardNum) * unit
			if bill.Spring {
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

	// room.Status = service.RoomStatusFree
	room.autoTime = time.Now().Add(120 * time.Second)
	sec := room.GetShowTime(room.autoTime)
	room.Broadcast("Award", map[string]any{"Details": bills, "Times": sec, "Sec": sec})

	room.GameOver()
}

func (room *PaodekuaiRoom) GameOver() {
	if room.IsTypeScore() && room.ExistTimes+1 == room.LimitTimes {
		type TotalAwardInfo struct {
			// 炸弹数
			Boom int
			// 赢的局数
			WinTimes int
			// 最大赢取金币
			MaxWinGold int64
			Gold       int64
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
				})
			}
		}
		room.Broadcast("TotalAward", map[string]any{"Details": details})
	}
	room.Room.GameOver()

	room.dealer = nil
	room.discardPlayer = nil
	room.expectDiscardPlayer = nil

	room.spades3Player = nil
	room.hearts10Player = nil
	room.winPlayer = nil
}

func (room *PaodekuaiRoom) StartDealCard() {
	// 发牌
	room.autoTime = time.Now().Add(maxOperateTime)
	sec := room.GetShowTime(room.autoTime)

	n := 16
	if room.CanPlay(OptCard15) {
		n = 15
	}

	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		for k := 0; k < n; k++ {
			c := room.CardSet().Deal()
			if c == -1 {
				break
			}
			p.cards[c]++

			if c == 0x33 && room.ExistTimes == 0 && room.CanPlay(OptShoulunheitaosanxianchu) { // 黑桃三
				room.spades3Player = p
			}
			if c == 0x33 && room.CanPlay(OptMeilunheitaosanbichu) {
				room.spades3Player = p
			}
			if c == 0x2a && room.CanPlay(OptHongtaoshizhaniao) { // 红桃10
				room.hearts10Player = p
			}
		}
		log.Debug("start deal card", p.GetSortedCards())
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

func (room *PaodekuaiRoom) GetPlayer(seatId int) *PaodekuaiPlayer {
	if seatId < 0 || seatId >= room.NumSeat() {
		return nil
	}
	if p := room.SeatPlayers[seatId]; p != nil {
		return p.GameAction.(*PaodekuaiPlayer)
	}
	return nil
}

func (room *PaodekuaiRoom) Turn() {
	// 没人可以出牌，选取出牌的下家当庄家
	current := room.expectDiscardPlayer
	if p := room.discardPlayer; p != nil {
		cards := p.GetSortedCards()
		next := room.GetPlayer((current.SeatId + 1) % room.NumSeat())
		// 最后出的炸弹才给钱
		if len(cards) == 0 || p == next {
			typ, _, _ := room.helper.GetType(p.action)
			if typ == cardutil.PaodekuaiZhadan {
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

func (room *PaodekuaiRoom) OnTurn() {
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
