package roomutils

// 2018-01-10
// 房间

import (
	"time"

	"gofishing-game/internal/cardutils"
	"gofishing-game/service"

	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/util"
)

// 定制的房间
type CustomRoom interface {
	StartGame()
	GameOver()
}

type Room struct {
	SubId          int
	Status         int
	customRoom     CustomRoom
	freeDuration   time.Duration
	playDuration   time.Duration
	countdownTimer *util.Timer
	cardSet        *cardutils.CardSet // 牌堆

	seatPlayers []*service.Player
	allPlayers  map[int]*service.Player
}

func NewRoom(subId int, CustomRoom CustomRoom) *Room {
	var seatNum int
	var freeDuration, playDuration time.Duration
	config.Scan("room", subId, "seatNum,freeDuration,playDuration", &seatNum, &freeDuration, &playDuration)
	return &Room{
		SubId:        subId,
		allPlayers:   make(map[int]*service.Player),
		Status:       0,
		customRoom:   CustomRoom,
		freeDuration: freeDuration,
		playDuration: playDuration,
		cardSet:      cardutils.NewCardSet(),
		seatPlayers:  make([]*service.Player, seatNum),
	}
}

func (room *Room) Countdown() int64 {
	if room.countdownTimer != nil && room.countdownTimer.IsValid() {
		return room.countdownTimer.Expire().Unix()
	}
	return 0
}

func (room *Room) GetAllPlayers() []*service.Player {
	var players []*service.Player
	for _, player := range room.allPlayers {
		players = append(players, player)
	}
	return players
}

func (room *Room) GetSeatPlayers() []*service.Player {
	var seats []*service.Player
	for _, player := range room.seatPlayers {
		if player != nil {
			seats = append(seats, player)
		}
	}
	return seats
}

func (room *Room) CardSet() *cardutils.CardSet {
	return room.cardSet
}

func (room *Room) CustomRoom() CustomRoom {
	return room.customRoom
}

func (room *Room) Broadcast(name string, data any, blacklist ...int) {
	set := make(map[int]bool)
	for _, uid := range blacklist {
		set[uid] = true
	}
	for _, player := range room.allPlayers {
		if _, ok := set[player.Id]; !ok {
			player.WriteJSON(name, data)
		}
	}
}

func (room *Room) StartGame() {
	room.cardSet.Shuffle()
	room.Status = RoomStatusPlaying

	util.StopTimer(room.countdownTimer)
	room.countdownTimer = util.NewTimer(room.customRoom.GameOver, room.playDuration)
}

func (room *Room) FreeDuration() time.Duration {
	return room.freeDuration
}

func (room *Room) SetFreeDuration(d time.Duration) {
	room.freeDuration = d
}

func (room *Room) PlayDuration() time.Duration {
	return room.playDuration
}

func (room *Room) SetPlayDuration(d time.Duration) {
	room.playDuration = d
}

func (room *Room) GameOver() {
	room.Status = 0
	room.CardSet().Shuffle()
	util.StopTimer(room.countdownTimer)
	room.countdownTimer = util.NewTimer(room.customRoom.StartGame, room.FreeDuration())
}

func (room *Room) GetEmptySeat() int {
	for i, player := range room.seatPlayers {
		if player == nil {
			return i
		}
	}
	return NoSeat
}
