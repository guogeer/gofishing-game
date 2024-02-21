package internal

import (
	"gofishing-game/internal/gameutils"
	"gofishing-game/service/roomutils"
	"math/rand"

	"github.com/guogeer/quasar/cmd"
)

type clientFingerGuessingUser struct {
	Uid       int    `json:"uid"`
	Gesture   string `json:"gesture"`
	SeatIndex int    `json:"seatIndex"`
	Gold      int64  `json:"gold"`
	Nickname  string `json:"nickname"`
}

type clientFingerGuessingRoom struct {
	Status      int                        `json:"status"`
	SeatPlayers []clientFingerGuessingUser `json:"seatPlayers"`
	Countdown   int64                      `json:"countdown"`
}

type fingerGuessingRoom struct {
	*roomutils.Room
}

func (room *fingerGuessingRoom) GetClientInfo() clientFingerGuessingRoom {
	info := clientFingerGuessingRoom{
		Status:      room.Status,
		SeatPlayers: []clientFingerGuessingUser{},
	}
	info.Countdown = room.Countdown()
	for _, seatPlayer := range room.GetSeatPlayers() {
		p := seatPlayer.GameAction.(*fingerGuessingPlayer)
		seatIndex := roomutils.GetRoomObj(seatPlayer).GetSeatIndex()
		user := clientFingerGuessingUser{Uid: p.Id, Gesture: p.gesture, SeatIndex: seatIndex, Nickname: p.Nickname}
		user.Gold = p.BagObj().NumItem(gameutils.ItemIdGold)
		info.SeatPlayers = append(info.SeatPlayers, user)
	}

	return info
}

func (room *fingerGuessingRoom) StartGame() {
	room.Room.StartGame()

	room.Broadcast("startGame", cmd.M{"countdown": room.Countdown()})
}

type userResult struct {
	WinGold int64  `json:"winGold"`
	Seat    int    `json:"seat"`
	Gesture string `json:"gesture"`
	Cmp     int    `json:"cmp"`
}

func (room *fingerGuessingRoom) GameOver() {
	gesture := fingerGuessingGuestures[rand.Intn(len(fingerGuessingGuestures))]
	users := make([]userResult, 0, 4)
	for _, seatPlayer := range room.GetSeatPlayers() {
		p := seatPlayer.GameAction.(*fingerGuessingPlayer)
		if p.gesture != "" {
			winGold, cmp := p.GameOver(gesture)
			roomObj := roomutils.GetRoomObj(seatPlayer)
			users = append(users, userResult{WinGold: winGold, Cmp: cmp, Seat: roomObj.GetSeatIndex(), Gesture: p.gesture})
		}
	}

	for _, seatPlayer := range room.GetSeatPlayers() {
		p := seatPlayer.GameAction.(*fingerGuessingPlayer)
		p.gesture = ""
	}

	room.Room.GameOver()
	room.Broadcast("gameOver", cmd.M{
		"gesture":   gesture,
		"result":    users,
		"countdown": room.Countdown(),
	})
}
