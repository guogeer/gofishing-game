package roomutil

// 2018-01-10
// 房间

import (
	"time"

	"gofishing-game/internal/cardutil"
	"gofishing-game/service"

	"github.com/guogeer/quasar/config"
)

// 定制的房间
type CustomRoom interface {
	StartGame()
	GameOver()
}

type Room struct {
	SubId       int
	allPlayers  map[int]*service.Player
	Status      int
	customRoom  CustomRoom
	restartTime time.Duration
	cardSet     *cardutil.CardSet // 牌堆
}

func NewRoom(subId int, CustomRoom CustomRoom) *Room {
	return &Room{
		SubId:       subId,
		allPlayers:  make(map[int]*service.Player),
		Status:      RoomStatusFree,
		customRoom:  CustomRoom,
		restartTime: -1,
		cardSet:     cardutil.NewCardSet(),
	}
}

func (room *Room) GetAllPlayers() []*service.Player {
	var players []*service.Player
	for _, player := range room.allPlayers {
		players = append(players, player)
	}
	return players
}

// 房间倒计时
func (room *Room) GetShowTime(deadline time.Time) int {
	return GetShowTime(deadline)
}

func (room *Room) CardSet() *cardutil.CardSet {
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
	room.restartTime = -1
}

func (room *Room) RestartTime() time.Duration {
	restartTime := room.restartTime
	if restartTime >= 0 {
		return restartTime
	}
	d, ok := config.Duration("Room", room.SubId, "FreeTime")
	if ok {
		return d
	}

	return restartTime
}

func (room *Room) SetRestartTime(d time.Duration) {
	room.restartTime = d
}

func (room *Room) GameOver() {
	room.Status = RoomStatusFree
	room.CardSet().Shuffle()
}
