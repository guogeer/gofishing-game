package roomutil

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"gofishing-game/service"

	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
	"github.com/guogeer/quasar/util"
)

var gSubGames map[int]*subGame // 所有的场次

const NoSeat = -1

const (
	_                 = iota
	RoomStatusPlaying // 游戏中
)

type subGame struct {
	Id         int
	MaxSeatNum int
	UserNum    int    // 机器人+玩家数量
	serverName string // 房间的场次名称
	Online     int

	rooms []*Room
}

// 虚假的在线人数
func (sub *subGame) updateOnline() {
	sub.Online = 0
	for _, player := range service.GetAllPlayers() {
		if player.EnterReq().SubId == sub.Id {
			if !player.IsRobot && !player.IsSessionClose {
				sub.Online++
			}
		}
	}
	// log.Infof("current %s online users %d:%d", GetName(), sub.Online, len(gAllPlayers))
}

func GetServerName(subId int) string {
	if sub, ok := gSubGames[subId]; ok {
		return sub.serverName
	}
	return ""
}

func updateOnline() {
	var onlines []service.ClientOnline
	for _, sub := range gSubGames {
		sub.updateOnline()
		one := service.ClientOnline{
			Online:     sub.Online,
			ServerName: service.GetName() + ":" + strconv.Itoa(sub.Id),
		}
		onlines = append(onlines, one)
	}
	cmd.Forward("hall", "FUNC_UpdateOnline", cmd.M{"Games": onlines})
}

type greatWorld interface {
	Servers() []string
}

type RoomWorld interface {
	service.World
	NewRoom(subId int) *Room
}

func LoadGames(w RoomWorld) {
	gSubGames = map[int]*subGame{}

	servers := []string{service.GetName()}
	if gw, ok := w.(greatWorld); ok {
		servers = append(servers, gw.Servers()...)
	}
	for _, rowId := range config.Rows("Room") {
		tagStr, _ := config.String("Room", rowId, "Tags")
		tags := strings.Split(tagStr, ",")

		var name string
		for _, sname := range servers {
			if slices.Index(tags, sname) >= 0 {
				name = sname
				break
			}
		}

		if name != "" {
			var subId, seatNum, roomNum int
			config.Scan("Room", rowId, "RoomID,SeatNum,RoomNum", &subId, &seatNum, &roomNum)

			log.Infof("load game:%d name:%s", subId, name)
			gSubGames[subId] = &subGame{
				Id:         subId,
				MaxSeatNum: seatNum,
				serverName: name,
			}
		}
	}
}

func init() {
	util.GetTimerSet().NewPeriodTimer(tick10s, time.Now(), 10*time.Second)
}

func tick10s() {
	updateOnline()
}
