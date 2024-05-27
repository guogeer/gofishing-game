package paodekuai

import (
	"gofishing-game/internal/cardutils"
	"gofishing-game/migrate/internal/cardrule"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
	"math/rand"
	"time"

	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
)

const (
	OptXianshipai              = "xianshipai"
	OptBuxianshipai            = "buxianshipai"
	OptZhuangjiaxianchu        = "zhuangjiaxianchu"
	OptHeitaosanxianchu        = "heitaosanxianchu"
	OptHeitaosanbichu          = "heitaosanbichu"
	OptBixuguan                = "bixuguan"
	OptKebuguan                = "kebuguan"
	OptHongtaoshizhaniao       = "hongtaoshizhaniao"
	OptCard15                  = "card15"
	OptCard16                  = "card15"
	OptShoulunheitaosanxianchu = "shoulunheitaosanxianchu" // 首轮黑桃3先出
	OptSidaisan                = "sidaisan"                // 四带三
	OptSandaidui               = "sandaidui"               // 三带对
	OptMeilunheitaosanbichu    = "meilunheitaosanbichu"    // 每轮黑桃3必出
)

var (
	maxAutoTime    = 1500 * time.Millisecond
	maxOperateTime = 16 * time.Second
)

type Bill struct {
	// 炸弹数
	Boom int `json:"boom,omitempty"`
	// 剩余牌数
	CardNum  int   `json:"cardNum,omitempty"`
	Cards    []int `json:"cards,omitempty"`
	Gold     int64 `json:"gold,omitempty"`
	Spring   bool  `json:"spring,omitempty"`   // 春天
	Hearts10 bool  `json:"hearts10,omitempty"` // 扎鸟
}

type PaodekuaiRoom struct {
	*roomutils.Room

	helper *cardrule.PaodekuaiHelper

	dealer, nextDealer  *PaodekuaiPlayer
	discardPlayer       *PaodekuaiPlayer
	expectDiscardPlayer *PaodekuaiPlayer
	spades3Player       *PaodekuaiPlayer // 摸到黑桃三的玩家
	hearts10Player      *PaodekuaiPlayer // 摸到红桃10玩家
	winPlayer           *PaodekuaiPlayer

	autoTime time.Time
}

func (room *PaodekuaiRoom) OnEnter(player *service.Player) {
	comer := player.GameAction.(*PaodekuaiPlayer)
	log.Infof("player %d enter room %d", comer.Id, room.Id)

	// 玩家重连
	data := map[string]interface{}{
		"status":    room.Status,
		"subId":     room.SubId,
		"countdown": room.Countdown(),
	}

	var seats []*PaodekuaiUserInfo
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil {
			info := p.GetUserInfo(comer.Id == p.Id)
			seats = append(seats, info)
		}
	}
	data["seatPlayers"] = seats

	// 玩家可能没座位
	comer.SetClientValue("roomInfo", data)

	if room.Status != 0 {
		room.OnTurn()
	}
}

func (room *PaodekuaiRoom) OnLeave(player *service.Player) {
	room.nextDealer = nil
}

func (room *PaodekuaiRoom) OnCreate() {
	room.CardSet().Recover(cardutils.GetAllCards()...)
	if room.CanPlay(OptCard15) {
		room.CardSet().Remove(0xf0, 0xf1, 0x02, 0x12, 0x22, 0x0e, 0x1e, 0x2e, 0x0d)
	} else {
		room.CardSet().Remove(0xf0, 0xf1, 0x02, 0x12, 0x22, 0x0e)
	}

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
	if host := room.GetPlayer(room.HostSeatIndex()); room.dealer == nil && host != nil && host.Room() == room {
		room.dealer = host
	}
	// 随机
	if room.dealer == nil {
		seatId := rand.Intn(room.NumSeat())
		room.dealer = room.GetPlayer(seatId)
	}
	room.Broadcast("newDealer", map[string]interface{}{"uid": room.dealer.Id})

	if p := room.spades3Player; p != nil {
		room.expectDiscardPlayer = p
		p.forceDiscardCard = 0x33 // 黑桃三
	} else {
		room.expectDiscardPlayer = room.dealer
	}
	room.Turn()
}

func (room *PaodekuaiRoom) Award() {
	if !room.CanPlay(OptMeilunheitaosanbichu) {
		room.nextDealer = room.winPlayer
	}

	way := service.GetServerName()
	unit, _ := config.Int("room", room.SubId, "unit")

	winPlayer := room.winPlayer
	winSeatId := winPlayer.GetSeatIndex()
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
					bills[p.GetSeatIndex()].Gold += gold
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
		p.AddGold(gold, way)
	}

	room.GameOver()
	room.Broadcast("award", map[string]interface{}{"details": bills, "countdown": room.Countdown()})
}

func (room *PaodekuaiRoom) GameOver() {
	if room.IsTypeScore() && room.ExistTimes+1 == room.LimitTimes {
		type TotalAwardInfo struct {
			// 炸弹数
			Boom int `json:"boom,omitempty"`
			// 赢的局数
			WinTimes int `json:"winTimes,omitempty"`
			// 最大赢取金币
			MaxWinGold int64 `json:"maxWinGold,omitempty"`
			Gold       int64 `json:"gold,omitempty"`
		}

		// 积分场最后一局
		details := make([]TotalAwardInfo, 0, 8)
		for i := 0; i < room.NumSeat(); i++ {
			if p := room.GetPlayer(i); p != nil {
				details = append(details, TotalAwardInfo{
					Boom:       p.totalBoomTimes,
					WinTimes:   p.winTimes,
					MaxWinGold: p.maxWinGold,
					Gold:       p.NumGold(),
				})
			}
		}
		room.Broadcast("totalAward", map[string]interface{}{"details": details})
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

	data := map[string]interface{}{
		"countdown": room.Countdown(),
	}
	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		data["cards"] = p.GetSortedCards()
		p.WriteJSON("startDealCard", data)
	}
}

func (room *PaodekuaiRoom) GetPlayer(seatIndex int) *PaodekuaiPlayer {
	if seatIndex < 0 || seatIndex >= room.NumSeat() {
		return nil
	}
	if p := room.GetPlayer(seatIndex); p != nil {
		return p.GameAction.(*PaodekuaiPlayer)
	}
	return nil
}

func (room *PaodekuaiRoom) Turn() {
	// 没人可以出牌，选取出牌的下家当庄家
	current := room.expectDiscardPlayer
	if p := room.discardPlayer; p != nil {
		cards := p.GetSortedCards()
		next := room.GetPlayer((current.GetSeatIndex() + 1) % room.NumSeat())
		// 最后出的炸弹才给钱
		if len(cards) == 0 || p == next {
			typ, _, _ := room.helper.GetType(p.action)
			if typ == cardrule.PaodekuaiZhadan {
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

	data := map[string]interface{}{
		"uid":       current.Id,
		"countdown": room.Countdown(),
	}
	if c := current.forceDiscardCard; c > 0 {
		data["forceCards"] = []int{c}
	}
	if p := room.discardPlayer; p == nil {
		data["newLoop"] = true
	} else {
		if ans := room.helper.Match(current.GetSortedCards(), p.action); len(ans) == 0 {
			data["pass"] = true
		}
	}
	room.Broadcast("turn", data)
}
