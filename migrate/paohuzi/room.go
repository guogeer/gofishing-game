package paohuzi

import (
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
	"math/rand"
	"third/cardutil"

	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
	"github.com/guogeer/quasar/util"

	// "third/pb"
	// "third/rpc"
	"time"
	// "golang.org/x/net/context"
)

var (
	maxAutoTime    = 2 * time.Second
	maxOperateTime = 4 * time.Second
)

type PaohuziRoom struct {
	*service.Room

	helper *cardutil.PaohuziHelper

	dealer, nextDealer  *PaohuziPlayer
	discardPlayer       *PaohuziPlayer
	expectDiscardPlayer *PaohuziPlayer
	expectPongPlayer    *PaohuziPlayer
	expectKongPlayer    *PaohuziPlayer
	expectChowPlayers   map[int]*PaohuziPlayer
	expectWinPlayers    map[int]*PaohuziPlayer
	winPlayers          []*PaohuziPlayer

	pongPlayer *PaohuziPlayer // 偎
	kongPlayer *PaohuziPlayer // 提龙

	lastCard int

	autoTime  time.Time
	autoTimer *util.Timer
}

func (room *PaohuziRoom) OnEnter(player *service.Player) {
	room.Room.OnEnter(player)

	comer := player.GameAction.(*PaohuziPlayer)
	log.Infof("player %d enter room %d", comer.Id, room.Id)

	// 自动坐下
	seatId := room.GetEmptySeat()
	if player.GetSeatIndex == roomutils.NoSeat && seatId != roomutils.NoSeat {
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

	var seats []*PaohuziUserInfo
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

	if room.Status != service.RoomStatusFree {
		room.Timing()
		comer.Prompt()
	}
}

func (room *PaohuziRoom) Leave(player *service.Player) ErrCode {
	ply := player.GameAction.(*PaohuziPlayer)
	log.Debugf("player %d leave room %d", ply.Id, room.Id)
	return Ok
}

func (room *PaohuziRoom) OnLeave(player *service.Player) {
	room.Room.OnLeave(player)

	p := player.GameAction.(*PaohuziPlayer)
	if room.nextDealer == p {
		room.nextDealer = nil
	}
}

func (room *PaohuziRoom) OnCreate() {
}

func (room *PaohuziRoom) StartGame() {
	room.Room.StartGame()

	room.dealer = room.nextDealer
	room.nextDealer = nil
	// 优先房主
	if room.dealer == nil {
		room.dealer = GetPlayer(room.HostId)
	}
	// 随机
	if room.dealer == nil {
		seatId := rand.Intn(room.NumSeat())
		room.dealer = room.GetPlayer(seatId)
	}
	room.Broadcast("NewDealer", map[string]any{"UId": room.dealer.Id})

	room.StartDealCard()

	// 提龙
	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		for _, c := range cardutil.GetAllCards() {
			if p.cards[c] == 4 {
				meld := room.helper.NewMeld([]int{c, 0, 0, 0})
				room.Broadcast("Kong", map[string]any{"UId": p.Id, "Meld": meld})
				p.cards[c] = 0
				p.melds = append(p.melds, meld)
			}
		}
	}

	room.expectDiscardPlayer = room.dealer
	room.dealer.tryDiscard()
	room.Timing()
}

func (room *PaohuziRoom) OnWin() {
	if len(room.winPlayers) == 0 {
		return
	}
	// 一炮多响时离放炮最近的人胡
	others := make([]*PaohuziPlayer, 0, 8)
	others = append(others, room.winPlayers...)
	for _, other := range room.expectWinPlayers {
		others = append(others, other)
	}

	startId, nearId := roomutils.NoSeat, roomutils.NoSeat
	if other := room.discardPlayer; other != nil {
		startId = other.SeatId
	}
	for _, other := range others {
		if nearId == roomutils.NoSeat ||
			room.distance(startId, nearId) > room.distance(startId, other.SeatId) {
			nearId = other.SeatId
		}
	}

	var near *PaohuziPlayer
	for _, p := range room.winPlayers {
		if p.SeatId == nearId {
			near = p
			break
		}
	}
	if near == nil {
		return
	}
	room.winPlayers = nil
	for _, other := range room.expectWinPlayers {
		other.Pass()
	}
	room.winPlayers = []*PaohuziPlayer{near}

	room.Award()
}

func (room *PaohuziRoom) Award() {
	room.GameOver()
}

func (room *PaohuziRoom) GameOver() {
	guid := util.GUID()
	way := service.GetName()
	unit, _ := config.Int("Room", room.SubId, "Unit")

	room.autoTime = time.Now().Add(room.RestartTime())
	sec := room.GetShowTime(room.autoTime)

	type UserDetail struct {
		Cards []int
		Gold  int64
		Win   *WinOption `json:",omitempty"`
	}
	details := make([]UserDetail, room.NumSeat())
	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		details[i].Cards = p.GetSortedCards()

		kong := 0
		for _, meld := range p.melds {
			if meld.Type == cardutil.PaohuziInvisibleKong {
				if meld.Cards[0]&0xf0 == 0x00 {
					kong += 1
				} else {
					kong += 2
				}
			}
		}
		gold := int64(kong) * unit
		// 提龙
		for k := 0; k < room.NumSeat(); k++ {
			if other := room.GetPlayer(k); p != other {
				details[i].Gold += gold
				details[k].Gold -= gold
			}
		}
	}
	// 有且仅有一个人胡
	if len(room.winPlayers) == 1 {
		winner := room.winPlayers[0]
		winOpt := winner.TryWin()

		winner.winTimes++
		details[winner.SeatId].Win = winOpt
		score := room.helper.Sum(winOpt.Melds) + room.helper.Sum(winOpt.Split)

		quality := 1
		if score == 10 {
			score = 20
		} else if score == 20 {
			score = 30
		}
		if score < 16 {
			quality = 1
		} else if score < 21 {
			quality = 2
		} else {
			quality = (score-21)/3 + 2
		}
		if winner.maxScore < score {
			winner.maxScore = score
		}

		gold := int64(quality) * unit
		for i := 0; i < room.NumSeat(); i++ {
			if p := room.GetPlayer(i); p != winner {
				details[p.SeatId].Gold -= gold
				details[winner.SeatId].Gold += gold
			}
		}
	}

	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		p.AddGold(details[p.SeatId].Gold, guid, way)
		if p.maxGold < p.Gold {
			p.maxGold = p.Gold
		}
	}
	// 积分场最后一局
	room.Broadcast("Award", map[string]any{"Sec": sec, "Seats": details})
	if room.IsTypeScore() && room.ExistTimes+1 == room.LimitTimes {
		type TotalAward struct {
			WinTimes   int
			MaxQuality int
			MaxScore   int
			MaxGold    int64
		}

		seats := make([]TotalAward, 0, room.NumSeat())
		for i := 0; i < room.NumSeat(); i++ {
			p := room.GetPlayer(i)
			seats[i] = TotalAward{
				WinTimes:   p.winTimes,
				MaxQuality: p.maxQuality,
				MaxScore:   p.maxScore,
				MaxGold:    p.maxGold,
			}
		}
		room.Broadcast("TotalAward", map[string]any{"Sec": sec, "Seats": seats})
	}

	room.Room.GameOver()
	room.winPlayers = nil
}

func (room *PaohuziRoom) StartDealCard() {
	// 发牌
	room.autoTime = time.Now().Add(maxAutoTime)
	sec := room.GetShowTime(room.autoTime)

	data := map[string]any{
		"Sec": sec,
	}

	c := room.CardSet().Deal()
	room.dealer.cards[c]++
	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		for k := 0; k < 20; k++ {
			c := room.CardSet().Deal()
			p.cards[c]++
		}
		data["Cards"] = p.GetSortedCards()
		log.Debug("start deal card", p.GetSortedCards())
		p.WriteJSON("StartDealCard", data)
	}
}

func (room *PaohuziRoom) GetPlayer(seatId int) *PaohuziPlayer {
	if seatId < 0 || seatId >= room.NumSeat() {
		return nil
	}
	if p := room.SeatPlayers[seatId]; p != nil {
		return p.GameAction.(*PaohuziPlayer)
	}
	return nil
}

func (room *PaohuziRoom) Turn() {
	// 没人可以出牌，选取出牌的下家当庄家
	if p := room.discardPlayer; p != nil {
		nextId := (p.SeatId + 1) % room.NumSeat()
		next := room.GetPlayer(nextId)
		next.Draw()
	}
	room.Timing()
}

func (room *PaohuziRoom) Timing() {
	current := room.expectDiscardPlayer
	if room.discardPlayer != nil {
		current = room.discardPlayer
	}
	data := map[string]any{
		"UId": current.Id,
		"Sec": room.GetShowTime(room.autoTime),
	}
	room.Broadcast("Timing", data)
}

func (room *PaohuziRoom) distance(from, to int) int {
	return (to - from + room.NumSeat()) % room.NumSeat()
}
