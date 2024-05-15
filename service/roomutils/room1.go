package roomutils

// 2018-01-10
// 房间

import (
	"quasar/utils"
	"time"

	"gofishing-game/internal/cardutils"
	"gofishing-game/internal/gameutils"
	"gofishing-game/service"

	"github.com/guogeer/quasar/config"
)

// 定制的房间
type CustomRoom interface {
	StartGame()
	GameOver()
}

type Room struct {
	Id             int
	SubId          int
	Status         int
	customRoom     CustomRoom
	freeDuration   time.Duration
	playDuration   time.Duration
	countdownTimer *utils.Timer
	cardSet        *cardutils.CardSet // 牌堆

	seatPlayers []*service.Player
	allPlayers  map[int]*service.Player
	chipItemId  int

	// TODO 积分场逻辑。待实现
	hostSeatIndex  int // 房主
	ExistTimes     int
	LimitTimes     int
	TimesByLoop    int
	TimesPerLoop   int
	StartGameUsers []*service.UserInfo
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
		chipItemId:   gameutils.ItemIdGold,
	}
}

func (room *Room) SetChipItem(itemId int) {
	room.chipItemId = itemId
}

func (room *Room) GetChipItem() int {
	return room.chipItemId
}

func (room *Room) Countdown() int64 {
	if room.countdownTimer != nil && room.countdownTimer.IsValid() {
		return room.countdownTimer.Expire().Unix()
	}
	return 0
}

func (room *Room) SetCountdown(f func(), d time.Duration) {
	utils.StopTimer(room.countdownTimer)
	room.countdownTimer = utils.NewTimer(f, d)
}

func (room *Room) Unit() int64 {
	unit, _ := config.Int("room", room.SubId, "cost")
	return unit
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

func (room *Room) FindPlayer(seatIndex int) *service.Player {
	return room.seatPlayers[seatIndex]
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

	utils.StopTimer(room.countdownTimer)
	room.countdownTimer = utils.NewTimer(room.customRoom.GameOver, room.playDuration)
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
	utils.StopTimer(room.countdownTimer)
	room.countdownTimer = utils.NewTimer(room.customRoom.StartGame, room.FreeDuration())
}

func (room *Room) GetEmptySeat() int {
	for i, player := range room.seatPlayers {
		if player == nil {
			return i
		}
	}
	return NoSeat
}

func (room *Room) IsTypeNormal() bool {
	return room.chipItemId == gameutils.ItemIdGold
}

// TODO 待实现
func (room *Room) IsTypeScore() bool {
	return room.chipItemId != gameutils.ItemIdGold
}

// TODO 积分场逻辑。待实现
func (room *Room) CanPlay(opt string) bool {
	return false
}

func (room *Room) NumSeat() int {
	return len(room.seatPlayers)
}

func (room *Room) HostSeatIndex() int {
	return room.hostSeatIndex
}

// TODO 积分场逻辑。待实现
func (room *Room) SetPlay(opt string, args ...any) {
}

// TODO 积分场逻辑。待实现
func (room *Room) SetNoPlay(opt string, args ...any) {
}

func (room *Room) GetPlayValue(prefix string) int {
	return 0
}

// TODO 积分场逻辑。待实现
func (room *Room) SetMainPlay(opt string) {
}

// TODO 自动开局，不需要等待准备
func (room *Room) AutoStart() {
}

// 统计座位上玩家
func (room *Room) NumSeatPlayer() int {
	var num int
	for _, p := range room.seatPlayers {
		if p != nil {
			num += 1
		}
	}
	return num
}
