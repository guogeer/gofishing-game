package service

import (
	"context"
	"fmt"
	"quasar/utils"
	"time"

	"gofishing-game/internal/gameutils"
	"gofishing-game/internal/pb"
	"gofishing-game/internal/rpc"

	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
	"github.com/guogeer/quasar/script"
)

var (
	playerObjectPool []*Player          // 玩家对象缓存
	gGatewayPlayers  map[string]*Player // 关联网络连接
	gAllPlayers      map[int]*Player    // 所有玩家

	defaultWorld World
)

// 在线人数
type ServerOnline struct {
	Id     string `json:"id,omitempty"` // serverId:serverName{:subId}
	Online int    `json:"online,omitempty"`
}

func createPlayer(uid int) *Player {
	for len(playerObjectPool) < 1 {
		p := GetWorld().NewPlayer()
		playerObjectPool = append(playerObjectPool, p)

		for _, key := range actionKeys {
			h := actionConstructors[key]
			p.enterActions[key] = h(p)
		}
	}

	n := len(playerObjectPool)
	comer := playerObjectPool[n-1]
	playerObjectPool = playerObjectPool[:n-1]
	comer.Id = uid
	return comer
}

type World interface {
	NewPlayer() *Player
	GetName() string
}

func GetWorld() World {
	return defaultWorld
}

func init() {
	gAllPlayers = make(map[int]*Player)
	gGatewayPlayers = make(map[string]*Player)

	startTime, _ := config.ParseTime("2018-01-31 00:00:00")
	utils.NewPeriodTimer(tick10s, startTime, 10*time.Second)
	utils.NewPeriodTimer(tick1d, startTime, 24*time.Hour)
}

func CreateWorld(w World) {
	defaultWorld = w
	gServiceDict.load()
}

func tick10s() {
	gServiceDict.save()
}

func Broadcast2Game(messageId string, data any) {
	for _, player := range gAllPlayers {
		player.WriteJSON(messageId, data)
	}
}

func Broadcast2Gateway(messageId string, i any) {
	pkg := &cmd.Package{Id: messageId, Body: i}
	data, err := cmd.EncodePackage(pkg)
	if err != nil {
		return
	}
	cmd.Route("router", "C2S_Broadcast", data)
}

func GetAllPlayers() []*Player {
	players := make([]*Player, 0, len(gAllPlayers))
	for _, player := range gAllPlayers {
		players = append(players, player)
	}
	return players
}

func GetPlayer(id int) *Player {
	if p, ok := gAllPlayers[id]; ok {
		return p
	}
	return nil
}

func GetGatewayPlayer(ssid string) *Player {
	p, ok := gGatewayPlayers[ssid]
	// 玩家不存在
	if !ok {
		return nil
	}
	// 玩家在进入游戏或离开游戏过程中，屏蔽其他消息请求
	if p.IsBusy() {
		return nil
	}
	return p
}

func GetServerName() string {
	return GetWorld().GetName()
}

func GetServerId() string {
	if *serverId == "" {
		return GetServerName()
	}
	return *serverId
}

func WriteMessage(ss *cmd.Session, id string, i any) {
	if ss == nil {
		return
	}
	serverName := GetServerName()
	if serverName != "" {
		id = fmt.Sprintf("%s.%s", serverName, id)
	}
	if m, ok := i.(map[any]any); ok {
		i = (script.GenericMap)(m)
	}

	pkg := &cmd.Package{Id: id, Body: i}
	buf, err := cmd.EncodePackage(pkg)

	if err != nil {
		log.Errorf("route message %s %v %v", id, i, err)
		return
	}
	ss.WriteJSON("FUNC_Route", buf)
}

func AddItems(uid int, items []gameutils.Item, way string) {
	// log.Debugf("player %d AddItems way %s", uid, itemLog.Way)
	if p := GetPlayer(uid); p != nil && !p.IsBusy() {
		p.bagObj.AddSomeItems(items, way)
	} else {
		// 玩家不在线
		bin := &pb.UserBin{Offline: &pb.OfflineBin{}}
		for _, item := range items {
			bin.Offline.Items = append(bin.Offline.Items, &pb.NumericItem{Id: int32(item.GetId()), Num: item.GetNum()})
		}
		AddSomeItemLog(uid, items, way)
		go func() {
			rpc.CacheClient().SaveBin(context.Background(), &pb.SaveBinReq{Uid: int32(uid), Bin: bin})
		}()
	}
}

func AddSomeItemLog(uid int, items []gameutils.Item, way string) {
	if len(items) == 0 {
		return
	}

	pbItems := make([]*pb.NumericItem, 0, 4)
	for _, item := range items {
		pbItems = append(pbItems, &pb.NumericItem{Id: int32(item.GetId()), Num: item.GetNum()})
	}

	if p := GetPlayer(uid); p != nil {
		for _, pbItem := range pbItems {
			pbItem.Balance = p.BagObj().NumItem(int(pbItem.Id))
		}
	}

	// 玩家日志按时序更新
	req := &pb.AddSomeItemLogReq{
		Uid:      int32(uid),
		Items:    pbItems,
		Uuid:     utils.GUID(),
		Way:      way,
		CreateTs: time.Now().Unix(),
	}
	go func() {
		rpc.CacheClient().AddSomeItemLog(context.Background(), req)
	}()
}

func tick1d() {
	for _, player := range gAllPlayers {
		player.dataObj.updateNewDay()
	}
}

type GameOnlineSegment struct {
	Id          int
	PlayerTotal int
	PlayerCure  []int
}

func GetPlayerByContext(ctx *cmd.Context) *Player {
	return GetGatewayPlayer(ctx.Ssid)
}
