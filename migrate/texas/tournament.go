package texas

import (
	"container/heap"
	"container/list"
	"gofishing-game/internal/gameutils"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
	"strconv"
	"strings"
	"time"

	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/utils"
)

const (
	TournamentStatusNone          = iota
	TournamentStatusFree          // 空闲
	TournamentStatusRegister      // 报名
	TournamentStatusDelayRegister // 延迟报名
	TournamentStatusPlay          // 游戏开始
)

type TournamentRoom interface {
	AddBlind(int64, int64, int64)
}

type TournamentUser struct {
	Id             int   `json:"id,omitempty"`
	RoomFee        int64 `json:"roomFee,omitempty"`
	RegisterFee    int64 `json:"registerFee,omitempty"`
	RegisterItemId int   `json:"registerItemId,omitempty"`

	Gold     int64  `json:"gold,omitempty"`
	Nickname string `json:"nickname,omitempty"`
	RankId   int    `json:"rankId,omitempty"`

	index        int
	RegisterTime time.Time `json:"registerTime,omitempty"`
}

type TournamentHeap []*TournamentUser

func (h TournamentHeap) Len() int           { return len(h) }
func (h TournamentHeap) Less(i, j int) bool { return h[i].Gold < h[j].Gold }
func (h TournamentHeap) Swap(i, j int) {
	h[i].index, h[j].index = j, i
	h[i], h[j] = h[j], h[i]
}

func (h *TournamentHeap) Push(x interface{}) {
	user := x.(*TournamentUser)
	*h = append(*h, x.(*TournamentUser))
	user.index = len(*h) - 1
}

func (h *TournamentHeap) Pop() interface{} {
	old := *h
	n := len(old)
	user := old[n-1]
	*h = old[:n-1]

	user.index = -1
	return user
}

type SimpleTournamentInfo struct {
	Id                    int    `json:"id,omitempty"`
	SubId                 int    `json:"subId,omitempty"`
	Icon                  string `json:"icon,omitempty"`
	Name                  string `json:"name,omitempty"`
	StartTime             string `json:"startTime,omitempty"` // 开局时间
	RoomFee               int64  `json:"roomFee,omitempty"`
	RegisterFee           int64  `json:"registerFee,omitempty"`
	RegisterItemId        int    `json:"registerItemId,omitempty"`        // 报名道具ID
	StartDuration         string `json:"startDuration,omitempty"`         // 开局间隔时间，包括延迟报名、报名时间
	NoRefundDuration      string `json:"noRefundDuration,omitempty"`      // 报名后，比赛开始前2m不能退款
	RegisterDuration      string `json:"registerDuration,omitempty"`      // 报名时间，包括不能退款时间
	DelayRegisterDuration string `json:"delayRegisterDuration,omitempty"` // 延迟报名时间，一般5m

	RegisterUserNum int `json:"registerUserNum,omitempty"` // 报名人数
	MinRegisterUser int `json:"minRegisterUser,omitempty"` // 最小报名人数
	MaxRegisterUser int `json:"maxRegisterUser,omitempty"` // 最大报名人数
	AutoStartUser   int `json:"autoStartUser,omitempty"`   // 自动开局人数
}

type TournamentConfig struct {
	*SimpleTournamentInfo

	Bankroll  int64  // 初始筹码
	Rebuy     string // 重购
	Addon     string // 增购
	BlindList string // 盲注，格式：小盲/大盲/前注;小盲/大盲/前注...
	AwardList string // 名次奖励
	UserNum   int    // 参赛人数

	RebuyFee   int64
	RebuyTimes int
	AddonFee   int64
	AddonTimes int
}

type Tournament struct {
	*TournamentConfig
	Users map[int]*TournamentUser `json:"-"`

	Rank *TournamentHeap   `json:"-"`
	Top  []*TournamentUser `json:"top,omitempty"` // 前几名
}

func (t *Tournament) IsRegister(uid int) bool {
	if _, ok := t.Users[uid]; ok {
		return true
	}
	return false
}

func (t *Tournament) UpdateRank(users []*TournamentUser) {
	for _, user := range users {
		if index := user.index; index == -1 {
			heap.Push(t.Rank, user)
		} else {
			heap.Fix(t.Rank, index)
		}
	}

	// 更新排行榜名次
	n := t.Rank.Len()
	tempRank := make([]*TournamentUser, n)
	copy(tempRank, ([]*TournamentUser)(*t.Rank))

	h := TournamentHeap(tempRank)
	t.Top = t.Top[:0]
	for i := 0; h.Len() > 0 && i < cap(t.Top); i++ {
		x := heap.Pop(&h)
		user := x.(*TournamentUser)
		user.RankId = i
		t.Top = append(t.Top, user)
	}
}

// 比赛副本
type TournamentCopy struct {
	*Tournament

	rooms []*list.List
	e     *list.Element

	addBlindLoop  int // 升盲轮数
	addBlindTimer *utils.Timer

	failUsers int
}

func (cp *TournamentCopy) StartGame() {
	// 分配房间
	fakeRoom := cp.rooms[0].Front().Value.(*roomutils.Room)
	subId := fakeRoom.SubId
	for len(fakeRoom.GetAllPlayers()) > 0 {
		room := service.GetWorld().(roomutils.RoomWorld).NewRoom(subId)
		texasRoom := room.CustomRoom().(*TexasRoom)
		texasRoom.tournament = cp
		cp.rooms[0].PushBack(room)

		for _, p := range room.GetAllPlayers() {
			seatIndex := room.GetEmptySeat()
			if seatIndex == roomutils.NoSeat {
				break
			}

			roomutils.GetRoomObj(p).ChangeRoom()
			roomutils.GetRoomObj(p).SitDown(seatIndex)
		}
	}
	cp.addBlind()
}

// 统计游戏中玩家数量
func (cp *TournamentCopy) CountActivePlayers() int {
	var counter int
	for _, rooms := range cp.rooms {
		for e := rooms.Front(); e != nil; e = e.Next() {
			room := e.Value.(*roomutils.Room)
			for _, p := range room.GetSeatPlayers() {
				tp := p.GameAction.(*TexasPlayer)
				if p != nil && !tp.isFail {
					counter++
				}
			}
		}
	}
	return counter
}

func (cp *TournamentCopy) addBlind() {
	loop := cp.addBlindLoop
	blindList := strings.Split(cp.BlindList, ";")
	utils.StopTimer(cp.addBlindTimer)
	if loop < len(blindList) {
		blinds := strings.Split(blindList[loop], "/")
		if len(blinds) > 2 {
			smallBlind, _ := strconv.ParseInt(blinds[0], 10, 64)
			bigBlind, _ := strconv.ParseInt(blinds[1], 10, 64)
			frontBlind, _ := strconv.ParseInt(blinds[2], 10, 64)
			sec, _ := strconv.Atoi(blinds[3])
			for _, rooms := range cp.rooms {
				for e := rooms.Front(); e != nil; e = e.Next() {
					room := e.Value.(*roomutils.Room).CustomRoom().(TournamentRoom)
					room.AddBlind(smallBlind, bigBlind, frontBlind)
				}
			}
			cp.addBlindTimer = utils.NewTimer(cp.addBlind, time.Duration(sec)*time.Second)
		}
		cp.addBlindLoop++
	}
}

func (cp *TournamentCopy) MergeRoom(room *roomutils.Room) {
	countUsers := func(tempRoom *roomutils.Room) int {
		var n int
		for _, p := range tempRoom.GetAllPlayers() {
			tp := p.GameAction.(*TexasPlayer)
			if p != nil && tp.isFail == false {
				n++
			}
		}
		return n
	}

	pairRoom := room
	counter := countUsers(room)
	// TODO 查找有没有可以合并的房间
	if counter > 0 && room.Status == 0 {
		for k := 1; k < len(cp.rooms) && counter+k <= room.NumSeat(); k++ {
			for e := cp.rooms[k].Front(); e != nil; e = e.Next() {
				if room != pairRoom {
					pairRoom = e.Value.(*roomutils.Room)
				}
			}
		}
	}
	if room != pairRoom {
		room.MergeTo(pairRoom)
	}
}

func (room *TexasRoom) MergeTo(to *TexasRoom) {
	return
}

func (cp *TournamentCopy) isAbleRebuyOrAddon(format string, loop int) bool {
	for _, line := range strings.Split(format, ";") {
		line = strings.Replace(line, "~", "-", -1)
		points := strings.Split(line, "-")

		values := make([]int, 0, 2)
		for _, point := range points {
			n, _ := strconv.Atoi(point)
			values = append(values, n)
		}
		if len(values) > 0 {
			values = append(values, values[0])
		}
		if loop >= values[0] && loop <= values[1] {
			return true
		}
	}
	return false
}

func (cp *TournamentCopy) IsAbleRebuy(n int) bool {
	return cp.isAbleRebuyOrAddon(cp.Rebuy, n)
}

func (cp *TournamentCopy) IsAbleAddon(n int) bool {
	return cp.isAbleRebuyOrAddon(cp.Addon, n)
}

// 奖励
func (cp *TournamentCopy) Award(itemString string) {
	way := "tournament_award"
	for _, rooms := range cp.rooms {
		for e := rooms.Front(); e != nil; e = e.Next() {
			room := e.Value.(*roomutils.Room)
			for _, p := range room.GetSeatPlayers() {
				tp := p.GameAction.(*TexasPlayer)
				if p != nil && tp.isFail == false {
					p.BagObj().AddSomeItems(gameutils.ParseNumbericItems(itemString), way)
				}
			}
		}
	}
}

// 一种比赛可能会同时进行几场
type TournamentGame struct {
	Id     int
	copies *list.List
}

// 当前的比赛
func (g *TournamentGame) CurrentCopy() *TournamentCopy {
	if back := g.copies.Back(); back != nil {
		return back.Value.(*TournamentCopy)
	}
	return nil
}

func (g *TournamentGame) Register(user *TournamentUser) {
	uid := user.Id
	cp := g.CurrentCopy()
	if cp.IsRegister(uid) {
		return
	}
	cp.Rank.Push(user)
	cp.Users[uid] = user
}

func (g *TournamentGame) CancelRegister(uid int) {
	cp := g.CurrentCopy()
	if cp.IsRegister(uid) == false {
		return
	}

	user := cp.Users[uid]
	heap.Remove(cp.Rank, user.RankId)
	delete(cp.Users, uid)
}

func (g *TournamentGame) UpdateRank(users []*TournamentUser) {
	g.CurrentCopy().UpdateRank(users)
	cmd.Forward("hall", "FUNC_UpdateTournamentRank", map[string]any{"users": users})
}

// TODO 比赛场
func (room *TexasRoom) IsTypeTournament() bool {
	return false
}

// TODO 比赛场
func (room *TexasRoom) Tournament() *TournamentCopy {
	return room.tournament
}

// TODO推荐房间
func RecommendRooms(level int) []*roomutils.Room {
	return nil
}
