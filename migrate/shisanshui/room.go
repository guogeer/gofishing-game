package shisanshui

import (
	"gofishing-game/internal/errcode"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
	"math/rand"

	"github.com/guogeer/quasar/log"
	"github.com/guogeer/quasar/util"

	// "third/pb"
	// "third/rpc"
	"time"
	// "golang.org/x/net/context"
)

const (
	_                    = iota
	OptDaxiaowang        // 有大小王
	OptXianghubipai      // 相互比牌
	OptFangzhudangzhuang // 房主当庄
	OptLipai50s          // 50s理牌
	OptLipai80s          // 80s理牌
	OptLipai70s          // 70s理牌
)

var (
	maxAutoTime = 120 * time.Second
)

type ShisanshuiRoom struct {
	*roomutils.Room

	helper *cardutils.ShisanshuiHelper

	autoTime           time.Time
	dealer, nextDealer *ShisanshuiPlayer
}

func (room *ShisanshuiRoom) OnEnter(player *service.Player) {
	room.Room.OnEnter(player)

	comer := player.GameAction.(*ShisanshuiPlayer)
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

	var seats []*ShisanshuiUserInfo
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
}

func (room *ShisanshuiRoom) Leave(player *service.Player) errcode.Error {
	p := player.GameAction.(*ShisanshuiPlayer)
	log.Debugf("player %d leave room %d", p.Id, room.Id)
	return nil
}

func (room *ShisanshuiRoom) OnLeave(player *service.Player) {
	room.Room.OnLeave(player)

	p := player.GameAction.(*ShisanshuiPlayer)
	if room.dealer == p {
		room.dealer = nil
	}
	if room.nextDealer == p {
		room.nextDealer = nil
	}
}

func (room *ShisanshuiRoom) OnCreate() {
	// room.CardSet().Recover(cardutils.GetAllCards()...)
	room.Room.OnCreate()
	room.CardSet().Remove(0xf0, 0xf1)
	if room.CanPlay(OptDaxiaowang) {
		room.CardSet().Recover(0xf0, 0xf1)
	}
}

func (room *ShisanshuiRoom) StartGame() {
	room.Room.StartGame()

	var counter int
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil && roomutils.GetRoomObj(p.Player).IsReady() {
			counter++
			p.initGame()
		}
	}

	room.CardSet().ClearExtraCards()
	if counter == 5 {
		for c := 0x32; c <= 0x3e; c++ {
			room.CardSet().AddExtraCards(c)
		}
	}
	room.CardSet().Shuffle()

	if room.CanPlay(OptXianghubipai) {
		room.nextDealer = room.GetPlayer(room.HostSeatIndex())
	}
	// 选庄家
	if room.nextDealer != nil {
		room.dealer = room.nextDealer
	}

	room.nextDealer = nil
	// 房主当庄
	if host := room.GetPlayer(room.HostSeatIndex()); room.dealer == nil && host != nil && host.Room() == room {
		room.dealer = host
	}
	// 随机
	if room.dealer == nil {
		var seats []int
		for i := 0; i < room.NumSeat(); i++ {
			if p := room.GetPlayer(i); p != nil && roomutils.GetRoomObj(p.Player).IsReady() {
				seats = append(seats, i)
			}
		}
		seatId := seats[rand.Intn(len(seats))]
		room.dealer = room.GetPlayer(seatId)
	}
	if room.CanPlay(OptXianghubipai) {
		room.dealer = nil
	}
	if room.dealer != nil {
		room.Broadcast("NewDealer", map[string]any{"UId": room.dealer.Id})
	}
	room.StartDealCard()
}

func (room *ShisanshuiRoom) OnSplitCards() {
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil && roomutils.GetRoomObj(p.Player).IsReady() && p.splitCards[0] == 0 {
			return
		}
	}
	room.Award()
}

func (room *ShisanshuiRoom) Award() {
	guid := util.GUID()
	way := service.GetServerName()
	// unit, _ := config.Int("Room", room.SubId, "Unit")
	unit := room.Unit()

	// room.Status = 0
	room.autoTime = time.Now().Add(room.RestartTime())
	sec := room.GetShowTime(room.autoTime)

	type UserAward struct {
		Gold                 int64
		Total                int
		AllParts, ExtraParts [3]int
		Cards                []int
		Parts                []int `json:",omitempty"`
		SpecialType          int   `json:",omitempty"`
		Daqiang              int   `json:",omitempty"`
		Quanleida            int   `json:",omitempty"`
		Teshupaixing         int   `json:",omitempty"`
		DaqiangSeats         int   `json:",omitempty"`
	}

	readyPlayerNum := 0
	users := make([]UserAward, room.NumSeat())
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil && roomutils.GetRoomObj(p.Player).IsReady() {
			readyPlayerNum++
		}
	}
	// 房主坐庄
	dealer := room.dealer
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil && roomutils.GetRoomObj(p.Player).IsReady() {
			_, typ := room.helper.GetSpecialType(p.cards)
			if typ == 0 {
				typ1, _ := room.helper.GetPartType(p.splitCards[:3])
				typ2, _ := room.helper.GetPartType(p.splitCards[3:8])
				typ3, _ := room.helper.GetPartType(p.splitCards[8:])
				users[i].Parts = []int{typ1, typ2, typ3}
			}
			users[i].Cards = p.splitCards
			users[i].SpecialType = typ

			quanleida := (readyPlayerNum > 3) && (typ == 0)
			tempUsers := make([]UserAward, room.NumSeat())
			for k := 0; k < room.NumSeat(); k++ {
				other := room.GetPlayer(k)
				if dealer != nil && p != dealer && other != dealer {
					continue
				}
				if other != nil && p != other && other.RoomObj.IsReady() {
					if typ != 0 && room.helper.Less(other.splitCards, p.splitCards) {
						score := 0
						switch typ {
						case cardutils.ShisanshuiZhizunqinglong:
							score = 108
						case cardutils.ShisanshuiYitiaolong:
							score = 36
						case cardutils.ShisanshuiShierhuangzu:
							score = 24
						case cardutils.ShisanshuiSantonghuashun:
							score = 22
						case cardutils.ShisanshuiSanfentianxia:
							score = 20
						case cardutils.ShisanshuiQuanda:
							score = 15
						case cardutils.ShisanshuiQuanxiao:
							score = 12
						case cardutils.ShisanshuiCouyise:
							score = 10
						case cardutils.ShisanshuiSitaosantiao:
							score = 6
						case cardutils.ShisanshuiWuduisantiao:
							score = 5
						case cardutils.ShisanshuiLiuduiban:
							score = 5
						case cardutils.ShisanshuiSanshunzi:
							score = 5
						case cardutils.ShisanshuiSantonghua:
							score = 5
						}
						tempUsers[k].Teshupaixing += score
					}
					wins := 0
					_, otherType := room.helper.GetSpecialType(other.cards)
					if typ == 0 && otherType == 0 {
						if room.helper.LessPart(other.splitCards[:3], p.splitCards[:3]) {
							wins++

							score := 1
							switch t, _ := room.helper.GetPartType(p.splitCards[:3]); t {
							case cardutils.ShisanshuiSantiao:
								score = 3
							}
							tempUsers[k].AllParts[0] = score
							tempUsers[k].ExtraParts[0] = score - 1
						}
						if room.helper.LessPart(other.splitCards[3:8], p.splitCards[3:8]) {
							wins++

							score := 1
							switch t, _ := room.helper.GetPartType(p.splitCards[3:8]); t {
							case cardutils.ShisanshuiHulu:
								score = 2
							case cardutils.ShisanshuiTiezhi:
								score = 8
							case cardutils.ShisanshuiTonghuashun:
								score = 10
							case cardutils.ShisanshuiWutong:
								score = 15
							}
							tempUsers[k].AllParts[1] = score
							tempUsers[k].ExtraParts[1] = score - 1
						}
						if room.helper.LessPart(other.splitCards[8:], p.splitCards[8:]) {
							wins++

							score := 1
							switch t, _ := room.helper.GetPartType(p.splitCards[8:]); t {
							case cardutils.ShisanshuiTiezhi:
								score = 4
							case cardutils.ShisanshuiTonghuashun:
								score = 5
							case cardutils.ShisanshuiWutong:
								score = 7
							}
							tempUsers[k].AllParts[2] = score
							tempUsers[k].ExtraParts[2] = score - 1
						}
					}
					if wins != 3 {
						quanleida = false
					}
					if wins == 3 {
						sum := tempUsers[k].Daqiang
						for _, score := range tempUsers[k].AllParts {
							sum += score
						}
						tempUsers[k].Daqiang += sum
					}
				}
			}
			if quanleida {
				for k := range tempUsers {
					sum := tempUsers[k].Daqiang
					for _, score := range tempUsers[k].AllParts {
						sum += score
					}
					tempUsers[k].Quanleida += sum
				}
			}
			for x := 0; x < room.NumSeat(); x++ {
				if i != x {
					for y := range users[x].AllParts {
						users[x].AllParts[y] -= tempUsers[x].AllParts[y]
						users[i].AllParts[y] += tempUsers[x].AllParts[y]

						users[x].ExtraParts[y] -= tempUsers[x].ExtraParts[y]
						users[i].ExtraParts[y] += tempUsers[x].ExtraParts[y]
					}
					users[x].Daqiang -= tempUsers[x].Daqiang
					users[x].Quanleida -= tempUsers[x].Quanleida
					users[x].Teshupaixing -= tempUsers[x].Teshupaixing

					users[i].Daqiang += tempUsers[x].Daqiang
					users[i].Quanleida += tempUsers[x].Quanleida
					users[i].Teshupaixing += tempUsers[x].Teshupaixing
					if tempUsers[x].Daqiang > 0 {
						users[i].DaqiangSeats |= 1 << uint(x)
					}
				}
			}
		}
	}

	var userWinGold, userLoseGold int64
	for i := 0; i < room.NumSeat(); i++ {
		for k := range users[i].AllParts {
			users[i].Total += users[i].AllParts[k]
		}
		users[i].Total += users[i].Daqiang
		users[i].Total += users[i].Quanleida
		users[i].Total += users[i].Teshupaixing
		users[i].Gold = int64(users[i].Total) * unit

		if p := room.GetPlayer(i); p != nil && roomutils.GetRoomObj(p.Player).IsReady() {
			if !room.IsTypeScore() && users[i].Gold+p.Gold < 0 {
				users[i].Gold = -p.Gold
			}
			if users[i].Gold > 0 {
				userWinGold += users[i].Gold
			}
			if users[i].Gold < 0 {
				userLoseGold += -users[i].Gold
			}
		}
	}

	for i := 0; i < room.NumSeat(); i++ {
		g := users[i].Gold
		if g > 0 && userWinGold > 0 {
			g = int64(float64(g) / float64(userWinGold) * float64(userLoseGold))
		}
		if p := room.GetPlayer(i); p != nil {
			p.AddAliasGold(g, guid, way)
		}
		users[i].Gold = g
	}

	room.Broadcast("Award", map[string]any{"Users": users, "Sec": sec})
	room.GameOver()
}

func (room *ShisanshuiRoom) GameOver() {
	if room.IsTypeScore() && room.ExistTimes+1 == room.LimitTimes {
		room.Broadcast("TotalAward", map[string]any{})
	}
	room.Room.GameOver()
}

func (room *ShisanshuiRoom) splitCardsTime() time.Duration {
	t := maxAutoTime
	if room.CanPlay(OptLipai50s) {
		t = 50 * time.Second
	} else if room.CanPlay(OptLipai80s) {
		t = 80 * time.Second
	} else if room.CanPlay(OptLipai70s) {
		t = 70 * time.Second
	}
	return t
}

func (room *ShisanshuiRoom) StartDealCard() {
	// 发牌
	autoTime := room.splitCardsTime()
	room.autoTime = time.Now().Add(autoTime)
	sec := room.GetShowTime(room.autoTime)

	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil && roomutils.GetRoomObj(p.Player).IsReady() {
			for k := range p.cards {
				c := room.CardSet().Deal()
				p.cards[k] = c
			}
			log.Debug("start deal card", p.cards)
		}
	}
	data := map[string]any{
		"Sec": sec,
	}
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil && roomutils.GetRoomObj(p.Player).IsReady() {
			p.resultSet = room.helper.Match(p.cards)

			data["Cards"] = p.cards
			data["ResultSet"] = p.resultSet
			p.WriteJSON("StartDealCard", data)
			p.Timeout(func() { p.SplitCards(p.resultSet[0].Cards[:]) }, autoTime)
		}
	}
}

func (room *ShisanshuiRoom) GetPlayer(seatId int) *ShisanshuiPlayer {
	if seatId < 0 || seatId >= room.NumSeat() {
		return nil
	}
	if p := room.SeatPlayers[seatId]; p != nil {
		return p.GameAction.(*ShisanshuiPlayer)
	}
	return nil
}
