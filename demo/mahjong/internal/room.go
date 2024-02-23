package internal

import (
	mjutils "gofishing-game/demo/mahjong/utils"
	"gofishing-game/internal/cardutils"
	"gofishing-game/internal/errcode"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
	"gofishing-game/service/system"
	"math/rand"
	"quasar/utils"
	"quasar/utils/randutils"
	"strconv"
	"strings"
	"time"

	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
)

const (
	MaxOperateTime    = 14 * time.Second        // 等待时间
	MaxAutoPlayTime   = 1500 * time.Millisecond // 托管、自动出牌
	MaxBustTime       = 90 * time.Second
	maxDelayAfterKong = 1500 * time.Millisecond // 杠后延迟
	maxDelayAfterWin  = 2000 * time.Millisecond // 胡牌后延迟
)

// LocalMahjong 地方麻将
type LocalMahjong interface {
	// 新玩家进入后
	OnEnter(comer *MahjongPlayer)
	// 玩家已准备
	OnReady()
	// 玩家胡牌
	OnWin()
	// 牌局结算
	Award()
	// 牌局结束
	GameOver()
	// 听牌
	// ReadyHand([]int, []Meld) []ReadyHandOption
	// 创建房间
	OnCreateRoom()
}

// Bill 流水清单
type Bill struct {
	// Gold    int64
	Details []ChipDetail
}

// Sum 流水求和
func (bill Bill) Sum() int64 {
	var sum int64
	for _, d := range bill.Details {
		sum += d.Chip
	}
	return sum
}

// MahjongRoom 麻将房间
type MahjongRoom struct {
	*roomutils.Room

	dealer              *MahjongPlayer // 庄家
	nextDealer          *MahjongPlayer // 下把庄家
	expectKongPlayer    *MahjongPlayer
	expectChowPlayer    *MahjongPlayer
	expectPongPlayer    *MahjongPlayer
	expectDiscardPlayer *MahjongPlayer
	expectWinPlayers    map[int]*MahjongPlayer

	kongPlayer    *MahjongPlayer
	discardPlayer *MahjongPlayer
	winPlayers    []*MahjongPlayer

	deadline time.Time
	autoTime time.Time // 取消托管后的截止时间

	lastCard int

	chooseColorTimer      *utils.Timer
	exchangeTriCardsTimer *utils.Timer
	bustTimeout           func()

	buyHorse     int
	localMahjong LocalMahjong
	// 房间录像
	// replay *MahjongReplay
	helper        *mjutils.MahjongHelper
	isAbleBoom    bool             // 可点炮
	delayDuration time.Duration    // 操作延迟
	cheatSeats    int              // 作弊座位ID(可多个)
	sample        cardutils.Sample // 作弊的样例
	isBatch       bool             // 批量通知金币变化
}

func NewMahjongRoom(id, subId int) *MahjongRoom {
	mahjongRoom := &MahjongRoom{
		expectWinPlayers: map[int]*MahjongPlayer{},
	}
	mahjongRoom.Room = roomutils.NewRoom(subId, mahjongRoom)
	return mahjongRoom
}

func (room *MahjongRoom) OnEnter(player *service.Player) {
	comer := player.GameAction.(*MahjongPlayer)
	log.Infof("player %d enter room", comer.Id)

	// 玩家重连
	data := map[string]any{
		"Status":    room.Status,
		"SubId":     room.SubId,
		"CardNum":   room.CardSet().Count(),
		"TotalCard": room.CardSet().Total(),
	}

	var seatPlayers []*MahjongPlayerInfo
	for i := 0; i < room.NumSeat(); i++ {
		if other := room.GetPlayer(i); other != nil {
			info := other.GetUserInfo(comer.Id)
			seatPlayers = append(seatPlayers, info)
		}
	}
	data["SeatPlayers"] = seatPlayers

	dealerId := roomutils.NoSeat
	if room.dealer != nil {
		dealerId = room.dealer.Id
	}
	data["dealerId"] = dealerId
	comer.WriteJSON("getRoomInfo", data)

	// 正在游戏中
	if room.Status == roomutils.RoomStatusPlaying {
		// 金币场破产提示充值
		for i := 0; i < room.NumSeat(); i++ {
			if p := room.GetPlayer(i); p != nil && p.isBustOrNot {
				comer.WriteJSON("bustOrNot", map[string]any{
					"uid": p.Id,
					// "Bust":    p.BaseObj().GetBustInfo(),
					"ts": room.deadline.Unix(),
				})
			}
		}
		comer.Prompt()
		comer.notifyClock()
	}

	room.localMahjong.OnEnter(comer)
}

func (room *MahjongRoom) StartGame() {
	log.Debugf("mahjong room %d start game", room.Id)
	room.Room.StartGame()
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil {
			p.StartGame()
		}
	}
	room.ChooseDealer()

	room.delayDuration = 0
	room.localMahjong.OnReady()
}

func (room *MahjongRoom) OnCreate() {
	room.isAbleBoom = room.CanPlay(OptBoom) // 可点炮

	helper := &mjutils.MahjongHelper{}
	helper.Qidui = room.CanPlay(OptSevenPairs)
	helper.Qiduilaizizuodui = room.CanPlay(OptQiDuiLaiZiZuoDui)
	helper.Jiangyise = room.CanPlay(OptJiangYiSe)
	helper.Shisanyao = room.CanPlay(OptShiSanYao)
	room.helper = helper

	room.CardSet().Recover(cardutils.GetAllCards()...)
	room.localMahjong.OnCreateRoom()
}

// 推选庄家
func (room *MahjongRoom) ChooseDealer() {
	// 推选庄家
	if room.dealer != nil {
		// 连庄
		if room.dealer == room.nextDealer {
			room.dealer.continuousDealerTimes++
		} else {
			room.dealer.continuousDealerTimes = 0
		}
	}

	if room.nextDealer != nil {
		room.dealer = room.nextDealer
	}
	room.nextDealer = nil
	// 优先考虑房主
	host := room.GetPlayer(room.HostSeatIndex())
	if room.dealer == nil && host != nil && host.Room() == room {
		room.dealer = host
	}
	if room.dealer == nil {
		seatId := rand.Intn(room.NumSeat())
		room.dealer = room.GetPlayer(seatId)
	}
	room.Broadcast("NewDealer", map[string]any{"uid": room.dealer.Id})
}

// 开始发牌
func (room *MahjongRoom) StartDealCard() {
	robotSeats := make([]int, 0, 4)
	percent, _ := config.Float("Room", room.SubId, "SampleControlPercent")
	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		if system.GetLoginObj(p.Player).IsRobot() {
			robotSeats = append(robotSeats, i)
		}
	}
	room.sample = nil
	room.cheatSeats = 0
	// 无测试牌型
	test := cardutils.GetSample()
	if len(test) == 0 && len(robotSeats) > 0 && randutils.IsPercentNice(percent) {
		seatId := robotSeats[rand.Intn(len(robotSeats))]
		room.cheatSeats = 1 << uint(seatId)
	}
	rows := 0
	tableName, _ := config.String("Room", room.SubId, "SampleTableName")
	if tableName != "" {
		rows = config.NumRow(tableName)
	}
	if room.cheatSeats != 0 && rows > 0 {
		rowId := config.RowId(rand.Intn(rows))
		s, _ := config.String(tableName, rowId, "Cards")
		for _, v := range strings.Split(s, ",") {
			n, _ := strconv.Atoi(v)
			room.sample = append(room.sample, n)
		}
	}
	log.Infof("room %d load sample %v seats %v", room.Id, room.sample, room.cheatSeats)
	// 将作弊的牌放置到牌堆尾部
	room.CardSet().MoveBack(room.sample)
	// 发牌
	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer((i + room.dealer.GetSeatIndex()) % room.NumSeat())
		for i := 0; i < 13; i++ {
			c := p.TryDraw()
			p.handCards[c]++
		}
	}
	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		p.WriteJSON("DealCard", map[string]any{"Cards": SortCards(p.handCards)})
	}
	room.helper.AnyCards = room.GetAnyCards()

	// 庄家摸牌
	dealer := room.dealer
	c := room.dealer.TryDraw()
	dealer.drawCard = c
	dealer.handCards[c]++
	dealer.WriteJSON("Draw", map[string]any{"card": c, "uid": dealer.Id})
	room.Broadcast("Draw", map[string]any{"card": 0, "uid": dealer.Id}, dealer.Id)
}

func (room *MahjongRoom) StartExchangeTriCards() {
	room.deadline = time.Now().Add(MaxOperateTime)
	room.Status = roomStatusExchangeTriCards
	// 默认换三张
	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)

		var colors [8]int
		for _, c := range cardutils.GetAllCards() {
			colors[c/10] += p.handCards[c]
		}
		color, num := 0, 64
		for _, k := range []int{0, 2, 4} {
			if num > colors[k] && colors[k] > 2 {
				color = k
				num = colors[k]
			}
		}
		num = 0
		for _, c := range cardutils.GetAllCards() {
			for i := 0; c/10 == color && num < 3 && i < p.handCards[c]; i++ {
				p.defaultTriCards[num] = c
				num++
			}
		}
		p.WriteJSON("StartExchangeTriCards", map[string]any{"ts": room.deadline.UTC(), "TriCards": p.defaultTriCards})
	}

	room.exchangeTriCardsTimer = utils.NewTimer(func() {
		if room.IsTypeScore() {
			return
		}
		for i := 0; i < room.NumSeat(); i++ {
			p := room.GetPlayer(i)
			if p.triCards[0] > 0 {
				continue
			}
			p.ExchangeTriCards(p.defaultTriCards)
		}
	}, MaxOperateTime)
}

func (room *MahjongRoom) OnExchangeTriCards() {
	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		if p.triCards[0] == 0 {
			return
		}
	}

	// OK
	utils.StopTimer(room.exchangeTriCardsTimer)
	var exchangeOpts = [4][4]int{{0, 0, 0, 0}, {1, 2, 3, 0}, {2, 3, 0, 1}, {3, 0, 1, 2}}
	dict := rand.Intn(3) + 1
	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		otherSeatId := exchangeOpts[dict][p.GetSeatIndex()]
		other := room.GetPlayer(otherSeatId)
		for i := 0; i < 3; i++ {
			p.handCards[p.triCards[i]]--
			p.handCards[other.triCards[i]]++
		}
		if dealer := room.dealer; dealer.handCards[dealer.drawCard] == 0 {
			p.drawCard = other.triCards[0]
		}
		p.WriteJSON("FinishExchangeTriCards", map[string]any{"TriCards": p.triCards, "OtherTriCards": other.triCards, "Dict": dict})
	}

	room.StartChooseColor()
}

func (room *MahjongRoom) StartChooseColor() {
	// 默认定缺花色
	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		var colors [8]int
		for _, c := range cardutils.GetAllCards() {
			colors[c/10] += p.handCards[c]
		}
		color, num := 0, 64
		for _, k := range []int{0, 2, 4} {
			if num > colors[k] {
				color = k
				num = colors[k]
			}
		}
		p.defaultColor = color
	}

	room.Status = roomStatusChooseColor
	room.deadline = time.Now().Add(MaxOperateTime)
	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		color := p.defaultColor
		p.WriteJSON("StartChooseColor", map[string]any{"ts": room.deadline.Unix(), "Color": color})
	}

	room.chooseColorTimer = utils.NewTimer(func() {
		if room.IsTypeScore() {
			return
		}
		for i := 0; i < room.NumSeat(); i++ {
			p := room.GetPlayer(i)
			if p.discardColor != -1 { // 已定缺
				continue
			}
			color := p.defaultColor
			p.ChooseColor(color) // 默认选择最少的一门花色
		}
	}, MaxOperateTime)
}

func (room *MahjongRoom) OnChooseColor() {
	var colors [4]int
	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		if p.discardColor == -1 {
			return
		}
		colors[i] = p.discardColor
	}
	utils.StopTimer(room.chooseColorTimer)
	room.Broadcast("FinishChooseColor", map[string]any{"Colors": colors})
	room.Status = roomutils.RoomStatusPlaying
	room.dealer.OnDraw()
}

func (room *MahjongRoom) OnLeave(player *service.Player) {
	// seatId := ply.RoomObj.SeatId
	// room.Room.OnLeave(player)

	p := player.GameAction.(*MahjongPlayer)
	if room.nextDealer == p {
		room.nextDealer = nil
	}
	if room.dealer == p {
		room.dealer = nil
	}
}

func (room *MahjongRoom) GetActivePlayer() *MahjongPlayer {
	act := room.kongPlayer
	if p := room.discardPlayer; p != nil {
		act = p
	}
	if p := room.expectDiscardPlayer; p != nil {
		act = p
	}
	return act
}

func (room *MahjongRoom) Timing() {
	d := MaxOperateTime
	act := room.GetActivePlayer()
	if act.isAutoPlay && act == room.expectDiscardPlayer {
		d = MaxAutoPlayTime
	}

	room.deadline = time.Now().Add(d)
	room.autoTime = time.Now().Add(MaxOperateTime)
	room.notifyClock()
}

func (room *MahjongRoom) notifyClock() {
	seatId := -1
	if p := room.GetActivePlayer(); p != nil {
		seatId = p.GetSeatIndex()
	}
	room.Broadcast("timing", map[string]any{"seatIndex": seatId, "ts": room.deadline})
}

// 有人胡牌
func (room *MahjongRoom) OnWin() {
	// 没人胡牌
	if len(room.winPlayers) == 0 {
		return
	}
	// 没人可胡牌
	if len(room.expectWinPlayers) > 0 {
		return
	}

	var winList []int
	for _, p := range room.winPlayers {
		winList = append(winList, p.Id)
	}
	room.Broadcast("WinOk", map[string]any{
		"List": winList,
	})

	// 抢杠胡
	if p := room.kongPlayer; p != nil {
		last := p.lastKong
		if p.robKong && last.Type == mjutils.MeldBentKong {
			for k, m := range p.melds {
				if m.Card == last.Card {
					m.Type = mjutils.MeldTriplet
					p.melds[k] = m
				}
			}
			p.robKong = false
			p.operateTips = nil
			// 被抢杠胡后，取消玩家操作
			utils.StopTimer(p.operateTimer)
			code := errcode.New("other_rob_kong", "rob kong")
			room.Broadcast("kong", map[string]any{"code": code, "msg": code.Error(), "card": last.Card, "type": last.Type, "uid": p.Id})
		}
	}
	// 去掉出牌纪录
	if p := room.discardPlayer; p != nil {
		h := &p.discardHistory
		if len(*h) > 0 {
			*h = (*h)[:len(*h)-1]
		}
	}

	// 放炮的玩家
	boom := room.boomPlayer()
	// 别人放炮，一炮多响算一次
	if p := room.winPlayers[0]; boom != nil && p.drawCard == -1 {
		boom.totalTimes["dp"]++
	}
	for _, p := range room.winPlayers {
		if p.drawCard == -1 {
			p.totalTimes["jp"]++
		} else {
			p.totalTimes["zm"]++
		}
	}

	// 当前没有庄家，一炮多响时选择放炮的人当庄，其他情况选择胡牌的人当庄
	if room.nextDealer == nil {
		if len(room.winPlayers) > 1 {
			room.nextDealer = boom
		} else {
			room.nextDealer = room.winPlayers[0]
		}
	}

	room.localMahjong.OnWin()

	for _, p := range room.winPlayers {
		p.continuousKong = nil
	}
	room.kongPlayer = nil
}

// 一轮结算，比如胡牌（包括一炮多响），下雨等。
// 计算金币时，扣钱的玩家需要考虑携钱（金币）是否足够？
// 当前计算方式是根据需要输钱玩家的记录来计算出赢钱玩家的
// 实际所得
func (room *MahjongRoom) Billing(bills []Bill) {
	// log.Info("============== billing =============")
	for i := 0; i < len(bills); i++ {
		bill := &bills[i]

		var failChip, realChip int64
		failChip = bill.Sum()
		realChip = failChip
		// 通过输钱玩家推算，赢钱不考虑
		if failChip >= 0 {
			continue
		}

		p := room.GetPlayer(i)
		// 私人麻将馆金币允许为负
		if !room.IsTypeScore() && failChip+p.BagObj().NumItem(room.GetChipItem()) < 0 {
			realChip = -p.BagObj().NumItem(room.GetChipItem())
		}

		for k := 0; k < len(bill.Details); k++ {
			detail := &bill.Details[k]
			// detail.Seats = 1 << uint(detail.SeatId)
			detail.Chip = int64(float64(detail.Chip*realChip) / float64(failChip))
		}
		// 浮点问题，金币不够摊分金币向下取整时，玩家应该破产但仍可能剩1~2个金币，就干脆送给第一个人吧
		bill.Details[0].Chip += realChip - bill.Sum()

		for _, detail := range bill.Details {
			seatId := detail.GetSeatIndex()
			detail.Chip = -detail.Chip
			// detail.SeatId = p.SeatId
			detail.Seats = 1 << uint(p.GetSeatIndex())
			// bills[seatId].Chip += detail.Chip
			if otherBill := &bills[seatId]; len(otherBill.Details) > 0 {
				head := &(otherBill.Details[0])
				head.Chip += detail.Chip
				// one.Seats |= 1 << uint(p.SeatId)
				head.Seats |= detail.Seats
			} else {
				otherBill.Details = append(otherBill.Details, detail)
			}
		}
	}

	for i := 0; i < room.NumSeat(); i++ {
		bill := &bills[i]
		p := room.GetPlayer(i)
		p.AddChipOrBust(bill.Sum(), service.GetWorld().GetName()+"_award")
		p.AddChipHistory(bill.Details...)
	}
}

func (room *MahjongRoom) Award() {
	room.isBatch = true
	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		p.tempChip = p.BagObj().NumItem(room.GetChipItem())
	}
	room.localMahjong.Award()

	/*guid := utils.GUID()
	itemWay := service.ItemWay{
		Way:    "user." + service.GetName() + "_play",
		SubId:  room.SubId,
		IsTemp: true,
	}
	way := itemWay.String()
	room.isBatch = false
	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		p.OnAddItem(p.AliasId(), p.Chip-p.tempChip, p.Chip, guid, way)
	}
	*/

	// 流局
	if len(room.winPlayers) == 0 {
		room.Broadcast("WinOk", struct{}{})
	}

	room.expectDiscardPlayer = nil
	room.expectKongPlayer = nil
	room.expectPongPlayer = nil
	room.expectChowPlayer = nil
	room.expectWinPlayers = map[int]*MahjongPlayer{}

	room.winPlayers = nil
	room.kongPlayer = nil
	room.discardPlayer = nil

	room.GameOver()
}

func (room *MahjongRoom) Turn() {
	plays := 0 // 游戏中玩家数量
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil && !p.leaveGame {
			plays++
		}
	}
	// 游戏提前结束
	if plays <= 1 {
		room.Award()
		return
	}

	seatId := -1
	if p := room.discardPlayer; p != nil {
		seatId = p.GetSeatIndex()
	}

	// 自摸胡牌的玩家
	// 一炮多响时，离放炮玩家最远胡牌的人
	boom := room.boomPlayer()
	for _, p := range room.winPlayers {
		if seatId == -1 || boom == nil ||
			room.distance(boom.GetSeatIndex(), seatId) < room.distance(boom.GetSeatIndex(), p.GetSeatIndex()) {
			seatId = p.GetSeatIndex()
		}
	}

	oldSeatId := seatId
	for {
		seatId = (seatId + 1) % room.NumSeat()
		// p := room.SeatPlayers[seatId].(*MahjongPlayer)
		p := room.GetPlayer(seatId)
		if !p.leaveGame || seatId == oldSeatId {
			break
		}
	}

	// 自摸胡需去掉摸的牌
	for _, p := range room.winPlayers {
		if p.drawCard != -1 {
			p.handCards[p.drawCard]--
			p.drawCard = -1
		}
	}

	// p := room.SeatPlayers[seatId].(*MahjongPlayer)
	room.GetPlayer(seatId).Draw()
}

func (room *MahjongRoom) GameOver() {
	log.Debug("=========== game over ============")

	room.deadline = time.Now().Add(room.FreeDuration())
	data := map[string]any{
		"ts": room.deadline.UTC(),
	}

	// 显示其他三家手牌
	others := make([][]int, room.NumSeat())
	details := make([][]ChipDetail, room.NumSeat())
	for k := 0; k < room.NumSeat(); k++ {
		if p := room.GetPlayer(k); p != nil {
			others[k] = SortCards(p.handCards)
			details[k] = p.chipHistory

			if c := p.drawCard; c != -1 {
				data["DrawCard"] = c
				data["DrawSeat"] = p.GetSeatIndex()
			}
		}
	}
	data["Details"] = details
	data["HandCards"] = others
	data["LastCard"] = room.lastCard
	if room.CardSet().Count() < room.CardSet().Total() {
		room.Broadcast("Award", data)
	}

	// 积分场最后一局
	if room.IsTypeScore() && room.ExistTimes+1 == room.LimitTimes {
		total := make([]map[string]int, room.NumSeat())
		for k := 0; k < room.NumSeat(); k++ {
			p := room.GetPlayer(k)
			total[k] = p.totalTimes
		}
		room.Broadcast("TotalAward", map[string]any{"Details": total})
	}

	room.Room.GameOver()
	// room.Status = 0
	room.localMahjong.GameOver()
}

// 无金币玩家人数
func (room *MahjongRoom) CountBustPlayers() int {
	// 私人麻将馆不存在金币不够的问题
	if room.IsTypeScore() {
		return 0
	}
	if room.CardSet().Count() == 0 {
		return 0
	}
	counter := 0
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil && p.leaveGame {
			counter++
		}
	}
	if counter+1 == room.NumSeat() {
		return 0
	}

	counter = 0
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil && !p.leaveGame && p.BagObj().NumItem(room.GetChipItem()) <= 0 {
			counter++
		}
	}
	return counter
}

func (room *MahjongRoom) OnBust() {
	if room.CountBustPlayers() > 0 {
		return
	}
	if timeout := room.bustTimeout; timeout != nil {
		room.bustTimeout = nil
		timeout()
	}
}

func (room *MahjongRoom) GetPlayer(seatIndex int) *MahjongPlayer {
	if seatIndex < 0 || seatIndex >= room.NumSeat() {
		return nil
	}
	if player := room.FindPlayer(seatIndex); player != nil {
		return player.GameAction.(*MahjongPlayer)
	}
	return nil
}

// 放炮的玩家
func (room *MahjongRoom) boomPlayer() *MahjongPlayer {
	if p := room.discardPlayer; p != nil {
		return p
	}
	return room.kongPlayer
}

func (room *MahjongRoom) distance(from, to int) int {
	seatNum := room.NumSeat()
	return (to - from + seatNum) % seatNum
}

func (room *MahjongRoom) GetAnyCards() []int {
	// 癞子玩法
	type Laizi interface {
		getAnyCards() []int
	}

	if laizi, ok := room.localMahjong.(Laizi); ok {
		return laizi.getAnyCards()
	}
	return nil
}

func (room *MahjongRoom) IsAnyCard(c int) bool {
	return utils.InArray(room.GetAnyCards(), c) > 0
}

func (room *MahjongRoom) piao() int {
	if room.CanPlay(OptPiao10) {
		return 10
	}
	if room.CanPlay(OptPiao20) {
		return 20
	}
	if room.CanPlay(OptPiao30) {
		return 30
	}
	return 0
}
