package internal

// 2017-03-01 与达总沟通后，牌局过程中无金币时，弹出充值和破产补助提示
// 牌局结束后，玩家选择继续或换桌时触发提示，时间为30s
// 注：破产补助达到次数时，仅提示充值

import (
	"gofishing-game/internal/cardutils"
	"gofishing-game/internal/errcode"
	"gofishing-game/internal/gameutils"
	mjutils "gofishing-game/migrate/mahjong/utils"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
	"gofishing-game/service/system"
	"quasar/utils"
	"quasar/utils/randutils"
	"time"

	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
)

type ChipResult struct {
	SeatId int   `json:"seatIndex"`
	Chip   int64 `json:"gold"`
}

type OperateTip struct {
	Card      int  `json:"card"`
	Type      int  `json:"type"`
	Chow      int  `json:"chow"`
	IsAbleWin bool `json:"isAbleWin"`
	Limit     int  `json:"limit"`
}

type ChipDetail struct {
	Seats     int   `json:"seats,omitempty"`
	Operate   int   `json:"operate,omitempty"` // Win,StraightKong,BentKong,Invisible
	Chip      int64 `json:"chip,omitempty"`
	Times     int   `json:"times"`
	Points    int   `json:"points"`
	Multiples int   `json:"multiples"`
	// Addition  [AllAdditionNum]int
	Addition2 map[string]int `json:"addition2"`
}

func (d ChipDetail) GetSeatIndex() int {
	bits := d.Seats
	for i := 0; i < 32 && bits != 0; i++ {
		if mask := 1 << uint(i); bits&mask == mask {
			return i
		}
	}
	return -1
}

type KongDetail struct {
	Type  int   `json:"type"`
	Chip  int64 `json:"chip"`
	Card  int   `json:"card"`
	other *MahjongPlayer
}

type MahjongPlayerInfo struct {
	service.UserInfo
	SeatId         int            `json:"seatId"`
	Cards          []int          `json:"cards"`
	WinHistory     []int          `json:"winHistory"`
	DiscardHistory []int          `json:"discardHistory"`
	Flowers        []int          `json:"flowers,omitempty"`
	DiscardColor   int            `json:"discardColor"`
	DiscardCard    int            `json:"discardCard"` // 打出去的牌
	DrawCard       int            `json:"drawCard"`
	Disband        int            `json:"disband"`
	Melds          []mjutils.Meld `json:"melds"`
	IsWin          bool           `json:"isWin"`
	IsReady        bool           `json:"isReady"`
	IsAutoPlay     bool           `json:"isAutoPlay"`
	IsBust         bool           `json:"isBust"`
	IsClose        bool           `json:"isClose"`

	// 红中赖子杠玩法的痞子、癞子杠
	Pilaigang []int `json:"pilaigang,omitempty"`
	// 出牌黑名单
	BlackList     []int       `json:"blackList,omitempty"`
	ExpectDiscard bool        `json:"expectDiscard,omitempty"`
	IsReadyHand   bool        `json:"isReadyHand,omitempty"`
	Multiples     int         `json:"multiples,omitempty"`
	WinOpts       []WinOption `json:"winOpts,omitempty"`
}

type LocalObj interface {
	IsAbleChow(int) bool
	IsAblePong() bool
	IsAbleWin() bool
	GetKongType(int) int
	OnDiscard()
	OnChow()
	OnPong()
	OnKong()
	OnDouble()

	// 玩法有自己的听牌规则
	CheckReadyHand() []ReadyHandOption
}

// 湖北地区
type HubeiObj interface {
	ChoosePiao(int)
}

type MahjongPlayer struct {
	*service.Player

	handCards   []int // 手牌
	melds       []mjutils.Meld
	chipHistory []ChipDetail
	kongChip    []int64
	discardNum  int   // 出牌数
	flowers     []int // 花牌

	// rhHistory 加倍或听牌的记录
	discardHistory, drawHistory, winHistory, rhHistory  []int
	isBust, isWin, isAutoPlay, isReadyHand              bool // 破产
	delayChow, delayPong, delayKong, robKong, leaveGame bool

	chowCard, drawCard int

	lastKong       KongDetail   // 最近的一次杠
	kongHistory    []KongDetail // 杠牌历史记录
	continuousKong []KongDetail // 本轮杠的记录

	operateTips   []OperateTip
	readyHandTips []ReadyHandOption

	defaultTriCards, triCards  [3]int
	defaultColor, discardColor int

	totalTimes      map[string]int
	blackList       []int // 出牌黑名单
	expectReadyHand bool
	forceReadyHand  bool

	// 地方麻将差异
	localObj LocalObj

	isPassWin             bool         // 本轮过胡
	unableWinCards        map[int]bool // 本轮过掉的牌不能胡
	continuousDealerTimes int          // 连庄次数
	multiples             int
	isAbleLookOthers      bool
	tempChip              int64
	isBustOrNot           bool // 选择破产

	operateTimer *utils.Timer
}

func NewMahjongPlayer() *MahjongPlayer {
	p := &MahjongPlayer{
		drawCard:     -1,
		discardColor: -1,
		handCards:    make([]int, MaxCard),
		totalTimes:   make(map[string]int),

		unableWinCards: make(map[int]bool),
	}
	return p
}

func (ply *MahjongPlayer) TryEnter() errcode.Error {
	return nil
}

func (ply *MahjongPlayer) BeforeEnter() {
}

func (ply *MahjongPlayer) AfterEnter() {
	room := ply.Room()
	seatNum := room.NumSeat()
	if len(ply.kongChip) < seatNum {
		ply.kongChip = make([]int64, seatNum)
	}
}

func (ply *MahjongPlayer) BeforeLeave() {
	room := ply.Room()
	if room != nil && room.Status != 0 {
		roomutils.GetRoomObj(ply.Player).PrepareClone()
	}
	ply.AutoPlay(1)
	ply.continuousDealerTimes = 0
}

func (ply *MahjongPlayer) TryLeave() errcode.Error {
	room := ply.Room()
	if room != nil && room.Status != 0 && !ply.leaveGame {
		return errcode.New("game_playing", "game is playing")
	}
	return nil
}

func (ply *MahjongPlayer) Room() *MahjongRoom {
	if room := roomutils.GetRoomObj(ply.Player).Room().CustomRoom(); room != nil {
		return room.(*MahjongRoom)
	}
	return nil
}

func (ply *MahjongPlayer) AddChipOrBust(n int64, way string) int64 {
	room := ply.Room()
	if ply.leaveGame {
		return 0
	}
	// 破产
	chip := ply.BagObj().NumItem(room.GetChipItem())
	if !ply.isBustOrNot && n+chip <= 0 && room.IsTypeNormal() {
		n = -chip
		room.deadline = time.Now().Add(MaxBustTime)
		data := map[string]any{
			"uid": ply.Id,
			"ts":  room.deadline.Unix(),
			// "bust": ply.BaseObj().GetBustInfo(),
		}
		ply.isBustOrNot = true
		room.Broadcast("bustOrNot", data)
		ply.Timeout(ply.Fail)
	}
	// roomutils.GetRoomObj(ply.Player)().WinGold += n
	ply.BagObj().Add(room.GetChipItem(), n, "bust")
	return n
}

// 破产
func (ply *MahjongPlayer) Fail() {
	log.Debugf("player %d choose fail", ply.Id)

	room := ply.Room()
	if room == nil {
		return
	}
	if room.Status == 0 {
		return
	}
	if room.IsTypeScore() {
		return
	}
	if ply.leaveGame || ply.BagObj().NumItem(room.GetChipItem()) > 0 {
		return
	}

	ply.isBust = true
	ply.leaveGame = true
	ply.isBustOrNot = false
	utils.StopTimer(ply.operateTimer)
	room.Broadcast("Fail", map[string]any{"uid": ply.Id})
	room.OnBust()
}

func (ply *MahjongPlayer) GetUserInfo(otherId int) *MahjongPlayerInfo {
	room := ply.Room()
	other := GetPlayer(otherId)
	data := &MahjongPlayerInfo{}
	data.UserInfo = ply.UserInfo

	// data.UId = ply.Id
	data.IsWin = ply.isWin
	data.WinHistory = ply.winHistory
	data.DiscardHistory = ply.discardHistory
	data.DiscardColor = ply.discardColor
	data.IsBust = ply.isBust
	data.IsReady = roomutils.GetRoomObj(other.Player).IsReady()
	data.IsAutoPlay = ply.isAutoPlay
	data.SeatId = ply.GetSeatIndex()
	data.Melds = ply.melds
	data.DrawCard = ply.drawCard
	data.IP = ply.IP
	// data.Disband = roomutils.GetRoomObj(ply.Player)().DisbandAnswer
	data.Cards = SortCards(ply.handCards)
	if ply.Id != otherId && !other.isAbleLookOthers {
		if data.DrawCard != -1 {
			data.DrawCard = 1
		}
		for k := range data.Cards {
			data.Cards[k] = 0
		}
	}
	if ply.Id == otherId && room.Status != 0 {
		data.WinOpts = ply.CheckWin()
	}

	data.Flowers = ply.flowers
	// TODO
	/*if obj, ok := ply.localObj.(*HongzhonglaizigangObj); ok {
		data.Pilaigang = obj.history
	}
	*/
	data.IsClose = ply.IsSessionClose

	if ply == room.discardPlayer {
		data.DiscardCard = room.lastCard
	}
	data.BlackList = ply.blackList
	if ply == room.expectDiscardPlayer {
		data.ExpectDiscard = true
	}
	data.IsReadyHand = ply.isReadyHand
	if room.CanPlay(OptAbleDouble) {
		data.Multiples = ply.multiples
	}
	return data
}

/*
func (ply *MahjongPlayer) Clone() *MahjongPlayer {
	sub := ply.Room().GetSubWorld()
	clone := sub.NewPlayer().CardPlayer().(*MahjongPlayer)
	clone.Player.Clone(ply.Player)
	ply.Player, clone.Player = clone.Player, ply.Player
	return clone
}
*/

func (ply *MahjongPlayer) AddChipHistory(details ...ChipDetail) {
	if len(details) == 0 {
		return
	}
	ply.chipHistory = append(ply.chipHistory, details...)
}

func (ply *MahjongPlayer) OnAddItems(items []gameutils.Item, way string) {
	ply.Player.OnAddItems(items, way)
	// 破产时金币到账
	room := ply.Room()
	for _, item := range items {
		if item.GetId() == gameutils.ItemIdGold && ply.BagObj().NumItem(gameutils.ItemIdGold) > 0 {
			ply.isBustOrNot = false
			room.OnBust()
		}
	}

}

func (ply *MahjongPlayer) StartGame() {
	// log.Infof("player %d start service., ply.Id)
	for i := range ply.handCards {
		ply.handCards[i] = 0
	}

	ply.isBust = false
	ply.isAutoPlay = false
	ply.isWin = false
	ply.leaveGame = false
	ply.discardColor = -1
	ply.drawCard = -1
	ply.discardColor = -1
	ply.winHistory = nil
	ply.rhHistory = nil
	ply.discardHistory = nil
	ply.drawHistory = nil
	ply.operateTips = nil
	ply.readyHandTips = nil
	ply.melds = nil
	ply.chipHistory = nil
	ply.triCards = [3]int{}

	// 出牌数
	ply.discardNum = 0
	// 杠牌记录
	ply.kongHistory = nil
	// 花牌
	ply.flowers = nil
	// 出牌黑名单
	ply.blackList = nil
	ply.isReadyHand = false
	ply.expectReadyHand = false
	ply.forceReadyHand = false
	for i := range ply.kongChip {
		ply.kongChip[i] = 0
	}
	ply.isPassWin = false
	ply.unableWinCards = make(map[int]bool)
	ply.multiples = 1
	ply.isAbleLookOthers = false
	ply.isBustOrNot = false
}

func (ply *MahjongPlayer) Prompt() {
	if ply.isAutoPlay {
		return
	}
	if len(ply.operateTips) == 0 && len(ply.readyHandTips) == 0 {
		return
	}
	room := ply.Room()
	tips := ply.operateTips
	// TODO 当前仅考虑二人
	if room.CanPlay(OptAbleDouble) {
		if _, ok := room.expectWinPlayers[ply.Id]; ok {
			points := 0
			tip := OperateTip{Type: mjutils.OperateDouble}
			for _, opt := range ply.CheckWin() {
				if opt.WinCard == room.lastCard {
					points = opt.Points
					break
				}
			}

			minChip := int64(0)
			for i := 0; i < room.NumSeat(); i++ {
				if other := room.GetPlayer(i); other != nil && other != ply {
					minChip = other.BagObj().NumItem(room.GetChipItem())
				}
			}
			cost, _ := config.Int("room", room.SubId, "cost")
			base := cost * int64(points)
			tip.Limit = int((minChip + base - 1) / base)
			tips = append(tips, tip)
		}
	}

	data := map[string]any{
		"operate":   tips,
		"readyHand": ply.readyHandTips,
	}
	if ply.forceReadyHand {
		data["isReadyHand"] = true
	}
	ply.WriteJSON("operateTip", data)
}

func (ply *MahjongPlayer) IsCheat() bool {
	return system.GetLoginObj(ply.Player).IsRobot()
}

func (ply *MahjongPlayer) TryDraw() int {
	room := ply.Room()
	seatId := ply.GetSeatIndex()
	if room.cheatSeats&(1<<uint(seatId)) > 0 && len(room.sample) > 0 {
		room.CardSet().MoveFront(room.sample[0])
		room.sample = room.sample[1:]
	}
	return room.CardSet().Deal()
}

// 摸牌
func (ply *MahjongPlayer) Draw() {
	room := ply.Room()

	cards := ply.handCards
	drawCard := InvalidCard
	for {
		percent, _ := config.Float("Room", room.SubId, "WinPercent")
		if randutils.IsPercentNice(percent) && room.cheatSeats == 0 {
			color := ply.discardColor
			for _, c := range cardutils.GetAllCards() {
				if cards[c] == 2 && c/10 != color && room.CardSet().Cheat(c) > 0 {
					drawCard = c
					break
				}
			}
		}
		if drawCard == -1 {
			drawCard = ply.TryDraw()
		}
		if !IsFlower(drawCard) {
			break
		}
		ply.flowers = append(ply.flowers, drawCard)
		room.Broadcast("Flower", map[string]any{"Flower": drawCard, "uid": ply.Id})
	}

	ply.drawCard = drawCard
	log.Infof("player %d draw card %d", ply.Id, drawCard)
	ply.OnDraw()
}

func (ply *MahjongPlayer) OnDraw() {
	log.Infof("player %d on draw card %d", ply.Id, ply.drawCard)

	c := ply.drawCard
	room := ply.Room()
	room.expectChowPlayer = nil
	room.expectPongPlayer = nil
	room.expectKongPlayer = nil
	room.expectWinPlayers = map[int]*MahjongPlayer{}

	room.discardPlayer = nil
	room.winPlayers = nil
	// 摸牌后，其他玩家不会再出现抢杠胡、杠上炮
	if room.kongPlayer != ply {
		room.kongPlayer = nil
	}
	// 清除出牌黑名单
	ply.blackList = nil
	// 如果其他人都离开游戏
	num := 0
	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		if p != nil && !p.leaveGame {
			num++
		}
	}
	if c == -1 || num < 2 {
		room.Award() // 牌局结束
		return
	}

	// 庄家摸的首张牌
	dealerFirstCard := (ply == room.dealer && len(ply.drawHistory) == 0)

	room.lastCard = c
	room.expectDiscardPlayer = ply

	ply.isPassWin = false
	ply.unableWinCards = make(map[int]bool)
	ply.drawHistory = append(ply.drawHistory, c)
	cards := ply.handCards[:]
	if !dealerFirstCard {
		cards[c]++
	}

	// 超时出牌
	ply.Timeout(func() { ply.autoDiscard() })

	var tips []OperateTip

	for _, c1 := range cardutils.GetAllCards() {
		if ply.localObj.GetKongType(c1) != -1 {
			tips = append(tips, OperateTip{Type: mjutils.OperateKong, Card: c1})
			room.expectKongPlayer = ply
			kc := c1
			ply.Timeout(func() { ply.Kong(kc) })
		}
	}

	// 胡牌
	if ply.localObj.IsAbleWin() {
		tips = append(tips, OperateTip{Type: mjutils.OperateWin, Card: c})
		room.expectWinPlayers[ply.Id] = ply
		ply.Timeout(func() { ply.Win() })
	}
	if room.CanPlay(OptBaoTing) && !ply.isReadyHand {
		opts := ply.ReadyHand()
		if len(opts) > 0 {
			ply.expectReadyHand = true
			tips = append(tips, OperateTip{Type: mjutils.OperateReadyHand})
		}
	}

	room.Timing()
	if !dealerFirstCard {
		for i := 0; i < room.NumSeat(); i++ {
			tempCard := 0
			other := room.GetPlayer(i)
			if ply == other || other.isAbleLookOthers {
				tempCard = c
			}
			other.WriteJSON("draw", map[string]any{"card": tempCard, "uid": ply.Id})
		}
	}

	log.Infof("draw operate tips %v", tips)
	ply.operateTips = tips
	if !ply.isReadyHand {
		ply.readyHandTips = ply.ReadyHand()
	}
	ply.Prompt()
}

func (ply *MahjongPlayer) GetKongType(c int) int {
	type_ := -1
	room := ply.Room()
	if c/10 == ply.discardColor { // 缺门的花色
		return type_
	}
	// 玩家可离开游戏
	if ply.leaveGame {
		return type_
	}
	if room.CardSet().Count() == 0 {
		return type_
	}
	// 赖子不可杠
	if room.IsAnyCard(c) {
		return type_
	}

	cards := ply.handCards
	if ply.drawCard == -1 && cards[c] == 3 && room.lastCard == c {
		type_ = mjutils.MeldStraightKong // 直杠
	}
	// 摸牌或吃碰可杠
	if ply.drawCard != -1 || room.CanPlay(OptAbleKongAfterChowOrPong) {
		if cards[c] == 4 {
			type_ = mjutils.MeldInvisibleKong // 暗杠
		}
		for _, m := range ply.melds {
			if m.Type == mjutils.MeldTriplet && m.Card == c && cards[c] > 0 {
				type_ = mjutils.MeldBentKong
				break
			}
		}
	}
	if type_ == -1 || !ply.isReadyHand {
		return type_
	}

	// 玩家已胡牌
	n := cards[c]
	bitmap := make(map[int]int)
	for _, wc := range ply.rhHistory {
		bitmap[wc] = 1
	}

	cards[c] = 0
	if c == ply.drawCard {
		cards[c] = 1
	}
	opts := ply.CheckWin()
	cards[c] = n

	if len(opts) == 0 {
		return -1
	}
	for _, opt := range opts {
		bitmap[opt.WinCard] |= 2
	}
	for _, b := range bitmap {
		if b == 1 {
			return -1
		}
	}
	return type_
}

func (ply *MahjongPlayer) GetSeatIndex() int {
	return roomutils.GetRoomObj(ply.Player).GetSeatIndex()
}

// 杠
func (ply *MahjongPlayer) Kong(c int) {
	log.Infof("player %d kong %d", ply.Id, c)
	if !cardutils.IsCardValid(c) {
		return
	}

	room := ply.Room()
	type_ := ply.localObj.GetKongType(c)
	// 无效的杠
	if type_ == -1 {
		return
	}
	if ply != room.expectKongPlayer {
		return
	}

	// 有其他玩家要吃牌
	if p := room.expectChowPlayer; p != nil && p != ply {
		// 不可胡牌
		if _, ok := room.expectWinPlayers[p.Id]; !ok {
			p.Pass()
		}
	}
	delete(room.expectWinPlayers, ply.Id)
	if len(room.winPlayers) > 0 { // 有玩家已胡牌
		ply.Pass()
		return
	}

	ply.delayKong = true
	ply.operateTips = nil
	ply.readyHandTips = nil
	if len(room.expectWinPlayers) > 0 { // 有玩家胡牌
		return
	}

	// OK
	ply.delayKong = false
	if p := room.discardPlayer; p != nil {
		h := p.discardHistory
		p.discardHistory = h[:len(h)-1]
	}
	room.Broadcast("kong", map[string]any{"code": errcode.CodeOk, "card": c, "type": type_, "uid": ply.Id})
	ply.lastKong = KongDetail{Card: c, Type: type_, other: room.discardPlayer}

	// 被抢杠胡时，玩家摸牌状态需清除
	// 导致了玩家被抢杠胡后，玩家在本轮接炮胡时，导致手牌异常被减去一张
	ply.drawCard = -1
	ply.handCards[c] = 0
	if type_ == mjutils.MeldBentKong {
		for k, m := range ply.melds {
			if m.Card == c {
				m.Type = mjutils.MeldBentKong
				ply.melds[k] = m
			}
		}
	} else {
		m := mjutils.Meld{Card: c, Type: type_}
		if other := room.discardPlayer; other != nil {
			m.SeatId = other.GetSeatIndex()
		}
		ply.melds = append(ply.melds, m)
	}

	room.expectChowPlayer = nil
	room.expectPongPlayer = nil
	room.expectKongPlayer = nil
	room.expectDiscardPlayer = nil
	room.discardPlayer = nil
	room.kongPlayer = ply
	// 摸的牌和杠的牌的可能不同
	room.lastCard = c
	utils.StopTimer(ply.operateTimer)

	// 判断是否有抢杠胡
	// 增加明杠可抢
	// 2017-5-16 Guogeer
	if (type_ == mjutils.MeldBentKong ||
		(type_ == mjutils.MeldStraightKong && room.CanPlay(OptMingGangKeQiang))) &&
		room.CanPlay(OptAbleRobKong) {
		for i := 0; i < room.NumSeat(); i++ {
			if other := room.GetPlayer(i); ply != other && other.localObj.IsAbleWin() {
				room.expectWinPlayers[other.Id] = other
				other.Timeout(func() { other.Win() })
				other.readyHandTips = nil
				other.operateTips = []OperateTip{{Type: mjutils.OperateWin, Card: c}}
				other.Prompt()
			}
		}
	}

	if len(room.expectWinPlayers) > 0 {
		ply.robKong = true
	} else {
		ply.KongOk()
	}
}

func (ply *MahjongPlayer) KongOk() {
	room := ply.Room()
	ply.robKong = false
	ply.kongHistory = append(ply.kongHistory, ply.lastKong)
	ply.continuousKong = append(ply.continuousKong, ply.lastKong)
	ply.localObj.OnKong()

	type_ := ply.lastKong.Type
	if type_ == mjutils.MeldInvisibleKong {
		ply.totalTimes["AG"]++
	} else {
		ply.totalTimes["MG"]++
	}

	// ply.GameData.Kong++
	// 杠后结算
	if room.CanPlay(OptCostAfterKong) {
		var result []ChipResult
		var effectPlayers []*MahjongPlayer

		// 默认1倍底注
		times := 1
		unit := room.Unit()
		bills := make([]Bill, room.NumSeat())
		// 暗杠2倍
		if type_ == mjutils.MeldInvisibleKong {
			times = 2
		}
		// 直杠默认3倍
		if type_ == mjutils.MeldStraightKong {
			times = 3
			if room.CanPlay(OptStraightKong2) {
				times = 2
			}
		}
		log.Info("kong", unit, type_)

		// detail := ChipChip{Operate: type_, Times: times, SeatId: ply.GetSeatIndex()}
		detail := ChipDetail{Operate: type_, Times: times, Seats: 1 << uint(ply.GetSeatIndex())}
		if type_ == mjutils.MeldStraightKong {
			effectPlayers = append(effectPlayers, ply.lastKong.other)
		} else {
			for i := 0; i < room.NumSeat(); i++ {
				p := room.GetPlayer(i)
				if p != nil && p != ply && !p.leaveGame {
					effectPlayers = append(effectPlayers, p)
				}
			}
		}

		for _, p := range effectPlayers {
			detail.Chip = -unit * int64(times)

			bill := &bills[p.GetSeatIndex()]
			bill.Details = append(bill.Details, detail)

		}
		room.Billing(bills)
		for seatId, bill := range bills {
			if len(bill.Details) > 0 {
				one := bill.Details[0]
				result = append(result, ChipResult{SeatId: seatId, Chip: one.Chip})
				if seatId == ply.GetSeatIndex() {
					ply.lastKong.Chip = one.Chip
				} else {
					ply.kongChip[seatId] += one.Chip
				}
			}
		}
		room.Broadcast("compute", map[string]any{"operate": type_, "result": result})
	}

	room.delayDuration += maxDelayAfterKong
	// 检测是否有人破产
	if room.CountBustPlayers() == 0 {
		ply.Draw()
	} else {
		room.bustTimeout = ply.Draw
	}
	room.delayDuration -= maxDelayAfterKong
}

// 吃
func (ply *MahjongPlayer) Chow(c int) {
	log.Infof("player %d chow", ply.Id)
	room := ply.Room()
	if room.expectChowPlayer != ply {
		return
	}
	if !cardutils.IsCardValid(c) {
		return
	}
	dc := room.lastCard
	if c < dc-2 || c > dc+2 {
		return
	}
	cards := ply.handCards
	cards[dc]++
	ok := cards[c] > 0 && cards[c+1] > 0 && cards[c+2] > 0
	cards[dc]--
	if !ok {
		return
	}

	// OK
	delete(room.expectWinPlayers, ply.Id)
	if len(room.winPlayers) > 0 { // 有玩家已胡牌
		ply.Pass()
		return
	}

	ply.chowCard = c
	ply.delayChow = true
	ply.operateTips = nil
	ply.readyHandTips = nil
	if len(room.expectWinPlayers) > 0 { // 有玩家胡牌
		return
	}
	// 有其他玩家碰牌
	if p := room.expectPongPlayer; p != nil && p != ply {
		if p.delayPong {
			ply.Pass()
		}
		return
	}
	// 有其他玩家杠牌
	if p := room.expectKongPlayer; p != nil && p != ply {
		if p.delayKong {
			ply.Pass()
		}
		return
	}

	ply.delayChow = false
	log.Infof("player %d chow card %d", ply.Id, dc)
	if p := room.discardPlayer; p != nil {
		h := p.discardHistory
		p.discardHistory = h[:len(h)-1]
	}

	// OK
	m := mjutils.Meld{Card: c, Type: mjutils.MeldSequence}
	if other := room.discardPlayer; other != nil {
		m.SeatId = other.GetSeatIndex()
	}
	ply.melds = append(ply.melds, m)
	cards[dc]++
	cards[c]--
	cards[c+1]--
	cards[c+2]--
	data := map[string]any{
		"code": errcode.CodeOk,
		"card": c,
		"chow": dc,
		"uid":  ply.Id,
	}
	var blackList = []int{dc}
	if dc == c && cardutils.IsCardValid(c+3) &&
		!room.IsAnyCard(c+3) {
		blackList = append(blackList, c+3)
	}
	if dc == c+2 && cardutils.IsCardValid(c-1) &&
		!room.IsAnyCard(c-1) {
		blackList = append(blackList, c-1)
	}
	ply.blackList = blackList
	data["blackList"] = blackList
	room.Broadcast("chow", data)

	ply.localObj.OnChow()

	room.expectChowPlayer = nil
	room.expectPongPlayer = nil
	room.expectKongPlayer = nil
	room.expectDiscardPlayer = ply
	room.discardPlayer = nil
	room.kongPlayer = nil
	utils.StopTimer(ply.operateTimer)

	ply.operateTips = nil
	if !ply.isReadyHand {
		ply.readyHandTips = ply.ReadyHand()
	}
	// 自动出牌
	ply.Timeout(func() { ply.autoDiscard() })
	// 湖北钟祥地区吃碰以后还可以继续杠
	var tips []OperateTip
	for _, c1 := range cardutils.GetAllCards() {
		if ply.localObj.GetKongType(c1) != -1 {
			tips = append(tips, OperateTip{Type: mjutils.OperateKong, Card: c1})
			room.expectKongPlayer = ply
			kc := c1
			ply.Timeout(func() { ply.Kong(kc) })
		}
	}
	if room.CanPlay(OptBaoTing) && !ply.isReadyHand {
		opts := ply.ReadyHand()
		if len(opts) > 0 {
			ply.expectReadyHand = true
			tips = append(tips, OperateTip{Type: mjutils.OperateReadyHand})
		}
	}

	ply.operateTips = tips

	room.Timing()
	ply.Prompt()
}

// 碰
func (ply *MahjongPlayer) Pong() {
	log.Infof("player %d pong", ply.Id)
	cards := ply.handCards
	room := ply.Room()
	dc := room.lastCard
	if cards[dc] < 2 {
		return
	}

	if room.expectPongPlayer != ply {
		return
	}

	// 有其他玩家要吃牌
	if p := room.expectChowPlayer; p != nil && p != ply {
		// 不可胡牌
		if _, ok := room.expectWinPlayers[p.Id]; !ok {
			p.Pass()
		}
	}
	// 一炮多响时，某个玩家同时碰(吃、杠)、胡
	delete(room.expectWinPlayers, ply.Id)
	if len(room.winPlayers) > 0 { // 已有玩家胡牌
		ply.Pass()
		return
	}
	ply.delayPong = true
	ply.operateTips = nil
	ply.readyHandTips = nil

	if len(room.expectWinPlayers) > 0 {
		return
	}

	ply.delayPong = false
	m := mjutils.Meld{Card: dc, Type: mjutils.MeldTriplet}
	if other := room.discardPlayer; other != nil {
		m.SeatId = other.GetSeatIndex()
	}
	ply.melds = append(ply.melds, m) // 增加一个刻子
	log.Infof("player %d pong card %d", ply.Id, dc)
	if p := room.discardPlayer; p != nil {
		h := &p.discardHistory
		*h = (*h)[:len(*h)-1]
	}

	cards[dc] -= 2
	room.Broadcast("Pong", map[string]any{"code": errcode.CodeOk, "card": dc, "uid": ply.Id})
	ply.localObj.OnPong()

	room.expectChowPlayer = nil
	room.expectPongPlayer = nil
	room.expectKongPlayer = nil
	room.expectDiscardPlayer = ply
	room.discardPlayer = nil
	utils.StopTimer(ply.operateTimer)

	ply.operateTips = nil
	if !ply.isReadyHand {
		ply.readyHandTips = ply.ReadyHand()
	}

	ply.Timeout(func() { ply.autoDiscard() })
	// 湖北钟祥地区吃碰以后还可以继续杠
	var tips []OperateTip
	for _, c1 := range cardutils.GetAllCards() {
		if ply.localObj.GetKongType(c1) != -1 {
			tips = append(tips, OperateTip{Type: mjutils.OperateKong, Card: c1})
			room.expectKongPlayer = ply
			kc := c1
			ply.Timeout(func() { ply.Kong(kc) })
		}
	}
	if room.CanPlay(OptBaoTing) && !ply.isReadyHand {
		opts := ply.ReadyHand()
		if len(opts) > 0 {
			ply.expectReadyHand = true
			tips = append(tips, OperateTip{Type: mjutils.OperateReadyHand})
		}
	}

	ply.operateTips = tips

	room.Timing()
	ply.Prompt()
}

// 胡牌
func (ply *MahjongPlayer) Win() {
	if roomutils.GetRoomObj(ply.Player).Room() == nil {
		return
	}

	room := ply.Room()
	if _, ok := room.expectWinPlayers[ply.Id]; !ok {
		return
	}
	if p := room.expectKongPlayer; p != ply && p != nil {
		// 玩家不可胡牌
		if _, ok := room.expectWinPlayers[p.Id]; !ok {
			p.Pass()
		}
	}
	if p := room.expectPongPlayer; p != ply && p != nil {
		// 玩家不可胡牌
		if _, ok := room.expectWinPlayers[p.Id]; !ok {
			p.Pass()
		}
	}
	if p := room.expectChowPlayer; p != ply && p != nil {
		// 玩家不可胡牌
		if _, ok := room.expectWinPlayers[p.Id]; !ok {
			p.Pass()
		}
	}

	ply.operateTips = nil
	ply.readyHandTips = nil
	ply.WriteErr("Win", nil)

	// OK
	utils.StopTimer(ply.operateTimer)
	ply.isWin = true
	ply.isReadyHand = true
	// drawCard := ply.drawCard

	delete(room.expectWinPlayers, ply.Id)
	room.winPlayers = append(room.winPlayers, ply)
	if ply == room.expectChowPlayer {
		room.expectChowPlayer = nil
	}
	if ply == room.expectPongPlayer {
		room.expectPongPlayer = nil
	}
	if ply == room.expectKongPlayer {
		room.expectKongPlayer = nil
	}

	// 一炮多响时离放炮最近的人胡
	if ply.drawCard == -1 && room.CanPlay(OptFangPaoJiuJinHu) {
		somebody := make([]*MahjongPlayer, 0, 8)
		somebody = append(somebody, room.winPlayers...)
		for _, other := range room.expectWinPlayers {
			somebody = append(somebody, other)
		}

		last := roomutils.NoSeat
		boom := room.boomPlayer()
		for _, other := range somebody {
			if last == roomutils.NoSeat || room.distance(boom.GetSeatIndex(), last) > room.distance(boom.GetSeatIndex(), other.GetSeatIndex()) {
				last = other.GetSeatIndex()
			}
		}
		if last == ply.GetSeatIndex() {
			room.winPlayers = []*MahjongPlayer{ply}

			wait := len(room.expectWinPlayers) > 0
			for _, other := range room.expectWinPlayers {
				other.Pass()
			}
			if !wait {
				room.OnWin()
			}
			return
		}
	}
	room.OnWin()
}

func (ply *MahjongPlayer) Discard(c int) {
	log.Infof("player %d discard card %d", ply.Id, c)
	PrintCards(ply.handCards)
	if !cardutils.IsCardValid(c) {
		return
	}

	room := ply.Room()
	cards := ply.handCards
	if cards[c] < 1 || room.expectDiscardPlayer != ply {
		return
	}
	if utils.InArray(ply.blackList, c) > 0 {
		return
	}

	// 听(胡)牌以后必须出摸到的牌
	if ply.isReadyHand && ply.drawCard != c {
		return
	}
	if ply.isReadyHand && ply.expectReadyHand {
		return
	}

	if ply.isReadyHand {
		ply.forceReadyHand = false
		ply.expectReadyHand = false
	}
	var winOpts []mjutils.WinOption
	if ply.expectReadyHand {
		lastCard := room.lastCard
		room.lastCard = c
		winOpts = ply.CheckWin()
		room.lastCard = lastCard
		if len(winOpts) == 0 && len(winOpts) > 0 {
			return
		}
		ply.forceReadyHand = true
	}

	// OK
	data := map[string]any{"code": errcode.CodeOk, "uid": ply.Id, "card": c}
	if ply.expectReadyHand {
		data["IsReadyHand"] = true
	}
	room.Broadcast("Discard", data)
	if ply.expectReadyHand {
		ply.ReadyHandOk()
	}

	log.Infof("player %d discard card %d ok", ply.Id, c)

	cards[c]--
	room.expectChowPlayer = nil
	room.expectPongPlayer = nil
	room.expectKongPlayer = nil
	room.expectDiscardPlayer = nil
	if room.kongPlayer != ply {
		room.kongPlayer = nil
	}
	room.discardPlayer = ply
	room.lastCard = c
	delete(room.expectWinPlayers, ply.Id)

	ply.drawCard = -1
	ply.discardNum++
	ply.discardHistory = append(ply.discardHistory, c)
	ply.continuousKong = nil
	ply.operateTips = nil
	ply.readyHandTips = nil
	// 清除出牌黑名单
	ply.blackList = nil
	ply.isPassWin = false
	// 清理过牌
	ply.unableWinCards = make(map[int]bool)
	// ply.unableWinCards[c] = true
	ply.localObj.OnDiscard()

	// 没人出牌
	if room.discardPlayer == nil {
		return
	}

	isTip := false
	for i := 0; i < room.NumSeat(); i++ {
		other := room.GetPlayer(i)
		if other == nil || other == ply {
			continue
		}

		if other.leaveGame || other.discardColor == c/10 {
			continue
		}

		var tips []OperateTip
		// 吃
		for start := c - 2; start <= c; start++ {
			if other.localObj.IsAbleChow(start) {
				room.expectChowPlayer = other
				tips = append(tips, OperateTip{Type: mjutils.OperateChow, Card: start, Chow: c})
			}
		}
		// 碰
		if other.localObj.IsAblePong() {
			room.expectPongPlayer = other
			tips = append(tips, OperateTip{Type: mjutils.OperatePong, Card: c})
			other.Timeout(func() { other.Pass() })
		}
		// 杠
		if t := other.localObj.GetKongType(c); t != -1 {
			room.expectKongPlayer = other
			operateTip := OperateTip{Type: mjutils.OperateKong, Card: c}
			if room.CanPlay(OptYiKouXiang) && !other.isReadyHand { // 一口香
				n := other.handCards[c]
				other.handCards[c] = 0
				if opts := other.CheckWin(); len(opts) == 0 {
					operateTip.IsAbleWin = true
				}
				other.handCards[c] = n
			}
			tips = append(tips, operateTip)

			kongCard := c
			other.Timeout(func() { other.Kong(kongCard) })
		}
		// 胡
		if room.isAbleBoom && other.localObj.IsAbleWin() {
			room.expectWinPlayers[other.Id] = other
			tips = append(tips, OperateTip{Type: mjutils.OperateWin, Card: c})
			other.Timeout(func() { other.Win() })
		}

		other.operateTips = tips
		other.readyHandTips = nil
		log.Infof("discard tips %v", tips)
		if len(tips) > 0 && !other.isAutoPlay {
			isTip = true
		}
		other.Prompt()
	}
	if !isTip {
		room.Turn()
	} else {
		room.Timing()
	}
}

func (ply *MahjongPlayer) Pass() {
	room := ply.Room()
	if room == nil {
		return
	}

	// 玩家不可吃、碰、杠、胡
	if _, ok := room.expectWinPlayers[ply.Id]; !(ok || room.expectKongPlayer == ply || room.expectPongPlayer == ply || room.expectChowPlayer == ply || ply.expectReadyHand) {
		return
	}
	log.Debugf("player %d pass", ply.Id)
	// OK
	utils.StopTimer(ply.operateTimer)
	ply.WriteErr("pass", nil)

	ply.delayChow = false
	ply.delayPong = false
	ply.delayKong = false

	ply.operateTips = nil
	ply.readyHandTips = nil
	ply.expectReadyHand = false
	if _, ok := room.expectWinPlayers[ply.Id]; ok {
		ply.isPassWin = true
		ply.unableWinCards[room.lastCard] = true
	}
	delete(room.expectWinPlayers, ply.Id)
	if room.expectChowPlayer == ply {
		room.expectChowPlayer = nil
	}
	if room.expectPongPlayer == ply {
		room.expectPongPlayer = nil
	}
	if room.expectKongPlayer == ply {
		room.expectKongPlayer = nil
	}
	// 没人胡
	if len(room.expectWinPlayers) == 0 && len(room.winPlayers) == 0 {
		if p := room.kongPlayer; p != nil && p.robKong {
			p.KongOk()
		} else if p := room.expectKongPlayer; p != nil && p.delayKong {
			p.Kong(room.lastCard)
		} else if p := room.expectPongPlayer; p != nil && p.delayPong {
			p.Pong()
		} else if p := room.expectChowPlayer; p != nil && p.delayChow {
			p.Chow(p.chowCard)
		} else {
			// 没人吃、碰、杠、胡
			if room.expectKongPlayer == nil &&
				room.expectPongPlayer == nil &&
				room.expectChowPlayer == nil {
				// 轮到你出牌
				if room.expectDiscardPlayer == ply {
					ply.Timeout(func() { ply.autoDiscard() })
				} else {
					room.Turn()
				}
			}
		}
	} else {
		// 离放炮最近的人胡
		if ply.drawCard == -1 && room.CanPlay(OptFangPaoJiuJinHu) {
			somebody := make([]*MahjongPlayer, 0, 8)
			somebody = append(somebody, room.winPlayers...)
			for _, other := range room.expectWinPlayers {
				somebody = append(somebody, other)
			}

			last := roomutils.NoSeat
			boom := room.boomPlayer()
			for _, other := range somebody {
				if last == roomutils.NoSeat || room.distance(boom.GetSeatIndex(), last) > room.distance(boom.GetSeatIndex(), other.GetSeatIndex()) {
					last = other.GetSeatIndex()
				}
			}
			if other := room.GetPlayer(last); other != nil {
				if _, ok := room.expectWinPlayers[other.Id]; !ok {
					room.winPlayers = []*MahjongPlayer{other}
					for _, other := range room.expectWinPlayers {
						other.Pass()
					}
				}
			}
		}

		room.OnWin()
	}
}

func (ply *MahjongPlayer) operateDuration() time.Duration {
	room := ply.Room()
	operateTime := MaxOperateTime
	_, isAbleWin := room.expectWinPlayers[ply.Id]
	if ply.isReadyHand {
		operateTime = MaxAutoPlayTime
	}
	if isAbleWin && room.CanPlay(OptAbleDouble) {
		operateTime = MaxOperateTime
	}

	if room.expectKongPlayer == ply {
		operateTime = MaxOperateTime
	}
	if room.CanPlay(OptBiHu) && isAbleWin {
		operateTime = MaxAutoPlayTime
	}
	return operateTime
}

func (ply *MahjongPlayer) Timeout(f func()) {
	room := ply.Room()
	if room.IsTypeScore() {
		_, ableWin := room.expectWinPlayers[ply.Id]
		if !(room.CanPlay(OptBiHu) && ableWin) {
			// 未听(胡)牌
			if !ply.isReadyHand {
				return
			}
			// 听(胡)牌后且不可杠牌或第一次胡牌，系统自动出牌
			if ply.isReadyHand && ((ableWin && !ply.isWin) || room.expectKongPlayer == ply) {
				return
			}
		}
	}

	operateTime := ply.operateDuration()
	if ply.isBustOrNot {
		operateTime = MaxBustTime
	}
	if ply.isAutoPlay {
		operateTime = MaxAutoPlayTime
	}
	log.Info("time out", operateTime)
	ply.operateTimer = ply.TimerGroup.NewTimer(f, operateTime)
}

func (ply *MahjongPlayer) GameOver() {
	utils.StopTimer(ply.operateTimer)
	ply.StartGame()
}

func (ply *MahjongPlayer) autoDiscard() {
	room := ply.Room()
	log.Infof("player %d auto discard, draw card %d", ply.Id, ply.drawCard)
	if room.expectDiscardPlayer != ply {
		return
	}
	ply.expectReadyHand = false
	if !ply.isAutoPlay && !ply.isReadyHand {
		ply.isAutoPlay = true
		room.Broadcast("autoPlay", map[string]any{"code": errcode.CodeOk, "type": 1, "uid": ply.Id})
	}
	if ply.drawCard != -1 {
		ply.Discard(ply.drawCard)
	} else {
		for c, n := range ply.handCards {
			if n > 0 && utils.InArray(ply.blackList, c) == 0 {
				ply.Discard(c)
				return
			}
		}
	}
}

func (ply *MahjongPlayer) ChangeRoom() {
	room := ply.Room()
	log.Infof("player %d change room", ply.Id)

	clone := ply
	if code := ply.TryLeave(); code != nil {
		return
	}
	// TODO 待实现
	if room != nil && room.Status != 0 {
		// clone = ply.Clone()
		log.Debug("TODO")
	}

	code := roomutils.GetRoomObj(ply.Player).ChangeRoom()
	// TODO 待实现
	// clone.WriteErr("ChangeRoom", code, "isMatch", clone.Room() && clone.Room().IsMatchOk)
	if code != nil {
		return
	}

	roomutils.GetRoomObj(ply.Player).PrepareClone()
	roomutils.GetRoomObj(clone.Player).CancelClone()
	utils.StopTimer(clone.operateTimer)

	clone.StartGame()
	clone.OnEnter()
	roomutils.GetRoomObj(clone.Player).Ready()
}

func (ply *MahjongPlayer) AutoPlay(t int) {
	room := ply.Room()
	if ply.isAutoPlay == (t != 0) {
		return
	}
	if ply.isReadyHand {
		return
	}
	if room.Status == 0 {
		return
	}

	isAutoPlay := ply.isAutoPlay
	response := map[string]any{"code": errcode.CodeOk, "Type": t, "uid": ply.Id}
	room.Broadcast("AutoPlay", response)
	// 玩家选择托管，立即操作
	d := time.Duration(0)
	if isAutoPlay {
		room.deadline = room.autoTime
		room.notifyClock()
		d = time.Until(room.autoTime)
	}
	ply.isAutoPlay = (t != 0)
	if ply.leaveGame {
		return
	}
	utils.ResetTimer(ply.operateTimer, d)
}

func (ply *MahjongPlayer) GetWinOptions() {
	opts := ply.CheckWin()
	ply.WriteJSON("getWinOptions", map[string]any{"detail": opts})
}

// 换三张
func (ply *MahjongPlayer) ExchangeTriCards(tri [3]int) {
	cards := ply.handCards
	counter := make([]int, MaxCard)
	// 必须三张相同花色的牌
	for _, c := range tri {
		if !cardutils.IsCardValid(c) || tri[0]/10 != c/10 {
			return
		}
		counter[c]++
	}

	for _, c := range tri {
		if cards[c] < counter[c] {
			return
		}
	}

	room := ply.Room()
	if room.Status != roomStatusExchangeTriCards {
		return
	}
	ply.triCards = tri
	ply.WriteJSON("exchangeTriCards", map[string]any{"code": errcode.CodeOk, "triCards": tri, "uid": ply.Id})
	room.Broadcast("exchangeTriCards", map[string]any{"code": errcode.CodeOk, "triCards": [3]int{}, "uid": ply.Id}, ply.Id)
	room.OnExchangeTriCards()
}

func (ply *MahjongPlayer) ChooseColor(color int) {
	log.Infof("player %d choose color %d", ply.Id, color)
	if ply.discardColor != -1 {
		return
	}
	if !cardutils.IsColorValid(color) {
		return
	}
	room := ply.Room()
	if room.Status != roomStatusChooseColor {
		return
	}
	ply.discardColor = color
	ply.WriteJSON("chooseColor", map[string]any{"code": errcode.CodeOk, "color": color, "uid": ply.Id})
	room.Broadcast("chooseColor", map[string]any{"code": errcode.CodeOk, "color": -1, "uid": ply.Id}, ply.Id)
	room.OnChooseColor()
}

func (ply *MahjongPlayer) CheckReadyHand() []ReadyHandOption {
	room := ply.Room()
	cards := ply.copyCardsWithoutNoneCard()

	type MahjongScorer interface {
		Score(cards []int, melds []mjutils.Meld) (int, int)
	}

	opts := make([]ReadyHandOption, 0, 16)
	for _, discard := range cardutils.GetAllCards() {
		if cards[discard] > 0 {
			cards[discard]--

			opt := ReadyHandOption{DiscardCard: discard}
			for _, add := range cardutils.GetAllCards() {
				cards[add]++

				// 定缺花色不存在
				if !HasColor(cards, ply.discardColor) {
					if winOpt := room.helper.Win(cards, ply.melds); winOpt != nil {
						tempOpt := *winOpt
						tempOpt.WinCard = add

						if scorer, ok := room.localMahjong.(MahjongScorer); ok {
							tempOpt.Chip, tempOpt.Points = scorer.Score(cards, ply.melds)
						}
						opt.WinOptions = append(opt.WinOptions, tempOpt)
					}
				}
				cards[add]--
			}
			cards[discard]++
			if len(opt.WinOptions) > 0 {
				opts = append(opts, opt)
			}
		}
	}
	return opts
}

// 听牌
func (ply *MahjongPlayer) ReadyHand() []ReadyHandOption {
	opts := ply.localObj.CheckReadyHand()
	// 优化胡任意牌
	for i, opt := range opts {
		table := make(map[int]bool)
		for _, winOpt := range opt.WinOptions {
			wc := winOpt.WinCard
			table[wc] = true
		}

		isWinNoneCard := true
		for _, c := range cardutils.GetAllCards() {
			if b, ok := table[c]; !ok || !b {
				isWinNoneCard = false
			}
		}
		if isWinNoneCard {
			for _, winOpt := range opt.WinOptions {
				winOpt.WinCard = NoneCard
				opts[i].WinOptions = []WinOption{winOpt}
				break
			}
		}
	}
	return opts
}

func (ply *MahjongPlayer) IsAbleWin() bool {
	cards := ply.handCards
	if ply.leaveGame {
		return false
	}

	if HasColor(cards, ply.discardColor) {
		return false
	}
	room := ply.Room()
	winCard := room.lastCard
	if _, ok := ply.unableWinCards[winCard]; ok {
		return false
	}

	// PrintCards(cards)
	opts := ply.ReadyHand()
	for _, opt := range opts {
		dc := opt.DiscardCard
		for _, winOpt := range opt.WinOptions {
			wc := winOpt.WinCard
			if dc == wc || wc == NoneCard || dc == NoneCard {
				return true
			}
		}
	}
	return false
}

func (ply *MahjongPlayer) CheckWin() []WinOption {
	room := ply.Room()
	opts := ply.ReadyHand()
	for _, opt := range opts {
		if opt.DiscardCard == room.lastCard {
			return opt.WinOptions
		}
	}
	return nil
}

// start 吃后顺子开始的牌
func (ply *MahjongPlayer) IsAbleChow(start int) bool {
	// 已胡牌
	if ply.isReadyHand {
		return false
	}
	room := ply.Room()

	// 无效的牌
	if !cardutils.IsCardValid(start) {
		return false
	}
	// 玩法不可吃
	if !room.CanPlay(OptAbleChow) {
		return false
	}
	// 没有出牌或自己出牌出牌的玩家不是上家
	lastId := (ply.GetSeatIndex() + room.NumSeat() - 1) % room.NumSeat()
	if p := room.discardPlayer; p == nil || p == ply || lastId != p.GetSeatIndex() {
		return false
	}
	dc := room.lastCard
	cards := ply.handCards
	cards[dc]++
	chow := cards[start] > 0 && cards[start+1] > 0 && cards[start+2] > 0
	cards[dc]--
	// 牌不够
	if !chow {
		return false
	}
	// 吃完牌以后没牌打了
	var blackList = []int{dc}
	if dc == start && cardutils.IsCardValid(start+3) {
		blackList = append(blackList, start+3)
	}
	if dc == start+2 && cardutils.IsCardValid(start-1) {
		blackList = append(blackList, start-1)
	}

	var counter int
	cards[dc]++
	cards[start]--
	cards[start+1]--
	cards[start+2]--
	for _, c := range cardutils.GetAllCards() {
		if cards[c] > 0 && utils.InArray(blackList, c) == 0 {
			counter++
		}
	}
	cards[dc]--
	cards[start]++
	cards[start+1]++
	cards[start+2]++

	log.Debug("test blacklist", blackList, counter)
	return counter != 0
}

func (ply *MahjongPlayer) IsAblePong() bool {
	room := ply.Room()

	if ply.isReadyHand {
		return false
	}
	if p := room.discardPlayer; p == nil || p == ply {
		return false
	}

	dc := room.lastCard
	if ply.handCards[dc] < 2 {
		return false
	}
	// 赖子牌不能碰
	if room.IsAnyCard(dc) {
		return false
	}
	return true
}

func (ply *MahjongPlayer) lastMeld() mjutils.Meld {
	n := len(ply.melds)
	return ply.melds[n-1]
}

func (ply *MahjongPlayer) OnDiscard() {
}

func (ply *MahjongPlayer) OnChow() {
}

func (ply *MahjongPlayer) OnPong() {
}

func (ply *MahjongPlayer) OnKong() {
}

func (ply *MahjongPlayer) OnDouble() {
}

func (ply *MahjongPlayer) copyCards() []int {
	return ply.copyCardsWithNoneCard(true)
}

func (ply *MahjongPlayer) copyCardsWithoutNoneCard() []int {
	return ply.copyCardsWithNoneCard(false)
}

// 手牌副本
func (ply *MahjongPlayer) copyCardsWithNoneCard(hasNoneCard bool) []int {
	room := ply.Room()
	cards := make([]int, MaxCard)
	for _, c := range cardutils.GetAllCards() {
		cards[c] = ply.handCards[c]
	}
	// 玩家不可出牌
	if room.expectDiscardPlayer != ply {
		cards[room.lastCard]++
	}
	if hasNoneCard {
		for _, c := range room.GetAnyCards() {
			cards[NoneCard] += cards[c]
			cards[c] = 0
		}
	}

	return cards
}

// TODO 待实现
/*
func (ply *MahjongPlayer) Replay(messageId string, i any) {
	switch messageId {
	case "Draw":
		data := i.(map[string]any)
		c := data["card"].(int)
		uid := data["uid"].(int)
		if other := GetPlayer(uid); other != nil {
			data["card"] = other.drawCard
		}
		defer func() { data["card"] = c }()
	case "DealCard":
		room := ply.Room()
		data := i.(map[string]any)
		all := make([][]int, room.NumSeat())
		for k := 0; k < room.NumSeat(); k++ {
			other := room.GetPlayer(k)
			all[k] = SortCards(other.handCards)
		}
		data["All"] = all
		defer func() { delete(data, "All") }()
	case "FinishExchangeTriCards":
		room := ply.Room()
		data := i.(map[string]any)
		all := make([][]int, room.NumSeat())
		for k := 0; k < room.NumSeat(); k++ {
			other := room.GetPlayer(k)
			all[k] = other.triCards[:]
		}
		data["All"] = all
		defer func() { delete(data, "All") }()
	}
	ply.Player.Replay(messageId, i)
}
*/

// 杠上炮
func (ply *MahjongPlayer) IsWinAfterOtherKong() bool {
	room := ply.Room()
	if room.kongPlayer != nil && room.discardPlayer == room.kongPlayer {
		return true
	}
	return false
}

// 杠上花
func (ply *MahjongPlayer) IsDrawAfterKong() bool {
	room := ply.Room()
	if ply.drawCard != -1 && room.kongPlayer == ply {
		return true
	}
	return false
}

// 抢杠胡
func (ply *MahjongPlayer) IsRobKong() bool {
	room := ply.Room()
	if room.kongPlayer != nil && room.kongPlayer != ply && room.discardPlayer == nil {
		return true
	}
	return false
}

func (ply *MahjongPlayer) notifyClock() {
	seatId := -1
	room := ply.Room()
	if p := room.GetActivePlayer(); p != nil {
		seatId = p.GetSeatIndex()
	}
	ply.WriteJSON("timing", map[string]any{"seatIndex": seatId, "ts": room.deadline.Unix()})
}

// TODO 加倍当前仅考虑二人
func (ply *MahjongPlayer) Double() {
	room := ply.Room()
	// 可听牌
	if !room.CanPlay(OptAbleDouble) {
		return
	}
	// 可胡牌
	if _, ok := room.expectWinPlayers[ply.Id]; !ok {
		return
	}
	// OK
	ply.localObj.OnDouble()
	if ply.forceReadyHand && !ply.isReadyHand {
		ply.expectReadyHand = true
	}
	if ply.isReadyHand {
		ply.rhHistory = append(ply.rhHistory, room.lastCard)
	}

	ply.multiples *= 2
	ply.forceReadyHand = false
	response := map[string]any{
		"uid":      ply.Id,
		"Multiple": ply.multiples,
	}
	room.Broadcast("Double", response)
	if ply.expectReadyHand {
		ply.ReadyHandOk()
	}

	room.expectChowPlayer = nil
	room.expectPongPlayer = nil
	room.expectKongPlayer = nil
	delete(room.expectWinPlayers, ply.Id)

	ply.operateTips = nil
	ply.readyHandTips = nil
	utils.StopTimer(ply.operateTimer)

	c := ply.drawCard
	if c == -1 {
		room.Turn()
	} else if ply.isReadyHand {
		ply.Discard(c)
	} else {
		// 超时出牌
		ply.Timeout(func() { ply.autoDiscard() })

		var tips []OperateTip
		for _, c1 := range cardutils.GetAllCards() {
			if ply.localObj.GetKongType(c1) != -1 {
				tips = append(tips, OperateTip{Type: mjutils.OperateKong, Card: c1})
				room.expectKongPlayer = ply
				kc := c1
				ply.Timeout(func() { ply.Kong(kc) })
			}
		}

		if room.CanPlay(OptBaoTing) && !ply.isReadyHand {
			opts := ply.ReadyHand()
			if len(opts) > 0 {
				ply.expectReadyHand = true
				tips = append(tips, OperateTip{Type: mjutils.OperateReadyHand})
			}
		}

		room.Timing()

		ply.operateTips = tips
		if !ply.isReadyHand {
			ply.readyHandTips = ply.ReadyHand()
		}
		ply.Prompt()
	}
}

type OtherInfo struct {
	SeatId int
	Cards  []int
}

func (ply *MahjongPlayer) GetOthers() []*OtherInfo {
	room := ply.Room()
	users := make([]*OtherInfo, 0, 4)
	for i := 0; i < room.NumSeat(); i++ {
		if other := room.GetPlayer(i); ply != other && ply != nil {
			cards := SortCards(other.handCards)
			users = append(users, &OtherInfo{SeatId: i, Cards: cards})
		}
	}
	return users
}

func (ply *MahjongPlayer) ReadyHandOk() {
	room := ply.Room()
	if ply.isReadyHand {
		return
	}
	if !ply.expectReadyHand {
		return
	}
	opts := ply.CheckWin()
	response := map[string]any{
		"uid": ply.Id,
	}
	room.Broadcast("readyHand", response, ply.Id)

	response["winOpts"] = opts
	if room.CanPlay(OptAbleLookOthersAfterReadyHand) {
		ply.isAbleLookOthers = true
		response["others"] = ply.GetOthers()
	}
	ply.isReadyHand = true
	ply.expectReadyHand = false
	ply.WriteJSON("readyHand", response)
}
