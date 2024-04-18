package texas

// 2017-11-16 彩票

import (
	"container/heap"
	"container/list"
	"gofishing-game/service"
	"third/cardutil"
	"third/errcode"
	"third/gameutil"
	"time"

	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/utils"
)

var (
	maxBetTime  = 90 * time.Second
	maxFreeTime = 10 * time.Second
)

const (
	LotteryStatusFree = iota
	LotteryStatusBet
)

const (
	LotteryRankSize = 16
)

type LotteryRecord struct {
	Id        int
	Users     int
	Cards     [5]int
	WinAreaId int
	Award     int64
}

type LotteryUser struct {
	*service.SimpleUserInfo
	areas [cardutil.TexasTypeAll]int64

	pos         int
	WinGold     int64
	lastWinGold int64
}

type LotteryUserHeap []*LotteryUser

func (h LotteryUserHeap) Len() int           { return len(h) }
func (h LotteryUserHeap) Less(i, j int) bool { return h[i].WinGold >= h[j].WinGold }
func (h LotteryUserHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].pos, h[j].pos = i, j
}

func (h *LotteryUserHeap) Push(x interface{}) {
	user := x.(*LotteryUser)
	*h = append(*h, user)
	user.pos = len(*h) - 1
}

func (h *LotteryUserHeap) Pop() interface{} {
	old := *h
	n := len(old)
	user := old[n-1]
	*h = old[:n-1]

	user.pos = -1
	return user
}

type LotterySystem struct {
	autoTimer *util.Timer
	deadline  time.Time

	history *list.List
	areas   [cardutil.TexasTypeAll]int64
	helper  *cardutil.TexasHelper
	cardSet *cardutil.CardSet
	users   map[int]*LotteryUser

	todayRank, yesterdayRank []LotteryUser

	prizePool     int64
	todayRankHeap LotteryUserHeap
	usersInRank   map[int]*LotteryUser

	id     int
	Status int
}

var defaultLotterySystem = NewLotterySystem()

func NewLotterySystem() *LotterySystem {
	sys := &LotterySystem{
		history:       list.New(),
		helper:        cardutil.NewTexasHelper(),
		cardSet:       cardutil.NewCardSet(),
		users:         make(map[int]*LotteryUser),
		usersInRank:   make(map[int]*LotteryUser),
		todayRank:     make([]LotteryUser, 0, LotteryRankSize),
		yesterdayRank: make([]LotteryUser, 0, LotteryRankSize),
	}
	heap.Init(&sys.todayRankHeap)
	return sys
}

func GetLotterySystem() *LotterySystem {
	return defaultLotterySystem
}

func init() {
	sys := GetLotterySystem()
	sys.StartBetting()
	util.NewPeriodTimer(sys.Sync, "2017-11-29", 2*time.Second)
	util.NewPeriodTimer(sys.updateNewDay, "2017-11-17", 24*time.Hour)
}

func (sys *LotterySystem) Bet(ply *TexasPlayer, areaId int, gold int64) errcode.ErrCode {
	if ply.Gold < gold {
		return errcode.MoreGold
	}
	if areaId < 0 || areaId >= len(sys.areas) {
		return errcode.Retry
	}
	if gold < 0 {
		return errcode.Retry
	}
	user := sys.users[ply.Id]
	if user == nil {
		user = sys.usersInRank[ply.Id]
	}
	if user == nil {
		user = &LotteryUser{SimpleUserInfo: ply.SimpleInfo(), pos: -1}
		user.WinGold = ply.BaseObj().Int64("daily.lottery_win")
	}
	if _, ok := sys.users[ply.Id]; !ok {
		sys.users[ply.Id] = user
	}

	user.areas[areaId] += gold
	sys.areas[areaId] += gold
	ply.AddGold(-gold, util.GUID(), "lottery_bet")

	// 判断是否在排行榜
	if _, ok := sys.usersInRank[ply.Id]; !ok {
		sys.usersInRank[ply.Id] = user
		sys.updateHeap(user)
	}
	return errcode.Ok
}

func (sys *LotterySystem) Sync() {
	if sys.Status == LotteryStatusBet {
		service.Broadcast2Game("SyncLottery", map[string]any{"Areas": sys.areas, "Users": len(sys.users)})
	}
}

func (sys *LotterySystem) StartBetting() {
	sys.Status = LotteryStatusBet

	sys.deadline = time.Now().Add(maxBetTime)
	sys.autoTimer = util.NewTimer(sys.Award, maxBetTime)

	now := time.Now()
	date := now.Year()*1000000 + int(now.Month())*100 + now.Day()
	if sys.id == 0 {
		sys.id = 10 * date
	}
	tempDate := sys.id
	for tempDate > 100000000 {
		tempDate /= 10
	}
	if date != tempDate {
		sys.id = 10 * date
	}
	sys.id += 1

	service.Broadcast2Game("StartLottery", map[string]any{"Id": sys.id, "Sec": service.GetShowTime(sys.deadline)})
}

func (sys *LotterySystem) Award() {
	sys.Status = LotteryStatusFree
	sys.deadline = time.Now().Add(maxBetTime)
	sys.autoTimer = util.NewTimer(sys.StartBetting, maxFreeTime)

	var totalGold, userWinGold, sharePrizePool int64
	var allCardType [cardutil.TexasTypeAll]int
	for i := 0; i < config.Row("texas_lottery"); i++ {
		rowId := config.RowId(i)
		cardType, _ := config.Int("texas_lottery", rowId, "ID")
		sample, _ := config.Int("texas_lottery", rowId, "Rate")
		allCardType[int(cardType)] = int(sample)
	}

	var cards [5]int
	sys.cardSet.Shuffle()
	for i := range cards {
		cards[i] = sys.cardSet.Deal()
	}
	winAreaId, _ := sys.helper.GetType(cards[:])
	sharePrizePoolPercent, _ := config.Float("config", "SharePrizePoolPercent", "Value")
	for _, gold := range sys.areas {
		totalGold += gold
	}
	if winAreaId == cardutil.TexasStraightFlush || winAreaId == cardutil.TexasRoyalFlush {
		if sys.areas[winAreaId] > 0 {
			sharePrizePool = int64(float64(sys.prizePool) * sharePrizePoolPercent / 100)
			sys.prizePool -= sharePrizePool
		}
	}

	guid := util.GUID()
	winTimes, _ := config.Float("texas_lottery", winAreaId, "Times")
	totalFlushGold := sys.areas[cardutil.TexasStraightFlush] + sys.areas[cardutil.TexasRoyalFlush]
	for _, user := range sys.users {
		userFlushGold := user.areas[cardutil.TexasStraightFlush] + user.areas[cardutil.TexasRoyalFlush]
		if gold := user.areas[winAreaId]; gold > 0 {
			winGold := int64(float64(gold) * winTimes)
			if totalFlushGold > 0 {
				winGold += int64(float64(sharePrizePool) / float64(totalFlushGold) * float64(userFlushGold))
			}
			item := &service.Item{Id: gameutil.ItemIdGold, Num: winGold}
			service.AddItemsOrNotifyHall(user.Id, guid, "sys.lottery_win", item)

			userWinGold += winGold
			user.WinGold += winGold
			user.lastWinGold = winGold
			sys.updateHeap(user)
		}
		for i := range user.areas {
			user.areas[i] = 0
		}
	}
	if totalFlushGold > 0 {
		sys.prizePool = 0
	}

	record := LotteryRecord{Id: sys.id, Cards: cards, WinAreaId: winAreaId, Users: len(sys.users), Award: userWinGold}
	sys.history.PushBack(record)

	sys.updateTodayRank()
	if sys.history.Len() > 30 {
		front := sys.history.Front()
		sys.history.Remove(front)
	}

	var addPrizePool int64
	if extra := totalGold - userWinGold; extra > 0 {
		addPrizePool = int64(float64(extra)*sharePrizePoolPercent) / 100
		sys.prizePool += addPrizePool
	}
	sys.users = make(map[int]*LotteryUser)
	for i := range sys.areas {
		sys.areas[i] = 0
	}

	sec := service.GetShowTime(sys.deadline)
	data := map[string]any{
		"Sec":          sec,
		"AddPrizePool": addPrizePool,
		"Record":       record,
	}
	// service.Broadcast2Game("EndLottery", data)
	for _, player := range service.GetAllPlayers() {
		var myWinGold int64
		if user, ok := sys.users[player.Id]; ok && user.lastWinGold > 0 {
			myWinGold = user.lastWinGold
		}
		data["MyWinGold"] = myWinGold
		player.WriteJSON("EndLottery", data)
	}
}

func (sys *LotterySystem) updateHeap(user *LotteryUser) {
	if _, ok := sys.usersInRank[user.Id]; !ok {
		sys.usersInRank[user.Id] = user
	}
	user = sys.usersInRank[user.Id]

	if user.pos != -1 {
		heap.Push(&sys.todayRankHeap, user)
	}
	if h := sys.todayRankHeap; h.Len() > 100 {
		back := heap.Remove(&h, h.Len()-1)
		user := back.(*LotteryUser)
		delete(sys.usersInRank, user.Id)
	}
}

func (sys *LotterySystem) updateTodayRank() {
	temp := make([]*LotteryUser, sys.todayRankHeap.Len())
	copy(temp, []*LotteryUser(sys.todayRankHeap))

	h := LotteryUserHeap(temp)
	for i := 0; i < cap(sys.todayRank) && h.Len() > 0; i++ {
		top := heap.Remove(&sys.todayRankHeap, 0)
		user := top.(*LotteryUser)
		sys.todayRank = append(sys.todayRank, *user)
	}
}

func (sys *LotterySystem) updateNewDay() {
	sys.todayRank, sys.yesterdayRank = sys.yesterdayRank, sys.todayRank
	sys.todayRank = sys.todayRank[:0]

	sys.usersInRank = make(map[int]*LotteryUser)
	if slice := []*LotteryUser(sys.todayRankHeap); len(slice) > 0 {
		sys.todayRankHeap = LotteryUserHeap(slice[:0])
	}
}
