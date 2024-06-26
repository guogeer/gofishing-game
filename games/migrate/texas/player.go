package texas

// 2017-8-22

import (
	"gofishing-game/internal/errcode"
	"gofishing-game/internal/gameutils"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
	"time"

	"github.com/guogeer/quasar/v2/config"
	"github.com/guogeer/quasar/v2/log"
	"github.com/guogeer/quasar/v2/utils"
)

const (
	AutoCheckOrFold = 1 << iota // 过或弃
	AutoCall                    // 跟到底
	AutoShow                    // 结束时亮牌
)

const (
	ActionNone  = iota
	ActionCheck // 过牌
	ActionCall  // 跟注
	ActionRaise // 加注
	ActionAllIn // 全压
	ActionFold  // 弃牌
)

func maxInArray(some []int64) int64 {
	max := some[0]
	for i := 1; i < len(some); i++ {
		if max < some[i] {
			max = some[i]
		}
	}
	return max
}

// 玩家信息
type TexasUserInfo struct {
	service.UserInfo
	SeatIndex int   `json:"seatIndex,omitempty"`
	Cards     []int `json:"cards,omitempty"`
	IsReady   bool  `json:"isReady,omitempty"`

	Bankroll        int64 `json:"bankroll,omitempty"`
	DefaultBankroll int64 `json:"defaultBankroll,omitempty"`

	Action    int   `json:"action,omitempty"`
	LastBlind int64 `json:"lastBlind,omitempty"`

	TotalBlind int64 `json:"totalBlind,omitempty"` // 押注

	IsShow bool `json:"isShow,omitempty"`

	Auto     int   `json:"auto,omitempty"`
	Match    []int `json:"match,omitempty"`
	CardType int   `json:"cardType,omitempty"`
}

type TexasPlayer struct {
	*service.Player

	cards [2]int // 手牌

	bankroll        int64 // 筹码
	defaultBankrool int64

	potId      int
	totalBlind int64 // 押注
	action     int
	lastBlind  int64

	winGold int64
	auto    int
	isShow  bool

	rebuyTimes int // 重购次数
	addonTimes int // 增购次数
	rebuyBlind int64
	addonBlind int64

	autoFoldCounter int
	autoTimer       *utils.Timer
	operateTimer    *utils.Timer
	failTimer       *utils.Timer
	isFail          bool
	// wallet          int
}

func (ply *TexasPlayer) TryEnter() errcode.Error {
	return nil
}

func (ply *TexasPlayer) BeforeEnter() {
}

func (ply *TexasPlayer) AfterEnter() {
	room := ply.Room()
	bankroll := ply.defaultBankroll()
	// 选择过筹码的场次，第二次直接坐下
	seatId := room.GetEmptySeat()
	if seatId != roomutils.NoSeat && ply.GetSeatIndex() == roomutils.NoSeat && bankroll > 0 {
		ply.SitDown(seatId)
	}
}

func (ply *TexasPlayer) TryLeave() errcode.Error {
	room := ply.Room()
	if room.IsTypeScore() && room.Status != 0 {
		return errcode.Retry
	}
	return nil
}

func (ply *TexasPlayer) BeforeLeave() {
	ply.SitUp()
}

func (ply *TexasPlayer) IsPlaying() bool {
	return roomutils.GetRoomObj(ply.Player).IsReady() && ply.action != ActionFold
}

func (ply *TexasPlayer) initGame() {
	for k := range ply.cards {
		ply.cards[k] = 0
	}

	ply.potId = 0
	ply.totalBlind = 0

	ply.action = ActionNone
	ply.lastBlind = 0

	ply.isShow = false
	ply.auto = 0
}

func (ply *TexasPlayer) GameOver() {
	ply.initGame()
}

// gold = -2, timeout
// gold = -1, fold
// gold =  0, check
// gold >  0, call or raise
func (ply *TexasPlayer) TakeAction(gold int64) {
	if !ply.IsPlaying() {
		return
	}

	room := ply.Room()
	maxAllIn := ply.maxAllIn()
	log.Debugf("player %d bank %d bet %d maxAllIn %d", ply.Id, ply.bankroll, gold, maxAllIn)
	if room.Status == 0 {
		return
	}
	if room.activePlayer != ply {
		return
	}
	// more than bankroll
	if gold > ply.bankroll || gold > maxAllIn {
		return
	}
	totalBlind := room.allBlind[ply.GetSeatIndex()]
	maxBlind := maxInArray(room.allBlind[:])
	log.Debugf("player %d bank %d bet %d totalBlind %d maxBlind %d ok", ply.Id, ply.bankroll, gold, totalBlind, maxBlind)
	if gold >= 0 && gold < ply.bankroll {
		if totalBlind+gold < maxBlind { // 跟注
			return
		}
		if totalBlind+gold > maxBlind && totalBlind+gold < room.raise { // 加注
			return
		}
	}
	ply.lastBlind = gold
	if gold < 0 { // 弃牌
		ply.action = ActionFold
	} else if gold == maxAllIn { // 全压
		ply.action = ActionAllIn
	} else if gold == 0 { // 过牌
		ply.action = ActionCheck
	} else if totalBlind+gold == maxBlind { // 跟注
		ply.action = ActionCall
	} else if totalBlind+gold > maxBlind { // 加注
		ply.action = ActionRaise
	}
	if gold >= 0 {
		ply.totalBlind += gold
		room.allBlind[ply.GetSeatIndex()] += gold
		ply.AddBankroll(-gold)
	}
	data := map[string]any{
		"uid":    ply.Id,
		"gold":   gold,
		"action": ply.action,
	}
	// 超时弃牌
	if gold == -2 {
		ply.autoFoldCounter++
	} else {
		ply.autoFoldCounter = 0
	}

	ply.SetAutoPlay(0) // 清除托管
	utils.StopTimer(ply.autoTimer)
	utils.StopTimer(ply.operateTimer)
	room.Broadcast("takeAction", data)
	room.OnTakeAction()

	// 超时两次自动弃牌
	if ply.autoFoldCounter > 1 {
		ply.SitUp()
	}
}

func (ply *TexasPlayer) GetUserInfo(self bool) *TexasUserInfo {
	info := &TexasUserInfo{}
	info.UserInfo = ply.UserInfo
	// info.UId = ply.GetCharObj().Id
	info.SeatIndex = ply.GetSeatIndex()
	info.Action = ply.action
	info.IsShow = ply.isShow
	info.IsReady = roomutils.GetRoomObj(ply.Player).IsReady()
	info.Bankroll = ply.bankroll
	info.DefaultBankroll = ply.defaultBankroll()

	room := ply.Room()
	if room.Status == roomutils.RoomStatusPlaying && ply.IsPlaying() {
		info.LastBlind = ply.lastBlind
		info.TotalBlind = ply.totalBlind
		if self {
			info.Cards = ply.cards[:]
			info.CardType, info.Match = ply.match()
			info.Auto = ply.auto
		}
	}
	return info
}

// 坐下
func (ply *TexasPlayer) SitDown(seatIndex int) {
	room := ply.Room()
	log.Debugf("player %d sit down", ply.Id)

	var e errcode.Error
	defer func() {
		if e != nil {
			ply.WriteErr("sitDown", nil, map[string]any{"uid": ply.Id})
		}
	}()
	subId := room.SubId
	minBankroll, _ := config.Int("texasroom", subId, "minBankroll")
	if ply.BagObj().NumItem(gameutils.ItemIdGold) < minBankroll {
		e = errcode.MoreItem(gameutils.ItemIdGold)
		return
	}
	if code := roomutils.GetRoomObj(ply.Player).SitDown(seatIndex); code != nil {
		return
	}
	// OK
	info := ply.GetUserInfo(false)
	room.Broadcast("sitDown", gameutils.MergeError(nil, map[string]any{"uid": ply.Id, "info": info}))
	ply.initBankroll()
	roomutils.GetRoomObj(ply.Player).Ready()
}

func (ply *TexasPlayer) initBankroll() {
	room := ply.Room()
	subId := room.SubId

	var bankroll int64
	if room.IsTypeTournament() {
		bankroll, _ = config.Int("tournament", room.Tournament().Id, "bankroll")
	} else {
		minBankroll, _ := config.Int("texasroom", subId, "minBankroll")

		bankroll = ply.defaultBankroll()
		if bankroll < minBankroll {
			bankroll = minBankroll
		}
		if bankroll > ply.BagObj().NumItem(gameutils.ItemIdGold) {
			bankroll = ply.BagObj().NumItem(gameutils.ItemIdGold)
		}
		ply.AddGold(-bankroll, "texas")
	}
	ply.AddBankroll(bankroll)
}

func (ply *TexasPlayer) AddBankroll(bankroll int64) {
	room := ply.Room()

	ply.bankroll += bankroll
	room.Broadcast("addBankroll", map[string]any{"uid": ply.Id, "gold": bankroll})
}

// 站起
func (ply *TexasPlayer) SitUp() {
	if ply.GetSeatIndex() == roomutils.NoSeat {
		return
	}

	room := ply.Room()
	log.Debugf("player %d sit up", ply.Id)
	if room.Status == roomutils.RoomStatusPlaying && ply.IsPlaying() {
		ply.action = ActionFold
		room.OnTakeAction()
	}
	// 回收筹码
	ply.AddGold(ply.bankroll, "texas_back")
	ply.bankroll = 0
	roomutils.GetRoomObj(ply.Player).SitUp()
}

func (ply *TexasPlayer) Room() *TexasRoom {
	if room := roomutils.GetRoomObj(ply.Player).CustomRoom(); room != nil {
		return room.(*TexasRoom)
	}
	return nil
}

func (ply *TexasPlayer) ChooseBankroll(gold int64) {
	room := ply.Room()
	subId := room.SubId
	minBankroll, _ := config.Int("texasroom", subId, "minBankroll")
	maxBankroll, _ := config.Int("texasroom", subId, "maxBankroll")
	if minBankroll != 0 && minBankroll > gold {
		return
	}
	if maxBankroll != 0 && maxBankroll < gold {
		return
	}
	if room.IsTypeTournament() {
		return
	}

	ply.defaultBankrool = gold
	ply.WriteJSON("chooseBankroll", map[string]any{"gold": gold})
}

func (ply *TexasPlayer) defaultBankroll() int64 {
	return ply.defaultBankrool
}

func (ply *TexasPlayer) match() (int, []int) {
	room := ply.Room()
	helper := room.helper
	if len(room.cards) < 3 {
		return 0, nil
	}

	cards := make([]int, len(ply.cards)+len(room.cards))
	copy(cards, ply.cards[:])
	copy(cards[len(ply.cards):], room.cards)
	typ, _ := helper.GetType(cards)
	return typ, helper.Match(cards)
}

func (ply *TexasPlayer) OnTurn() {
	room := ply.Room()
	act := room.activePlayer
	data := map[string]any{
		"uid":  act.Id,
		"sec":  room.Countdown(),
		"sec0": time.Now().Add(room.maxAutoTime()).Unix(),
	}
	if ply == act && ply.IsPlaying() {
		totalBlind := room.allBlind[ply.GetSeatIndex()]
		gold := maxInArray(room.allBlind[:]) - totalBlind
		raise := room.raise - totalBlind

		if gold > ply.bankroll {
			gold = ply.bankroll
		}
		if raise > ply.bankroll {
			raise = ply.bankroll
		}
		data["gold"] = gold
		data["raise"] = raise
		data["allIn"] = ply.maxAllIn()
	}
	if auto := ply.auto; auto|AutoShow == AutoShow {
		ply.WriteJSON("turn", data)
	}
}

func (ply *TexasPlayer) AutoPlay() {
	auto := ply.auto
	room := ply.Room()
	log.Debug("ply auto play", ply.auto)
	if auto&AutoShow != 0 {
		ply.ShowCard(true)
	}

	if ply == room.activePlayer {
		maxAllIn := ply.maxAllIn()
		gold := maxInArray(room.allBlind[:]) - room.allBlind[ply.GetSeatIndex()]
		if gold > ply.bankroll {
			gold = ply.bankroll
		}
		if gold > maxAllIn {
			gold = maxAllIn
		}
		if auto&AutoCheckOrFold != 0 {
			if gold != 0 {
				gold = -1
			}
			ply.TakeAction(gold)
		} else if auto&AutoCall != 0 {
			ply.TakeAction(gold)
		}
	}
}

func (ply *TexasPlayer) SetAutoPlay(auto int) {
	// 让或弃、跟到底两个选项不可同时存在
	rejectAuto := 0
	oldAuto := ply.auto
	if oldAuto&AutoCheckOrFold > 0 {
		rejectAuto = AutoCheckOrFold
	}
	if oldAuto&AutoCall > 0 {
		rejectAuto = AutoCall
	}
	if auto&AutoCheckOrFold > 0 && auto&AutoCall > 0 {
		auto ^= rejectAuto
	}

	ply.auto = auto
	ply.WriteJSON("setAutoPlay", map[string]any{"auto": auto})

	ply.AutoPlay()
}

func (ply *TexasPlayer) ShowCard(isShow bool) {
	room := ply.Room()
	if room.Status == 0 {
		return
	}
	if ply.isShow == isShow {
		return
	}
	ply.isShow = isShow
	room.Broadcast("showCard", map[string]any{"uid": ply.Id, "isShow": isShow})
}

func (ply *TexasPlayer) Rebuy() {
	room := ply.Room()
	if !room.IsTypeTournament() {
		return
	}
	tournament := room.Tournament()
	if tournament.RebuyFee > ply.BagObj().NumItem(gameutils.ItemIdGold) {
		return
	}
	if ply.rebuyBlind+ply.bankroll > tournament.Bankroll {
		return
	}
	if !tournament.IsAbleRebuy(room.blindLoop) {
		return
	}
	if ply.rebuyTimes > tournament.RebuyTimes {
		return
	}
	ply.rebuyTimes++
	ply.rebuyBlind += tournament.Bankroll
	ply.WriteJSON("rebuy", map[string]any{"uid": ply.Id})
}

func (ply *TexasPlayer) Addon() {
	room := ply.Room()
	if !room.IsTypeTournament() {
		return
	}
	tournament := room.Tournament()
	if tournament.AddonFee > ply.BagObj().NumItem(gameutils.ItemIdGold) {
		return
	}
	if ply.addonBlind+ply.bankroll > tournament.Bankroll*2 {
		return
	}
	if !tournament.IsAbleAddon(room.blindLoop) {
		return
	}
	if ply.addonTimes > tournament.AddonTimes {
		return
	}
	ply.addonTimes++
	ply.addonBlind += tournament.Bankroll * 2
	ply.WriteJSON("addon", map[string]any{"uid": ply.Id})
}

// 全压上限，不能超过游戏中筹码第二多的玩家
func (ply *TexasPlayer) maxAllIn() int64 {
	room := ply.Room()

	var first, second int64
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil && p.IsPlaying() {
			total := p.bankroll + p.totalBlind
			if first <= total {
				first, second = total, first
			} else if second < total {
				second = total
			}
		}
	}
	gold := second - ply.totalBlind
	if gold > ply.bankroll {
		gold = ply.bankroll
	}
	return gold
}

func (ply *TexasPlayer) GetSeatIndex() int {
	return roomutils.GetRoomObj(ply.Player).GetSeatIndex()
}

// TODO
func (ply *TexasPlayer) Fail() {
}

// 设置钱包
func (ply *TexasPlayer) SetWallet(n int64) {
}
