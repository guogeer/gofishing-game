package internal

import (
	"gofishing-game/games/migrate/internal/cardrule"
	"gofishing-game/internal/cardutils"
	"gofishing-game/internal/errcode"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
	"time"

	"github.com/guogeer/quasar/log"
	"github.com/guogeer/quasar/utils"
)

// 玩家信息
type DoudizhuUserInfo struct {
	service.UserInfo

	SeatIndex  int   `json:"seatIndex"`
	CardNum    int   `json:"cardNum"`
	BoomTimes  int   `json:"boomTimes"`
	Cards      []int `json:"cards"`
	Action     []int `json:"action"`  // 最近打出去的牌
	IsReady    bool  `json:"isReady"` // 准备
	Jiaodizhu  int   `json:"jiaodizhu"`
	Jiaofen    int   `json:"jiaofen"`
	Choice     int   `json:"choice"`
	IsAutoPlay bool  `json:"isAutoPlay"`
}

type DoudizhuPlayer struct {
	*service.Player

	cards        []int // 手牌
	action       []int // 本轮打出去的牌
	isAutoPlay   bool
	isSystemAuto bool // 系统自动出牌
	boomTimes    int
	discardTimes int

	winTimes       int   // 赢的局数
	maxWinChip     int64 // 最大赢的金币
	totalBoomTimes int   // 总的炸弹数

	jiaodizhu  int // -1 没答复；0、不叫；1、叫
	jiaofen    int // -1 没答复；0、不叫；
	qiangdizhu int // -1 没答复；0、不叫；1、叫；2、再抢

	operateTimer *utils.Timer
}

func (ply *DoudizhuPlayer) TryLeave() errcode.Error {
	room := roomutils.GetRoomObj(ply.Player).Room()
	if room.Status != 0 {
		return errcode.Retry
	}
	return nil
}

func (ply *DoudizhuPlayer) initGame() {
	for i := 0; i < len(ply.cards); i++ {
		ply.cards[i] = 0
	}

	ply.action = nil
	ply.discardTimes = 0
	ply.boomTimes = 0
	ply.isAutoPlay = false
	ply.isSystemAuto = false
	ply.jiaofen = -1
	ply.jiaodizhu = -1
	ply.qiangdizhu = -1
}

func (ply *DoudizhuPlayer) TryEnter() errcode.Error {
	return nil
}

func (ply *DoudizhuPlayer) BeforeEnter() {
}

func (ply *DoudizhuPlayer) AfterEnter() {
}

func (ply *DoudizhuPlayer) BeforeLeave() {
}

func (ply *DoudizhuPlayer) GameOver() {
	ply.initGame()
}

func (ply *DoudizhuPlayer) GetUserInfo(self bool) *DoudizhuUserInfo {
	roomObj := roomutils.GetRoomObj(ply.Player)

	info := &DoudizhuUserInfo{}
	info.UserInfo = ply.UserInfo
	// info.UId = ply.GetCharObj().Id
	info.SeatIndex = roomObj.GetSeatIndex()
	info.IsReady = roomObj.IsReady()
	info.Cards = ply.GetSortedCards()
	info.BoomTimes = ply.boomTimes
	info.Action = ply.action
	info.Jiaodizhu = ply.jiaodizhu
	info.Jiaofen = ply.jiaofen
	info.Choice = ply.qiangdizhu
	info.IsAutoPlay = ply.isAutoPlay
	return info
}

func (ply *DoudizhuPlayer) GetSortedCards() []int {
	room := ply.Room()
	helper := room.helper
	cards := make([]int, 0, 16)
	for v := 0; v <= helper.Value(0xf1); v++ {
		for _, c := range cardutils.GetCardSystem(roomutils.GetServerName(room.SubId)).GetAllCards() {
			if helper.Value(c) == v {
				for k := 0; k < ply.cards[c]; k++ {
					cards = append(cards, c)
				}
			}
		}
	}
	return cards
}

// 自动出牌或过
func (ply *DoudizhuPlayer) AutoPlay() {
	room := ply.Room()
	if room.IsTypeNormal() && !ply.isAutoPlay && !ply.isSystemAuto {
		ply.SetAutoPlay(1)
	}
	ply.isSystemAuto = false

	isAuto := ply.isAutoPlay
	cards := ply.GetSortedCards()

	action := make([]int, 0, 4)
	if other := room.discardPlayer; other == nil {
		// 没有其他玩家出牌
		// 最后一轮，玩家自动出牌
		action = cards[:1]
	} else {
		ans := room.helper.Match(cards, other.action)
		if len(ans) > 0 {
			action = ans
		}
	}

	// log.Debug("time out", isAuto)
	if !isAuto && room.IsTypeScore() {
		return
	}

	if len(action) == 0 {
		ply.Pass()
	} else {
		ply.Discard(action)
	}
}

func (ply *DoudizhuPlayer) SetAutoPlay(t int) {
	room := ply.Room()
	if ply.isAutoPlay == (t != 0) {
		return
	}
	if room.Status == 0 {
		return
	}

	isAutoPlay := ply.isAutoPlay
	room.Broadcast("autoPlay", map[string]any{"code": errcode.CodeOk, "type": t, "uid": ply.Id})

	ply.isAutoPlay = (t != 0)
	// 玩家选择托管，立即操作
	d := time.Duration(0)
	if isAutoPlay {
		d = room.autoTime.Sub(time.Now())
	}
	utils.ResetTimer(ply.operateTimer, d)
	room.OnTurn()
}

func (ply *DoudizhuPlayer) Discard(cards []int) {
	log.Debugf("player %d discard %v", ply.Id, cards)
	room := ply.Room()
	if ply != room.expectDiscardPlayer {
		return
	}
	// 判断牌是否有效、数量足够
	var m = make(map[int]int)
	for _, c := range cards {
		if !room.CardSet().IsCardValid(c) {
			return
		}
		m[c]++
	}
	for c, n := range m {
		if ply.cards[c] < n {
			return
		}
	}
	typ, _, _ := room.helper.GetType(cards)
	if typ == cardrule.DoudizhuNone {
		return
	}

	total := 0
	for _, n := range ply.cards {
		total += n
	}

	if other := room.discardPlayer; other != nil && !room.helper.Less(other.action, cards) {
		return
	}
	// OK
	log.Debugf("player %d discard ok %v", ply.Id, cards)

	ply.discardTimes++
	data := map[string]any{"cards": cards, "uid": ply.Id}
	if typ == cardrule.DoudizhuZhadan || typ == cardrule.DoudizhuWangzha {
		ply.boomTimes++
		ply.totalBoomTimes++
		room.boomTimes++

		maxBoomTimes := 1024
		if room.CanPlay(OptZhadan3) {
			maxBoomTimes = 3
		} else if room.CanPlay(OptZhadan4) {
			maxBoomTimes = 4
		} else if room.CanPlay(OptZhadan5) {
			maxBoomTimes = 5
		}
		multiple := 2
		if room.boomTimes > maxBoomTimes {
			multiple = 1
		}
		room.currentTimes *= multiple
		data["boomTimes"] = multiple
		data["currentTimes"] = room.currentTimes
	}
	utils.StopTimer(ply.operateTimer)
	room.Broadcast("discard", data)

	for _, c := range cards {
		ply.cards[c]--
	}
	ply.action = cards
	room.discardPlayer = ply

	if total == len(cards) {
		room.winPlayer = ply
		room.Award()
	} else {
		room.Turn()
	}
}

func (ply *DoudizhuPlayer) Jiaodizhu(choice int) {
	log.Debugf("player %d jiao di zhu", ply.Id)
	room := ply.Room()
	if room.jiaodizhuPlayer != ply {
		return
	}
	if ply.jiaodizhu != -1 {
		return
	}
	if choice != 0 {
		choice = 1
	}
	room.Broadcast("jiaodizhu", map[string]any{"choice": choice, "uid": ply.Id})
	ply.jiaodizhu = choice
	room.OnJiaodizhu()
}

func (ply *DoudizhuPlayer) Qiangdizhu(choice int) {
	log.Debugf("player %d qiang di zhu", ply.Id)
	room := ply.Room()
	if room.qiangdizhuPlayer != ply {
		return
	}
	if choice != 0 {
		choice = 1
	}
	for i := 0; i < room.NumSeat(); i++ {
		other := room.GetPlayer(i)
		if choice == 1 && other.qiangdizhu > 0 {
			choice = 2
		}
	}
	if choice == 2 {
		room.currentTimes *= 2
	}
	response := map[string]any{
		"choice":       choice,
		"uid":          ply.Id,
		"currentTimes": room.currentTimes,
	}
	room.Broadcast("qiangdizhu", response)
	ply.qiangdizhu = choice
	room.OnQiangdizhu()
}

func (ply *DoudizhuPlayer) Jiaofen(choice int) {
	log.Debugf("player %d jiao fen", ply.Id)
	room := ply.Room()
	if room.jiaofenPlayer != ply {
		return
	}
	if ply.jiaofen != -1 {
		return
	}
	if utils.InArray([]int{0, 1, 2, 3}, choice) == 0 {
		return
	}
	if other := room.choosePlayer; other != nil && choice > 0 && choice <= other.jiaofen {
		return
	}
	room.Broadcast("jiaofen", map[string]any{"choice": choice, "uid": ply.Id})
	ply.jiaofen = choice
	room.OnJiaofen()
}

func (ply *DoudizhuPlayer) Pass() {
	log.Debugf("player %d pass", ply.Id)
	room := ply.Room()
	if room.expectDiscardPlayer != ply {
		return
	}

	other := room.discardPlayer
	if other == nil {
		return
	}

	// OK
	ply.action = nil
	utils.StopTimer(ply.operateTimer)
	room.Broadcast("pass", map[string]any{"uid": ply.Id})
	room.Turn()
}

func (ply *DoudizhuPlayer) Timeout(fn func(), d time.Duration) {
	room := ply.Room()
	if room.IsTypeScore() {
		return
	}
	if ply.isAutoPlay {
		d = maxAutoTime
	}
	ply.operateTimer = ply.TimerGroup.NewTimer(fn, d)
}

func (ply *DoudizhuPlayer) Room() *DoudizhuRoom {
	if room := roomutils.GetRoomObj(ply.Player).CustomRoom(); room != nil {
		return room.(*DoudizhuRoom)
	}
	return nil
}

// 存档
/*
func (ply *DoudizhuPlayer) Replay(messageId string, i any) {
	switch messageId {
	case "startDealCard":
		room := ply.Room()
		data := i.(map[string]any)
		all := make([][]int, room.NumSeat())
		for k := 0; k < room.NumSeat(); k++ {
			other := room.GetPlayer(k)
			all[k] = other.GetSortedCards()
		}
		data["all"] = all
		defer func() { delete(data, "all") }()
	}
	ply.Player.Replay(messageId, i)
}
*/

func GetPlayer(id int) *DoudizhuPlayer {
	if p := service.GetPlayer(id); p != nil {
		return p.GameAction.(*DoudizhuPlayer)
	}
	return nil
}

func (ply *DoudizhuPlayer) GetSeatIndex() int {
	return roomutils.GetRoomObj(ply.Player).GetSeatIndex()
}
