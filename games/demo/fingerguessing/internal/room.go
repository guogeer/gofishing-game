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
	Chip      int64  `json:"chip"`
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
		user.Chip = p.BagObj().NumItem(gameutils.ItemIdGold)
		info.SeatPlayers = append(info.SeatPlayers, user)
	}

	return info
}

func (room *fingerGuessingRoom) StartGame() {
	room.Room.StartGame()

	room.Broadcast("startGame", cmd.M{"countdown": room.Countdown()})
}

type userResult struct {
	WinChip   int64  `json:"winChip"`
	SeatIndex int    `json:"seatIndex"`
	Gesture   string `json:"gesture"`
	Cmp       int    `json:"cmp"`
}

func (room *fingerGuessingRoom) GameOver() {
	gesture := fingerGuessingGuestures[rand.Intn(len(fingerGuessingGuestures))]
	users := make([]userResult, 0, 4)
	for _, seatPlayer := range room.GetSeatPlayers() {
		p := seatPlayer.GameAction.(*fingerGuessingPlayer)
		if p.gesture != "" {
			winChip, cmp := p.GameOver(gesture)
			roomObj := roomutils.GetRoomObj(seatPlayer)
			users = append(users, userResult{WinChip: winChip, Cmp: cmp, SeatIndex: roomObj.GetSeatIndex(), Gesture: p.gesture})
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
