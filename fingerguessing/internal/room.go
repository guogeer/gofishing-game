package internal

import (
	"gofishing-game/service/roomutils"
	"math/rand"
	"quasar/util"
	"time"

	"github.com/guogeer/quasar/cmd"
)

const gameDuration = time.Second * 10

type clientFingerGuessingUser struct {
	Gesture   string `json:"gesture,omitempty"`
	SeatIndex int    `json:"seatIndex,omitempty"`
}

type clientFingerGuessingRoom struct {
	SeatPlayers []clientFingerGuessingUser `json:"seatPlayers,omitempty"`
	OverTs      int64                      `json:"overTs,omitempty"`
}

type fingerGuessingRoom struct {
	*roomutils.Room

	overTimer *util.Timer
}

func (room *fingerGuessingRoom) GetClientInfo() clientFingerGuessingRoom {
	info := clientFingerGuessingRoom{
		SeatPlayers: []clientFingerGuessingUser{},
	}
	if room.overTimer.IsValid() {
		info.OverTs = room.overTimer.Expire().Unix()
	}
	for _, seatPlayer := range room.GetSeatPlayers() {
		p := seatPlayer.GameAction.(*fingerGuessingPlayer)
		seatIndex := roomutils.GetRoomObj(seatPlayer).GetSeat()
		info.SeatPlayers = append(info.SeatPlayers, clientFingerGuessingUser{Gesture: p.gesture, SeatIndex: seatIndex})
	}

	return info
}

func (room *fingerGuessingRoom) StartGame() {
	room.Room.StartGame()
	util.StopTimer(room.overTimer)

	room.overTimer = util.NewTimer(room.GameOver, gameDuration)
	room.Broadcast("startGame", cmd.M{"ts": room.overTimer.Expire().Unix()})
}

type seatResult struct {
	WinGold int64  `json:"winGold,omitempty"`
	Seat    int    `json:"seat,omitempty"`
	Gesture string `json:"gesture,omitempty"`
	Cmp     int    `json:"cmp,omitempty"`
}

func (room *fingerGuessingRoom) GameOver() {
	util.StopTimer(room.overTimer)

	gesture := fingerGuessingGuestures[rand.Intn(len(fingerGuessingGuestures))]
	seats := make([]seatResult, 0, 4)
	for _, seatPlayer := range room.GetSeatPlayers() {
		p := seatPlayer.GameAction.(*fingerGuessingPlayer)
		if p.gesture != "" {
			winGold, cmp := p.GameOver(gesture)
			roomObj := roomutils.GetRoomObj(seatPlayer)
			seats = append(seats, seatResult{WinGold: winGold, Cmp: cmp, Seat: roomObj.GetSeat(), Gesture: p.gesture})
		}
	}

	room.Broadcast("GameOver", cmd.M{
		"gesture": gesture,
		"Seats":   seats,
	})
	room.Room.GameOver()
}
