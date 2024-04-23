// 2018-10-29
// 增加抢地主，区别于叫地主第一个叫地主的就是地主，抢地主可以再抢
package internal

import (
	"gofishing-game/internal/cardutils"
	ddzutils "gofishing-game/migrate/doudizhu/utils"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
	"gofishing-game/service/system"
	"math/rand"
	"quasar/utils/randutils"
	"strconv"
	"strings"
	"time"

	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
)

const (
	OptJiaodizhu  = "jiaodizhu"  // 叫地主
	OptJiaofen    = "jiaofen"    // 叫分
	OptZhadan3    = "zhadan3"    // 三炸
	OptZhadan4    = "zhadan4"    // 四炸
	OptZhadan5    = "zhadan5"    // 五炸
	OptQiangdizhu = "qiangdizhu" // 抢地主
)

var (
	maxAutoTime    = 1500 * time.Millisecond
	maxOperateTime = 16 * time.Second
)

type Bill struct {
	// 炸弹数
	Boom int `json:"boom"`
	// 剩余牌数
	CardNum  int   `json:"cardNum"`
	Cards    []int `json:"cards"`
	Chip     int64 `json:"chip"`
	IsSpring bool  `json:"isSpring"` // 春天

	validChip int64
}

type DoudizhuRoom struct {
	*roomutils.Room

	helper *ddzutils.DoudizhuHelper

	boss                *DoudizhuPlayer // 地主
	dealer, nextDealer  *DoudizhuPlayer // 这里庄家的作用就是先叫底注
	discardPlayer       *DoudizhuPlayer
	expectDiscardPlayer *DoudizhuPlayer
	winPlayer           *DoudizhuPlayer

	jiaodizhuPlayer  *DoudizhuPlayer // 当前叫地主玩家
	jiaofenPlayer    *DoudizhuPlayer // 当前叫分玩家
	qiangdizhuPlayer *DoudizhuPlayer // 当前抢地主玩家

	choosePlayer *DoudizhuPlayer // 叫分最大的玩家

	clientTriCards, triCards [3]int
	autoTime                 time.Time

	currentTimes  int           // 当前的倍数
	boomTimes     int           // 总炸弹次数
	delayDuration time.Duration // 比如开局发牌
	sample        []int
	cheatSeats    int
}

func (room *DoudizhuRoom) OnEnter(player *service.Player) {
	comer := player.GameAction.(*DoudizhuPlayer)
	log.Infof("player %d enter room %d", comer.Id, room.Id)

	// 玩家重连
	data := map[string]any{
		"status":    room.Status,
		"subId":     room.SubId,
		"countdown": room.autoTime.Unix(),
		"triCards":  room.clientTriCards,
	}

	var seats []*DoudizhuUserInfo
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil {
			info := p.GetUserInfo(comer.Id == p.Id)
			seats = append(seats, info)
		}
	}
	data["seatPlayers"] = seats
	if room.dealer != nil {
		data["dealerId"] = room.dealer.Id
	}
	if room.boss != nil {
		data["dizhu"] = room.boss.Id
	}
	data["currentTimes"] = room.currentTimes

	// 玩家可能没座位
	comer.SetClientValue("roomInfo", data)

	if room.Status != 0 {
		room.OnTurn()
	}
}

func (room *DoudizhuRoom) OnLeave(player *service.Player) {
	room.nextDealer = nil
}

func (room *DoudizhuRoom) StartGame() {
	room.Room.StartGame()

	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		p.initGame()
	}
	// 选庄家
	if room.nextDealer != nil {
		room.dealer = room.nextDealer
	}

	room.nextDealer = nil
	// 房主当庄
	host := room.FindPlayer(room.HostSeatIndex()).GameAction.(*DoudizhuPlayer)
	if room.dealer == nil && host != nil && host.Room() == room {
		room.dealer = host
	}
	// 随机
	if room.dealer == nil {
		seatId := rand.Intn(room.NumSeat())
		room.dealer = room.GetPlayer(seatId)
	}
	// room.Broadcast("NewDealer", map[string]any{"uid": room.dealer.Id})
	room.StartDealCard()

	room.currentTimes = 1
	d := room.maxOperateTime()
	room.autoTime = time.Now().Add(d)
	if room.CanPlay(OptJiaofen) {
		room.Jiaofen() // 叫分
	} else if room.CanPlay(OptJiaodizhu) {
		room.Jiaodizhu() // 叫地主
	} else {
		room.Qiangdizhu() // 抢地主
	}
}

// 叫地主，第一个叫地主的玩家成为地主
func (room *DoudizhuRoom) Jiaodizhu() {
	p := room.dealer
	if other := room.jiaodizhuPlayer; other != nil {
		p = room.GetPlayer((other.GetSeatIndex() + 1) % room.NumSeat())
	}
	d := room.maxOperateTime()
	if p.jiaodizhu == -1 {
		room.jiaodizhuPlayer = p
		room.autoTime = time.Now().Add(d)

		room.Broadcast("startJiaodizhu", map[string]any{"sec": room.autoTime.Unix(), "uid": p.Id})
		p.Timeout(func() { p.Jiaodizhu(0) }, d)
	} else {
		room.StartGame()
	}
}

func (room *DoudizhuRoom) OnJiaodizhu() {
	p := room.jiaodizhuPlayer
	if p.jiaodizhu == 1 {
		room.boss = p
		room.jiaodizhuPlayer = nil
		room.StartPlaying()
	} else {
		room.Jiaodizhu()
	}
}

// 抢地主，第一个叫地主的可以再抢
func (room *DoudizhuRoom) Qiangdizhu() {
	p := room.dealer
	if cur := room.qiangdizhuPlayer; cur != nil {
		for i := 0; i+1 < room.NumSeat(); i++ {
			cur = room.GetPlayer((cur.GetSeatIndex() + 1) % room.NumSeat())
			if cur.qiangdizhu != 0 {
				p = cur
				break
			}
		}
	}

	var none, agree int // 未叫；叫
	for i := 0; i < room.NumSeat(); i++ {
		other := room.GetPlayer(i)
		if other.qiangdizhu > 0 {
			agree++
		}
		if other.qiangdizhu == -1 {
			none++
		}
	}
	// 全部选择不叫
	d := room.maxOperateTime()
	if none == 0 && agree == 0 {
		room.StartGame()
	} else {
		// TODO 移除了StartQiangdizhu消息，用Turn代替
		room.qiangdizhuPlayer = p
		room.autoTime = time.Now().Add(d)
		room.OnTurn()
		p.Timeout(func() { p.Qiangdizhu(0) }, d)
	}
}

func (room *DoudizhuRoom) OnQiangdizhu() {
	var none, agree int // 未叫；叫；再叫
	for i := 0; i < room.NumSeat(); i++ {
		other := room.GetPlayer(i)
		if other.qiangdizhu > 0 {
			agree++
		}
		if other.qiangdizhu == -1 {
			none++
		}
	}
	// 仅一个玩家叫地主
	dealer := room.dealer
	if none == 0 && (agree == 1 || dealer.qiangdizhu != 1) {
		for i := 0; i < room.NumSeat(); i++ {
			seatIndex := (dealer.GetSeatIndex() + 1 + i) % room.NumSeat()
			other := room.GetPlayer(seatIndex)
			if other.qiangdizhu > 0 {
				room.boss = other
			}
		}
	}
	if room.boss == nil {
		room.Qiangdizhu()
	} else {
		room.qiangdizhuPlayer = nil
		room.StartPlaying()
	}
}

func (room *DoudizhuRoom) Jiaofen() {
	p := room.dealer

	if other := room.jiaofenPlayer; other != nil {
		p = room.GetPlayer((other.GetSeatIndex() + 1) % room.NumSeat())
	}

	d := room.maxOperateTime()
	if p.jiaofen == -1 {
		room.jiaofenPlayer = p
		room.autoTime = time.Now().Add(d)

		room.Broadcast("startJiaofen", map[string]any{"ts": room.autoTime.Unix(), "uid": p.Id})

		p.Timeout(func() { p.Jiaofen(0) }, d)
	} else {
		room.StartGame()
	}
}

func (room *DoudizhuRoom) OnJiaofen() {
	current := room.jiaofenPlayer
	if other := room.choosePlayer; other == nil || other.jiaofen < current.jiaofen {
		room.choosePlayer = current
	}

	next := room.GetPlayer((current.GetSeatIndex() + 1) % room.NumSeat())
	log.Debug("jiao fen ok", room.choosePlayer.jiaofen, next.jiaofen, next.Id)
	if current.jiaofen == 3 ||
		(room.choosePlayer.jiaofen > 0 && next.jiaofen != -1) {
		room.boss = room.choosePlayer
		room.jiaofenPlayer = nil
		room.StartPlaying()
	} else {
		room.Jiaofen()
	}
}

func (room *DoudizhuRoom) StartPlaying() {
	log.Debug("start playing")
	room.delayDuration = 2 * time.Second
	for i := 0; i < len(room.triCards); i++ {
		c := room.CardSet().Deal()
		room.triCards[i] = c
		room.boss.cards[c]++
	}
	room.Broadcast("StartPlaying", map[string]any{"TriCards": room.triCards, "Dizhu": room.boss.Id})
	room.clientTriCards = room.triCards
	room.expectDiscardPlayer = room.boss
	room.Turn()
}

func (room *DoudizhuRoom) Award() {
	room.nextDealer = room.winPlayer

	unit, _ := config.Int("room", room.SubId, "cost")
	bills := make([]*Bill, room.NumSeat())
	if room.CanPlay(OptJiaofen) {
		unit = unit * int64(room.choosePlayer.jiaofen)
	}
	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		cards := p.GetSortedCards()
		bill := &Bill{
			Cards:   cards,
			Boom:    p.boomTimes,
			CardNum: len(cards),
		}
		bills[i] = bill
		// 地主输了
		if room.winPlayer != room.boss && p == room.boss {
			if p.discardTimes < 2 {
				bill.IsSpring = true
			}
		}
		// 地主赢了
		if room.winPlayer == room.boss && p != room.boss {
			bill.IsSpring = true
			for k := 0; k < room.NumSeat(); k++ {
				other := room.GetPlayer(k)
				if other != room.boss && other.discardTimes > 0 {
					bill.IsSpring = false
				}
			}
		}
	}
	spring := false
	for _, bill := range bills {
		if bill.IsSpring {
			spring = true
			break
		}
	}

	if spring {
		room.currentTimes *= 2
	}
	expectChip := unit * int64(room.currentTimes)

	scale := 1.0
	totalChip := int64(0)
	limitChip := room.boss.BagObj().NumItem(room.GetChipItem())
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != room.boss {
			validChip := expectChip
			if validChip > p.BagObj().NumItem(room.GetChipItem()) && room.IsTypeNormal() {
				validChip = p.BagObj().NumItem(room.GetChipItem())
			}
			totalChip += validChip
			bills[i].validChip = validChip
		}
	}
	if limitChip > totalChip {
		limitChip = totalChip
	}
	if totalChip != 0 {
		scale = float64(limitChip) / float64(totalChip)
	}
	if room.IsTypeScore() {
		scale = 1.0
	}

	floatChip := 0.0
	bossSeatId := room.boss.GetSeatIndex()
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != room.boss {
			validChip := bills[i].validChip
			tempChip := float64(scale)*float64(validChip) + 1e-6
			if room.winPlayer != room.boss {
				p.winTimes++
				tempChip = -tempChip
			}
			floatChip += tempChip
			bills[i].Chip += -int64(tempChip)
		}
	}
	bills[bossSeatId].Chip += int64(floatChip)
	if room.boss == room.winPlayer {
		room.boss.winTimes++
	}

	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		chip := bills[i].Chip
		if p.maxWinChip < chip {
			p.maxWinChip = chip
		}
		p.BagObj().Add(room.GetChipItem(), chip, "ddz_award")
	}

	// room.Status = 0
	room.autoTime = time.Now().Add(room.FreeDuration())
	response := map[string]any{
		"details":      bills,
		"ts":           room.autoTime.Unix(),
		"currentTimes": room.currentTimes,
	}
	room.Broadcast("Award", response)

	room.GameOver()
}

func (room *DoudizhuRoom) GameOver() {
	if room.IsTypeScore() && room.ExistTimes+1 == room.LimitTimes {
		type TotalAwardInfo struct {
			// 炸弹数
			Boom int `json:"boom"`
			// 赢的局数
			WinTimes int `json:"winTimes"`
			// 最大赢取金币
			MaxWinChip int64 `json:"maxWinChip"`
			Chip       int64 `json:"chip"`
		}

		// 积分场最后一局
		details := make([]TotalAwardInfo, 0, 8)
		for i := 0; i < room.NumSeat(); i++ {
			if p := room.GetPlayer(i); p != nil {
				details = append(details, TotalAwardInfo{
					Boom:       p.totalBoomTimes,
					WinTimes:   p.winTimes,
					MaxWinChip: p.maxWinChip,
					Chip:       p.BagObj().NumItem(room.GetChipItem()),
				})
			}
		}
		room.Broadcast("totalAward", map[string]any{"details": details})
	}
	room.Room.GameOver()

	room.dealer = nil
	room.winPlayer = nil
	room.discardPlayer = nil
	room.expectDiscardPlayer = nil
	room.choosePlayer = nil
	room.jiaofenPlayer = nil
	room.jiaodizhuPlayer = nil
	room.boss = nil

	room.currentTimes = 1
	room.boomTimes = 0
	for i := range room.clientTriCards {
		room.clientTriCards[i] = 0
	}
}

func (room *DoudizhuRoom) StartDealCard() {
	percent, _ := config.Float("room", room.SubId, "sampleControlPercent")

	// 发牌
	room.delayDuration = 2 * time.Second
	d := room.maxOperateTime()
	room.autoTime = time.Now().Add(d)

	room.sample = nil
	room.cheatSeats = 0
	// 无测试牌型
	test := cardutils.GetSample()
	if len(test) == 0 && randutils.IsPercentNice(percent) {
		dealerSeatId := room.dealer.GetSeatIndex()
		for i := 0; i < room.NumSeat(); i++ {
			p := room.GetPlayer((dealerSeatId + i + 1) % room.NumSeat())
			if system.GetLoginObj(p.Player).IsRobot() {
				room.cheatSeats = 1 << uint(p.GetSeatIndex())
			}
		}
	}

	rows := 0
	tableName, _ := config.String("Room", room.SubId, "sampleTableName")
	if tableName != "" {
		rows = config.NumRow(tableName)
	}
	if room.cheatSeats != 0 && rows > 0 {
		rowId := config.RowId(rand.Intn(rows))
		s, _ := config.String(tableName, rowId, "Cards")
		for _, v := range strings.Split(s, ",") {
			n, _ := strconv.ParseInt(v, 0, 64)
			room.sample = append(room.sample, int(n))
		}
	}
	log.Infof("room %d load sample %v seats %v", room.Id, room.sample, room.cheatSeats)
	// 将作弊的牌放置到牌堆尾部
	room.CardSet().MoveBack(room.sample)

	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		seatId := p.GetSeatIndex()
		if (1<<uint(seatId))&room.cheatSeats != 0 {
			room.CardSet().MoveFront(room.sample...)
		}

		for k := 0; k < 17; k++ {
			c := room.CardSet().Deal()
			p.cards[c]++
		}
		log.Debug("start deal card", p.GetSortedCards())
	}
	data := map[string]any{
		"ts": room.autoTime.Unix(),
	}
	for i := 0; i < room.NumSeat(); i++ {
		p := room.GetPlayer(i)
		data["Cards"] = p.GetSortedCards()
		p.WriteJSON("startDealCard", data)
	}
}

func (room *DoudizhuRoom) GetPlayer(seatIndex int) *DoudizhuPlayer {
	if seatIndex < 0 || seatIndex >= room.NumSeat() {
		return nil
	}
	if p := room.FindPlayer(seatIndex); p != nil {
		return p.GameAction.(*DoudizhuPlayer)
	}
	return nil
}

func (room *DoudizhuRoom) Turn() {
	current := room.expectDiscardPlayer
	next := current
	if p := room.discardPlayer; p != nil {
		next = room.GetPlayer((current.GetSeatIndex() + 1) % room.NumSeat())
		room.expectDiscardPlayer = next

		// 新的一轮
		if p == next {
			room.discardPlayer = nil
		}
	}
	d := room.maxOperateTime()
	room.autoTime = time.Now().Add(d)
	next.Timeout(next.AutoPlay, d)
	room.OnTurn()

	room.delayDuration = 0 // 下一轮的时候清理延迟
}

func (room *DoudizhuRoom) OnTurn() {
	data := map[string]any{
		"ts": room.autoTime.Unix(),
	}

	current := room.dealer
	if p := room.expectDiscardPlayer; p != nil {
		current = p
	}
	if p := room.jiaodizhuPlayer; p != nil {
		current = p
		data["jiaodizhu"] = true
	}
	if p := room.jiaofenPlayer; p != nil {
		current = p
		data["jiaofen"] = true
		if other := room.choosePlayer; other != nil {
			data["choice"] = other.jiaofen
		}
	}
	if p := room.qiangdizhuPlayer; p != nil {
		current = p
		data["qiangdizhu"] = true
	}

	data["uid"] = current.Id
	if p := room.discardPlayer; p == nil {
		data["newLoop"] = true
	} else {
		if ans := room.helper.Match(current.GetSortedCards(), p.action); len(ans) == 0 {
			data["pass"] = true
		}
	}
	room.Broadcast("turn", data)
}

func (room *DoudizhuRoom) maxOperateTime() time.Duration {
	return maxOperateTime + room.delayDuration
}
