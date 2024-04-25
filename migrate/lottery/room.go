package lottery

import (
	"container/list"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
	"third/cardutil"
	"time"

	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
	"github.com/guogeer/quasar/randutil"
	"github.com/guogeer/quasar/util"
)

const (
	defaultSamples = "685-180-75-56-3-1-0"
)

var gNextTurnType = -1 // 手动控制下一轮结果
var lotteryOdds = []float64{0, 1.25, 5, 12.5, 17.5, 190, 250, 5000}

type UserRecord struct {
	Prize  int64
	Areas  []int64
	Areas2 []int64 // 增加散牌押注区域
	Type   int
	Ts     int64
}

type AwardRecord struct {
	WinUsers int
	BetUsers int
	Prize    int64
	Cards    []int
	Type     int
	Ts       int64
	Users    map[int]UserRecord `json:"-"`
}

type lotteryRoom struct {
	*service.Room

	deadline          time.Time
	helper            *cardutil.ZhajinhuaHelper
	areas, robotAreas [cardutil.ZhajinhuaTypeAll]int64
	history           []int
	awards            list.List
	isFishing         bool // 代号时时乐捕鱼
	chips             []int64
	odds              []float64
}

func (room *lotteryRoom) OnEnter(player *service.Player) {
	room.Room.OnEnter(player)

	comer := player.GameAction.(*lotteryPlayer)
	log.Infof("player %d enter room %d", comer.Id, room.Id)

	// 玩家重连
	data := map[string]any{
		"Status":    room.Status,
		"SubId":     room.SubId,
		"Countdown": room.GetShowTime(room.deadline),
		"History":   room.history,
		"Odds":      room.odds,
	}
	if room.awards.Len() > 0 {
		award := room.awards.Back().Value.(*AwardRecord)
		data["LastRecord"] = award
	}
	if room.Status == service.RoomStatusPlaying {
		data["BetAreas"] = room.areas
	}

	var seats []*lotteryUserInfo
	for i := 0; i < room.NumSeat(); i++ {
		if p := room.GetPlayer(i); p != nil {
			info := p.GetUserInfo(comer.Id)
			seats = append(seats, info)
		}
	}
	data["SeatPlayers"] = seats
	if comer.SeatId == roomutils.NoSeat {
		data["PersonInfo"] = comer.GetUserInfo(comer.Id)
	}
	comer.WriteJSON("GetRoomInfo", data)
}

func (room *lotteryRoom) countPrize(gold int64, typ int) int64 {
	return int64(float64(gold) * room.odds[typ])
}

// 发牌
func (room *lotteryRoom) Award() {
	subId := room.SubId

	defaultCardSamples := util.ParseIntSlice(defaultSamples)
	s, _ := config.String("entertainment", subId, "CardSamples")
	cardSamples := defaultCardSamples
	if s != "" {
		cardSamples = util.ParseIntSlice(s)
	}
	validCards := room.CardSet().GetRemainingCards()
	check := room.GetInvisiblePrizePool().Check()

	var totalBet, robotBet int64
	for _, n := range room.areas {
		totalBet += n
	}
	for _, n := range room.robotAreas {
		robotBet += n
	}

	cards := []int{0x02, 0x15, 0x27}
	ipp := room.GetInvisiblePrizePool()
	for retry := 0; retry < 100; retry++ {
		if retry > 66 {
			cardSamples = defaultCardSamples
		}
		typ := randutil.Index(cardSamples) + 1
		if gNextTurnType > 0 {
			typ = gNextTurnType
		}
		cheatCards := room.helper.Cheat(typ, validCards)
		if len(cheatCards) == 0 {
			continue
		}
		cards = append(cards[:0], cheatCards...)
		if gNextTurnType > 0 {
			break
		}

		userAreaBet := room.areas[typ] - room.robotAreas[typ]
		winGold := room.countPrize(userAreaBet, typ)
		// 暗池不够赔重新发牌
		userWinGold := winGold - totalBet + robotBet
		if !ipp.IsValid(userWinGold) {
			continue
		}
		if check == -1 && userWinGold < 0 {
			break
		}
		if check == 1 && userWinGold > 0 {
			break
		}
		if check == 0 {
			break
		}
	}

	ts := time.Now().Unix()
	typ, _ := room.helper.GetType(cards)

	prize := room.countPrize(room.areas[typ], typ)
	awardData := &AwardRecord{
		Type:  typ,
		Cards: cards,
		Prize: prize,
		Users: make(map[int]UserRecord),
		Ts:    ts,
	}
	pct, _ := config.Float("Room", room.SubId, "TaxPercent")
	for _, player := range room.AllPlayers {
		p := player.GameAction.(*lotteryPlayer)

		userBet := p.areas[0] * 0
		for _, n := range p.areas {
			userBet += n
		}
		obj := p.RoomObj
		obj.BetGold += userBet
		if p.areas[typ] > 0 {
			awardData.WinUsers++

			gold := room.countPrize(p.areas[typ], typ)
			tax := int64(float64(gold-userBet) * pct / 100)
			p.winGold = gold - tax
			obj.WinGold += gold - tax
		}
		if userBet > 0 {
			awardData.BetUsers++
			awardUser := UserRecord{Prize: p.winGold}
			awardUser.Areas = append(awardUser.Areas, p.areas[2:]...)
			awardUser.Areas2 = append(awardUser.Areas2, p.areas[:]...)
			awardData.Users[p.Id] = awardUser
		}
		if p.IsRobot == false {
			room.GetInvisiblePrizePool().Add(p.winGold - userBet)
		}
	}
	room.awards.PushBack(awardData)
	if room.awards.Len() > 40 {
		front := room.awards.Front()
		room.awards.Remove(front)
	}
	room.history = append(room.history, typ)
	if n := 40; len(room.history) > n {
		for i := 0; i < n; i++ {
			room.history[i] = room.history[i+1]
		}
		room.history = room.history[:n]
	}
	log.Infof("room award cards %v type %d", cards, typ)
	room.Sync()

	restartTime := room.RestartTime()
	room.deadline = time.Now().Add(restartTime)
	util.NewTimer(room.StartGame, restartTime)

	result := make([]int, 0, 8)
	for i := 0; i < cardutil.ZhajinhuaTypeAll; i++ {
		result = append(result, i)
	}
	result[1], result[typ] = result[typ], result[1]
	randutil.Shuffle(result[2:])
	room.Broadcast("Award", map[string]any{
		"Sec":    room.GetShowTime(room.deadline),
		"Record": awardData,
		"Result": result[1:],
	})
	// 机器人日志合并
	var robot *lotteryPlayer
	var robotBetNum, robotWinNum int64

	guid := util.GUID()
	tway := service.ItemWay{Way: "sum.lottery_bet", SubId: subId}.String()
	sway := service.ItemWay{Way: "sys.lottery_bet", SubId: subId}.String()
	for _, player := range room.AllPlayers {
		way := sway
		sum := int64(0)
		p := player.GameAction.(*lotteryPlayer)
		for _, bet := range p.areas {
			sum += bet
		}
		if p.IsRobot {
			way = tway
			robot = p
			robotBetNum += sum
		}
		p.AddGoldLog(-sum, guid, way)
	}
	if robot != nil {
		robot.AddGoldLog(-robotBetNum, guid, sway)
	}

	tway = service.ItemWay{Way: "sum.lottery_award", SubId: subId}.String()
	sway = service.ItemWay{Way: "sys.lottery_award", SubId: subId}.String()
	for _, player := range room.AllPlayers {
		way := sway
		p := player.GameAction.(*lotteryPlayer)
		if p.IsRobot {
			way = tway
			robotWinNum += p.winGold
		}
		p.AddGold(p.winGold, guid, way)
	}
	if robot != nil {
		robot.AddGoldLog(robotWinNum, guid, sway)
	}

	room.GameOver()
}

func (room *lotteryRoom) GameOver() {
	room.Room.GameOver()

	for i := range room.areas {
		room.areas[i] = 0
	}
	for i := range room.robotAreas {
		room.robotAreas[i] = 0
	}
	gNextTurnType = -1
}

func (room *lotteryRoom) StartGame() {
	var odds []float64
	var points []string
	var d = 60 * time.Second

	subId := room.SubId
	room.Room.StartGame()
	config.Scan("entertainment", subId,
		"Chips,Odds,UserBetDuration",
		&room.chips, &odds, &points,
	)
	room.odds = append([]float64{0}, odds...)
	if len(room.odds) != len(lotteryOdds) {
		room.odds = lotteryOdds
	}

	if t := service.RandSeconds(points); t > 0 {
		d = t
	}
	room.deadline = time.Now().Add(d)
	room.Broadcast("StartGame", map[string]any{
		"Sec": room.GetShowTime(room.deadline),
	})
	util.NewTimer(room.Award, d)
}

func (room *lotteryRoom) GetPlayer(seatId int) *lotteryPlayer {
	if seatId < 0 || seatId >= room.NumSeat() {
		return nil
	}
	if p := room.SeatPlayers[seatId]; p != nil {
		return p.GameAction.(*lotteryPlayer)
	}
	return nil
}

func (room *lotteryRoom) Sync() {
	sub := room.GetSubWorld()

	total := 0
	for _, player := range room.AllPlayers {
		p := player.GameAction.(*lotteryPlayer)
		for _, n := range p.areas {
			if n > 0 {
				total++
				break
			}
		}
	}
	data := map[string]any{
		"Onlines":   sub.FakeOnline,
		"BetAreas":  room.areas[2:],
		"BetAreas2": room.areas[:], // 增加散牌可押注
		"BetUsers":  total,
	}
	room.Broadcast("Sync", data)
}
