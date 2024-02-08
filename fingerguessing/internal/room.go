package internal

import (
	"gofishing-game/internal/gameutils"
	"gofishing-game/service/roomutils"
	"math/rand"
	"time"

	"github.com/guogeer/quasar/cmd"
)

const gameDuration = time.Second * 10

type clientFingerGuessingUser struct {
	Uid       int              `json:"uid,omitempty"`
	Gesture   string           `json:"gesture,omitempty"`
	SeatIndex int              `json:"seatIndex,omitempty"`
	Items     []gameutils.Item `json:"items,omitempty"`
	Nickname  string           `json:"nickname,omitempty"`
}

type clientFingerGuessingRoom struct {
	Status      int                        `json:"status,omitempty"`
	SeatPlayers []clientFingerGuessingUser `json:"seatPlayers,omitempty"`
	Coutdown    int64                      `json:"coutdown,omitempty"`
}

type fingerGuessingRoom struct {
	*roomutils.Room
}

func (room *fingerGuessingRoom) GetClientInfo() clientFingerGuessingRoom {
	info := clientFingerGuessingRoom{
		SeatPlayers: []clientFingerGuessingUser{},
	}
	info.Coutdown = room.Countdown()
	for _, seatPlayer := range room.GetSeatPlayers() {
		p := seatPlayer.GameAction.(*fingerGuessingPlayer)
		seatIndex := roomutils.GetRoomObj(seatPlayer).GetSeatIndex()
		user := clientFingerGuessingUser{Gesture: p.gesture, SeatIndex: seatIndex, Nickname: p.Nickname}
		user.Items = append(user.Items, p.BagObj().GetItem(gameutils.ItemIdGold))
		info.SeatPlayers = append(info.SeatPlayers, user)
	}

	return info
}

func (room *fingerGuessingRoom) StartGame() {
	room.Room.StartGame()

	room.Broadcast("startGame", cmd.M{"countdown": room.Countdown()})
}

type userResult struct {
	WinGold int64  `json:"winGold,omitempty"`
	Seat    int    `json:"seat,omitempty"`
	Gesture string `json:"gesture,omitempty"`
	Cmp     int    `json:"cmp,omitempty"`
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

	room.Room.GameOver()
	room.Broadcast("GameOver", cmd.M{
		"gesture":  gesture,
		"result":   users,
		"coundown": room.Countdown(),
	})
}
