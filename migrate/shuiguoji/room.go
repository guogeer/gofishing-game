package shuiguoji

import (
	"container/list"
	"gofishing-game/internal/errcode"
	"gofishing-game/service"

	"github.com/guogeer/quasar/log"
)

type PrizePoolUser = service.RankUserInfo

type PrizePoolRank struct {
	top   *PrizePoolUser
	top20 *list.List
}

var (
	prizePoolRank = &PrizePoolRank{top20: list.New()}
)

func (rank *PrizePoolRank) getAverage() int64 {
	sum := int64(0)
	for e := rank.top20.Front(); e != nil; e = e.Next() {
		sum += e.Value.(*PrizePoolUser).Prize
	}
	if rank.top20.Len() == 0 {
		return 0
	}
	return sum / int64(rank.top20.Len())
}

func (rank *PrizePoolRank) update(user *PrizePoolUser) {
	if user == nil || user.Prize == 0 {
		return
	}
	// v1.0 超过最近20名平均值更新
	// v1.1 简化为直接更新最新的
	if true || rank.top == nil || user.Prize >= rank.getAverage() {
		rank.top = user
		service.ServiceConfig().Set("sgj_pp_top_user", int64(user.Id))
		service.ServiceConfig().Set("sgj_pp_top_gold", user.Prize)
	}
	rank.top20.PushBack(user)
	if rank.top20.Len() > 20 {
		front := rank.top20.Front()
		rank.top20.Remove(front)
	}
}

type shuiguojiRoom struct {
	*service.Room

	lastPrizePool int64
}

func (room *shuiguojiRoom) OnCreate() {
	room.Room.OnCreate()
	room.StartGame()
}

func (room *shuiguojiRoom) OnEnter(player *service.Player) {
	room.Room.OnEnter(player)

	comer := player.GameAction.(*shuiguojiPlayer)
	log.Infof("player %d enter room %d", comer.Id, room.Id)

	// 自动坐下
	seatId := room.GetEmptySeat()
	if comer.SeatId == roomutils.NoSeat && seatId != roomutils.NoSeat {
		comer.RoomObj.SitDown(seatId)

		info := comer.GetUserInfo(0)
		room.Broadcast("SitDown", map[string]any{"Code": Ok, "Info": info}, comer.Id)
	}

	// 玩家重连
	prizePool := room.GetPrizePool().Add(0)
	data := map[string]any{
		"Status":    room.Status,
		"SubId":     room.GetSubId(),
		"PrizePool": prizePool,
	}

	var seats []*shuiguojiUserInfo
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil {
			info := p.GetUserInfo(comer.Id)
			seats = append(seats, info)
		}
	}
	data["SeatPlayers"] = seats
	comer.WriteJSON("GetRoomInfo", data)
}

func (room *shuiguojiRoom) GameOver() {
	room.Room.GameOver()
}

func (room *shuiguojiRoom) GetPlayer(seatId int) *shuiguojiPlayer {
	if seatId < 0 || seatId >= room.NumSeat() {
		return nil
	}
	if p := room.SeatPlayers[seatId]; p != nil {
		return p.GameAction.(*shuiguojiPlayer)
	}
	return nil
}

func (room *shuiguojiRoom) Sync() {
	isUpdate := false
	gold := room.GetPrizePool().Add(0)
	if room.lastPrizePool != gold {
		isUpdate = true
	}
	room.lastPrizePool = gold
	if isUpdate == true {
		room.Broadcast("Sync", map[string]any{"PrizePool": gold})
	}
}
