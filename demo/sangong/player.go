// 耒阳地区的三公玩法
// 2017-8-9
package sangong

import (
	"gofishing-game/service"
	. "third/errcode"

	"github.com/guogeer/quasar/log"
)

// 玩家信息
type SangongPlayerInfo struct {
	service.UserInfo
	SeatId int
	Cards  []int `json:",omitempty"`
	// 准备、房主开始游戏，亮牌、看牌
	IsReady, StartGameOrNot, IsDone bool `json:",omitempty"`

	Chip     int
	RobOrNot int
	CardType int
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

func (ply *SangongPlayer) TryLeave() ErrCode {
	room := ply.Room()
	if room.Status != service.RoomStatusFree {
		return Retry
	}
	return Ok
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
	if ply.RoomObj.IsReady() == false {
		return
	}
	if ply.IsDone() == true {
		return
	}

	room := ply.Room()
	log.Debug("finish", ply.cards, room.Status)
	if room.Status != service.RoomStatusLook {
		return
	}

	// 庄家最后显示
	typ, _ := room.helper.GetType(ply.cards)
	room.Broadcast("Finish", map[string]any{
		"UId":   ply.Id,
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
	if ply.RoomObj.IsReady() == false {
		return
	}

	room := ply.Room()
	log.Debugf("player %d bet %d %d status %d step %d", ply.Id, ply.chip, chip, room.Status, ply.Step())
	if ply.chip != -1 {
		return
	}
	if room.Status != service.RoomStatusBet {
		return
	}

	unit := ply.Step()
	if chip%unit != 0 || chip < 1 || chip/unit > 10 {
		return
	}

	// OK
	ply.chip = chip
	room.Broadcast("Bet", map[string]any{"UId": ply.Id, "Chip": chip})
	room.OnBet()
}

func (ply *SangongPlayer) ChooseDealer(b bool) {
	if ply.RoomObj.IsReady() == false {
		return
	}
	if ply.robOrNot != -1 {
		return
	}

	// OK
	room := ply.Room()

	ply.robOrNot = 0
	if b == true {
		ply.robOrNot = 1
	}
	room.Broadcast("ChooseDealer", map[string]any{"Code": Ok, "UId": ply.Id, "Ans": b})
	room.OnChooseDealer()
}

func (ply *SangongPlayer) GetUserInfo(self bool) *SangongPlayerInfo {
	info := &SangongPlayerInfo{}
	info.UserInfo = ply.UserInfo
	// info.UId = ply.GetCharObj().Id
	info.SeatId = ply.SeatId
	info.IsReady = ply.RoomObj.IsReady()
	info.RobOrNot = ply.robOrNot
	info.Chip = ply.chip

	room := ply.Room()
	if room.Status == service.RoomStatusLook && ply.RoomObj.IsReady() {
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

// 坐下
func (ply *SangongPlayer) SitDown(seatId int) {
	room := ply.Room()
	if code := ply.RoomObj.SitDown(seatId); code != Ok {
		return
	}
	// OK
	info := ply.GetUserInfo(false)
	room.Broadcast("SitDown", map[string]any{"Code": Ok, "Info": info})
}

func (ply *SangongPlayer) Room() *SangongRoom {
	if room := ply.RoomObj.CardRoom(); room != nil {
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

	unit := 1
	if room.CanPlay(OptChouma2) {
		unit = 2
	} else if room.CanPlay(OptChouma3) {
		unit = 3
	} else if room.CanPlay(OptChouma5) {
		unit = 5
	} else if room.CanPlay(OptChouma8) {
		unit = 8
	} else if room.CanPlay(OptChouma10) {
		unit = 10
	} else if room.CanPlay(OptChouma20) {
		unit = 20
	}
	return unit
}
