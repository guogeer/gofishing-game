// 耒阳地区的三公玩法
// 2017-8-9
package sangong

import (
	"gofishing-game/internal/errcode"
	"gofishing-game/internal/gameutils"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"

	"github.com/guogeer/quasar/log"
)

// 玩家信息
type SangongPlayerInfo struct {
	service.UserInfo
	SeatIndex int   `json:"seatIndex,omitempty"`
	Cards     []int `json:"cards,omitempty"`
	// 准备、房主开始游戏，亮牌、看牌
	IsReady        bool `json:"isReady,omitempty"`
	StartGameOrNot bool `json:"startGameOrNot,omitempty"`
	IsDone         bool `json:"isDone,omitempty"`

	Chip     int `json:"chip,omitempty"`
	RobOrNot int `json:"robOrNot,omitempty"`
	CardType int `json:"cardType,omitempty"`
}

type SangongPlayer struct {
	cards    []int // 手牌
	robOrNot int   // 自由抢庄
	chip     int   // 押注
	cardType int

	isDone  bool
	winGold int64
	*service.Player
}

// 已亮牌
func (ply *SangongPlayer) IsDone() bool {
	return ply.isDone
}

func (ply *SangongPlayer) TryEnter() errcode.Error {
	return nil
}

func (ply *SangongPlayer) BeforeEnter() {
}

func (ply *SangongPlayer) AfterEnter() {
}

func (ply *SangongPlayer) BeforeLeave() {
}

func (ply *SangongPlayer) TryLeave() errcode.Error {
	room := ply.Room()
	if room.Status != 0 {
		return errcode.Retry
	}
	return nil
}

func (ply *SangongPlayer) initGame() {
	for i := 0; i < len(ply.cards); i++ {
		ply.cards[i] = 0
	}

	ply.chip = -1
	ply.robOrNot = -1
	ply.isDone = false
}

// 算牌
func (ply *SangongPlayer) Finish() {
	if !roomutils.GetRoomObj(ply.Player).IsReady() {
		return
	}
	if ply.IsDone() {
		return
	}

	room := ply.Room()
	log.Debug("finish", ply.cards, room.Status)
	if room.Status != RoomStatusLook {
		return
	}

	// 庄家最后显示
	typ, _ := room.helper.GetType(ply.cards)
	room.Broadcast("Finish", map[string]any{
		"uid":   ply.Id,
		"Type":  typ,
		"Cards": ply.cards,
	})

	ply.isDone = true
	room.OnFinish()
}

func (ply *SangongPlayer) GameOver() {
	ply.initGame()
}

func (ply *SangongPlayer) Bet(chip int) {
	if !roomutils.GetRoomObj(ply.Player).IsReady() {
		return
	}

	room := ply.Room()
	log.Debugf("player %d bet %d %d status %d step %d", ply.Id, ply.chip, chip, room.Status, ply.Step())
	if ply.chip != -1 {
		return
	}
	if room.Status != RoomStatusBet {
		return
	}

	unit := ply.Step()
	if chip%unit != 0 || chip < 1 || chip/unit > 10 {
		return
	}

	// OK
	ply.chip = chip
	room.Broadcast("Bet", map[string]any{"uid": ply.Id, "Chip": chip})
	room.OnBet()
}

func (ply *SangongPlayer) ChooseDealer(b bool) {
	if !roomutils.GetRoomObj(ply.Player).IsReady() {
		return
	}
	if ply.robOrNot != -1 {
		return
	}

	// OK
	room := ply.Room()

	ply.robOrNot = 0
	if b {
		ply.robOrNot = 1
	}
	room.Broadcast("ChooseDealer", gameutils.MergeError(nil, map[string]any{"uid": ply.Id, "Ans": b}))
	room.OnChooseDealer()
}

func (ply *SangongPlayer) GetUserInfo(self bool) *SangongPlayerInfo {
	info := &SangongPlayerInfo{}
	info.UserInfo = ply.UserInfo
	// info.UId = ply.GetCharObj().Id
	info.SeatIndex = ply.GetSeatIndex()
	info.IsReady = roomutils.GetRoomObj(ply.Player).IsReady()
	info.RobOrNot = ply.robOrNot
	info.Chip = ply.chip

	room := ply.Room()
	if room.Status == RoomStatusLook && roomutils.GetRoomObj(ply.Player).IsReady() {
		info.IsDone = ply.IsDone()
		info.Cards = make([]int, len(ply.cards))
		copy(info.Cards, ply.cards)

		// 已亮牌
		if ply.IsDone() {
			info.IsDone = true
			info.CardType = ply.cardType
		}
	}
	return info
}

func (ply *SangongPlayer) Room() *SangongRoom {
	if room := roomutils.GetRoomObj(ply.Player).CustomRoom(); room != nil {
		return room.(*SangongRoom)
	}
	return nil
}

func (ply *SangongPlayer) Replay(messageId string, i interface{}) {
	switch messageId {
	case "SitDown":
		return
	}
	ply.Player.Replay(messageId, i)
}

func (ply *SangongPlayer) Step() int {
	room := ply.Room()

	unit := room.GetPlayValue(OptChouma)
	return unit
}

func (ply *SangongPlayer) GetSeatIndex() int {
	return roomutils.GetRoomObj(ply.Player).GetSeatIndex()
}
