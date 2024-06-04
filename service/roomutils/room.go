package roomutils

import (
	"encoding/json"
	"slices"
	"strconv"
	"strings"
	"time"

	"gofishing-game/internal/errcode"
	"gofishing-game/service"

	"github.com/guogeer/quasar/v2/cmd"
	"github.com/guogeer/quasar/v2/config"
	"github.com/guogeer/quasar/v2/log"
	"github.com/guogeer/quasar/v2/utils"
)

var gSubGames map[int]*subGame // 所有的场次

var ErrPlaying = errcode.New("playing", "wait game ending")

const NoSeat = -1

const (
	_                 = iota
	RoomStatusPlaying // 游戏中
)

var (
	OptAutoPlay                  = "autoPlay"
	OptForbidEnterAfterGameStart = "forbidEnterAfterGameStart"
	OptSeat                      = "seat_%d"
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
		if GetRoomObj(player).room.SubId == sub.Id {
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
	var onlines []service.ServerOnline
	for _, sub := range gSubGames {
		sub.updateOnline()
		one := service.ServerOnline{
			Online: sub.Online,
			Id:     service.GetServerId() + ":" + sub.serverName + ":" + strconv.Itoa(sub.Id),
		}
		onlines = append(onlines, one)
	}
	cmd.Forward("hall", "func_updateOnline", cmd.M{"games": onlines})
}

type RoomWorld interface {
	service.World
	NewRoom(subId int) *Room
}

func LoadGames() {
	servers := service.GetAllServers()

	games := map[int]*subGame{}
	for _, rowId := range config.Rows("room") {
		tagStr, _ := config.String("room", rowId, "tags")
		tags := strings.Split(tagStr, ",")

		var name string
		for _, tag := range tags {
			if slices.Index(servers, tag) >= 0 {
				name = tag
			}
		}

		var subId, seatNum int
		config.Scan("room", rowId, "id,seatNum", &subId, &seatNum)

		log.Infof("load game:%d name:%s", subId, name)
		games[subId] = &subGame{
			Id:         subId,
			MaxSeatNum: seatNum,
			serverName: name,
		}
	}
	gSubGames = games
}

func init() {
	service.GetEnterQueue().SetLocationFunc(func(uid int) string {
		args := &roomEnterArgs{}
		enterReq := service.GetEnterQueue().GetRequest(uid)
		json.Unmarshal(enterReq.RawData, args)
		return service.GetServerId() + ":" + strconv.Itoa(args.SubId)
	})
	utils.GetTimerSet().NewPeriodTimer(tick10s, time.Now(), 10*time.Second)
}

func tick10s() {
	updateOnline()
}
