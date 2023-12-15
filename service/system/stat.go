// 统计

package system

import (
	"time"

	"gofishing-game/internal"
	"gofishing-game/internal/gameutils"
	"gofishing-game/internal/pb"
	"gofishing-game/service"

	"github.com/guogeer/quasar/config"
)

const actionKeyStat = "stat"

const (
	_           = iota
	userTypeNew // 新手
	userTypeOld // 老手
)

type statObj struct {
	player *service.Player

	onlineTime  time.Time
	offlineTime time.Time
	lastServer  string
	data        *pb.StatBin
}

func (obj *statObj) BeforeEnter() {
	p := obj.player

	obj.onlineTime = time.Now()
	obj.offlineTime = time.Time{}
	obj.data.ClientVersion = p.EnterReq().Auth.ClientVersion

	curDate := time.Now().Format(internal.ShortDateFmt)
	if obj.data.LastEnterTime == "" || obj.data.LastEnterTime[:len(internal.ShortDateFmt)] != curDate {
		obj.data.LoginDayNum += 1
	}
	obj.data.LastEnterTime = time.Now().Format(internal.LongDateFmt)

	dayStat := obj.GetDayStat()
	dayStat.IsEnter = true
}

func (obj *statObj) Load(data any) {
	bin := data.(*pb.UserBin)
	obj.data = &pb.StatBin{}
	if bin.Stat != nil {
		obj.data = bin.Stat
	}
	gameutils.InitNilFields(obj.data)
	obj.lastServer = obj.data.LastServer
}

func (obj *statObj) Save(data any) {
	bin := data.(*pb.UserBin)

	p := obj.player
	obj.data.CopyLevel = int32(p.Level)

	for _, rowId := range config.Rows("item") {
		itemId, _ := config.Int("item", rowId, "ShopID")
		itemStat := obj.getItemStat(int(itemId))
		itemStat.CopyNum = obj.player.ItemObj().NumItem(int(itemId))
	}
	bin.Stat = obj.data
}

func (obj *statObj) Data() *pb.StatBin {
	return obj.data
}

func (obj *statObj) OfflineTime() time.Time {
	return obj.offlineTime
}

func (obj *statObj) getItemStat(itemId int) *pb.ItemStat {
	itemStat, ok := obj.data.Items[int32(itemId)]
	if !ok {
		itemStat = &pb.ItemStat{}
		obj.data.Items[int32(itemId)] = itemStat
	}
	return itemStat
}

func (obj *statObj) OnAddItems(itemLog *gameutils.ItemLog) {
	for _, item := range itemLog.Items {
		itemStat := obj.getItemStat(item.Id)
		if item.Num < 0 {
			itemStat.Cost += -item.Num
		} else {
			itemStat.Add += item.Num
		}
		itemStat.Count += 1

		obj.data.LastItemWay = itemLog.Way
	}
}

func (obj *statObj) OnClose() {
	obj.offlineTime = time.Now()
	obj.CountOnline()
}

func (obj *statObj) OnLeave() {
	obj.CountOnline()
}

func (obj *statObj) CountOnline() {
	p := obj.player
	if !p.IsSessionClose {
		secs := int32(time.Since(obj.onlineTime).Seconds())
		obj.data.OnlineSecs += secs
	}
	obj.onlineTime = time.Now()
}

// 保留最近15天的数据
func (obj *statObj) GetDayStat() *pb.UserDayStat {
	countKey := func(t time.Time) int32 {
		return int32(10000*t.Year() + 100*int(t.Month()) + t.Day())
	}

	statData := obj.data
	dayKey := countKey(time.Now())
	if _, ok := statData.Day[dayKey]; !ok {
		statData.Day[dayKey] = &pb.UserDayStat{}
	}

	expireKey := countKey(time.Now().Add(-15 * 24 * time.Hour))
	for key := range statData.Day {
		if key < expireKey {
			delete(statData.Day, key)
		}
	}
	return statData.Day[dayKey]
}

func GetStatObj(player *service.Player) *statObj {
	return player.GetAction(actionKeyStat).(*statObj)
}

func newStatObj(player *service.Player) service.EnterAction {
	obj := &statObj{player: player}
	player.DataObj().Push(obj)
	return obj
}

type statArgs struct {
	Type int
}

func init() {
	service.AddAction(actionKeyStat, newStatObj)
}
