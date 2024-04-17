package fruit

import (
	"gofishing-game/service"
	"math/rand"
	. "third/errcode"

	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"

	// "third/pb"
	// "third/rpc"
	"time"

	"github.com/guogeer/quasar/util"
)

const (
	RoomStatusFree    = iota + service.RoomStatusFree // 等待玩家准备
	RoomStatusPlaying                                 // 游戏中
	RoomStatusAward                                   // 结算
)

const (
	MaxBetArea = 11
	syncTime   = 1000 * time.Millisecond
)

// 牌：1、东；2、南；3、西；4、北；5、一；6、二；7、三；8、四；9、发X25；10、发X100；11、通杀；12、通赔
// 押注区域：0、风；1、东；2、南；3、西；4、北；5、万；6、一万；7、二万；8、三万；9、四万；10、发

var (
	AllFruits       = []int{1, 5, 2, 6, 10, 3, 7, 1, 5, 1, 5, 11, 2, 6, 3, 7, 2, 6, 9, 1, 5, 2, 6, 1, 5, 12, 4, 8}
	defaultAllTimes = []float64{1.98, 4, 5, 10, 20, 1.98, 4, 5, 10, 20, 25, 100}
	// AllFruits       = []int{3, 5, 2, 10, 5, 2, 7, 1, 6, 4, 11, 5, 1, 6, 3, 2, 1, 9, 5, 2, 7, 6, 1, 8, 12, 5, 1, 6}
	// AllTimes  = []float64{2, 6, 6, 8, 15, 2, 6, 6, 8, 15, 25, 100}

	defaultSystemTax = 0.0 // 系统扣税
)

type SeatArea struct {
	SeatId int
	Area   int
}

type FruitRoom struct {
	*service.Room
	BetArea  [MaxBetArea]int64
	Chips    []int64
	Deadline time.Time
	last     [64]int // 历史记录
	lasti    int     // 历史记录索引
	// seatAreas []SeatArea
}

func (room *FruitRoom) OnEnter(player *service.Player) {
	room.Room.OnEnter(player)

	ply := player.GameAction.(*FruitPlayer)
	log.Infof("player %d enter room %d", ply.Id, room.Id)
	// 玩家重连
	data := map[string]any{
		"Code":      Ok,
		"Status":    room.Status,
		"SubId":     room.GetSubId(),
		"Chips":     room.Chips,
		"Countdown": room.GetShowTime(room.Deadline),
		"Fruits":    AllFruits,
	}

	// 正在游戏中
	if room.Status == RoomStatusPlaying {
		data["BetArea"] = room.BetArea[:]
		data["Online"] = len(room.AllPlayers)
		data["BetUserNum"] = room.countBetUser()
	}
	var seats []SeatPlayerInfo
	for i := 0; i < room.SeatNum(); i++ {
		if p := room.GetPlayer(i); p != nil {
			info := SeatPlayerInfo{SeatId: p.SeatId}
			util.DeepCopy(&info.SimpleUserInfo, &p.UserInfo)
			seats = append(seats, info)
		}
	}
	data["SeatPlayers"] = seats
	ply.WriteJSON("GetRoomInfo", data)
}

func (room *FruitRoom) OnLeave(player *service.Player) {
	room.Room.OnLeave(player)
}

func (room *FruitRoom) OnBet(ply *FruitPlayer, area int, gold int64) {
	room.BetArea[area] += gold
}

func (room *FruitRoom) StartGame() {
	s, _ := config.String("config", "FruitChips", "Value")
	room.Chips = util.ParseIntSlice(s)

	// 等待玩家准备下一把
	log.Debugf("room %d start game", room.Id)
	room.Status = RoomStatusPlaying
	for k, _ := range room.BetArea {
		room.BetArea[k] = 0
	}

	d, _ := config.Duration("config", "FruitBetTime", "Value")
	room.Deadline = time.Now().Add(d)
	room.Broadcast("StartGame", map[string]any{"Sec": room.GetShowTime(room.Deadline)})
	util.NewTimer(room.Award, d)
}

func (room *FruitRoom) Award() {
	var winArea [MaxBetArea]float64

	fruitId := 1
	t := rand.Intn(1000000)
	for i := 0; i < config.Row("Fruits"); i++ {
		rowId := config.RowId(i)
		rate, _ := config.Int("Fruits", rowId, "Rate")
		if int(rate) > t {
			fruitId = i + 1
			break
		}
	}

	var allFruitId []int
	for k, v := range AllFruits {
		if v == fruitId {
			allFruitId = append(allFruitId, k)
		}
	}

	allTimes := defaultAllTimes
	// 统计水果位置
	pos := allFruitId[rand.Intn(len(allFruitId))]
	switch fruitId {
	case 1, 2, 3, 4:
		winArea[fruitId] = float64(allTimes[fruitId])
		winArea[0] = float64(allTimes[0])
	case 5, 6, 7, 8:
		winArea[fruitId+1] = float64(allTimes[fruitId+1])
		winArea[5] = float64(allTimes[5])
	case 9, 10:
		winArea[10] = float64(allTimes[fruitId])
	case 11: // 通杀
	case 12: // 通赔
		for i, _ := range winArea {
			winArea[i] = 2
		}
	}
	// room.Status = RoomStatusAward
	room.Sync()

	room.last[room.lasti] = fruitId
	room.lasti = (room.lasti + 1) % len(room.last)
	// log.Debug("roll", dice1, dice2)

	d := room.RestartTime()
	room.Deadline = time.Now().Add(d)
	util.NewTimer(room.StartGame, d)
	sec := room.GetShowTime(room.Deadline)

	guid := util.GUID()
	// 座位玩家中奖
	var areas []SeatArea
	var betArea [MaxBetArea]int64
	for i := 0; i < room.SeatNum(); i++ {
		if p := room.GetPlayer(i); p != nil {
			for k, v := range p.fruitObj.BetArea {
				if float64(v)*winArea[k] > 0 {
					areas = append(areas, SeatArea{SeatId: i, Area: k})
					betArea[k] += v
				}
			}
		}
	}
	for k, v := range betArea {
		if v < room.BetArea[k] && winArea[k] > 0 {
			areas = append(areas, SeatArea{SeatId: -1, Area: k})
		}
	}

	// room.seatAreas = areas
	// 大赢家
	type BigWinner struct {
		Info    *service.SimpleUserInfo
		WinGold int64
	}
	systemTax := defaultSystemTax
	if tax, ok := config.Float("config", "FruitSystemTax", "Value"); ok {
		systemTax = tax
	}
	var robot *FruitPlayer
	var totalRobotBet, totalRobotWin int64

	bigWinner := &BigWinner{}
	betWay := service.ItemWay{Way: "sys.fruit_bet", SubId: room.GetSubId()}.String()
	tempBetWay := service.ItemWay{Way: "sum.fruit_bet", SubId: room.GetSubId()}.String()
	for _, player := range room.AllPlayers {
		p := player.GameAction.(*FruitPlayer)
		way := betWay
		if p.IsRobot {
			robot = p
			way = tempBetWay
			totalRobotBet += p.fruitObj.AllBet
		}
		p.AddGoldLog(-p.fruitObj.AllBet, guid, way)

		var gold int64
		for k, v := range p.fruitObj.BetArea {
			gold += int64(float64(v) * winArea[k])
		}

		if bigWinner.WinGold < gold {
			bigWinner.WinGold = gold
			bigWinner.Info = p.GetSimpleInfo(0)
		}
		p.RoomObj.WinGold += gold
		p.winGold = int64(float64(gold) * (1.0 - systemTax))
	}

	data := map[string]any{"Roll": pos, "Sec": sec, "Luck": areas}
	if bigWinner.WinGold > 0 {
		data["BigWinner"] = bigWinner
	}

	awardWay := service.ItemWay{Way: "sys.fruit_award", SubId: room.GetSubId()}.String()
	tempAwardWay := service.ItemWay{Way: "sum.fruit_award", SubId: room.GetSubId()}.String()
	for _, player := range room.AllPlayers {
		p := player.GameAction.(*FruitPlayer)

		gold := p.winGold
		data["WinGold"] = gold
		p.WriteJSON("Award", data)

		way := awardWay
		if p.IsRobot {
			robot = p
			way = tempAwardWay
			totalRobotWin += gold
		}

		p.AddGold(gold, guid, way)
	}
	if robot != nil {
		robot.AddGoldLog(-totalRobotBet, guid, betWay)
		robot.AddGoldLog(totalRobotWin, guid, awardWay)
	}

	room.GameOver()
}

func (room *FruitRoom) OnTime() {
	room.Sync()
	util.NewTimer(room.OnTime, syncTime)
}

func (room *FruitRoom) Sync() {
	data := map[string]any{}
	data["Online"] = len(room.AllPlayers)
	data["BetUserNum"] = room.countBetUser()
	if room.Status == RoomStatusPlaying {
		data["BetArea"] = room.BetArea[:]
	}
	room.Broadcast("Sync", data)
}

func (room *FruitRoom) GetLast(n int) []int {
	var last []int
	N := len(room.last)
	if N == 0 {
		return last
	}
	for i := (N - n%N + room.lasti) % N; i != room.lasti; i = (i + 1) % N {
		d := room.last[i]
		if d > 0 {
			last = append(last, d)
		}
	}
	return last
}

func (room *FruitRoom) GetPlayer(i int) *FruitPlayer {
	if p := room.SeatPlayers[i]; p != nil {
		return p.GameAction.(*FruitPlayer)
	}
	return nil
}

func (room *FruitRoom) countBetUser() int {
	var counter int
	for _, player := range room.AllPlayers {
		p := player.GameAction.(*FruitPlayer)
		if p.fruitObj.AllBet > 0 {
			counter++
		}
	}
	return counter
}
