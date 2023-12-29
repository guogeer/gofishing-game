// 玩家数据表

package service

import (
	"context"
	"encoding/json"
	"time"

	"gofishing-game/internal/errcode"
	"gofishing-game/internal/gameutils"
	"gofishing-game/internal/pb"
	"gofishing-game/internal/rpc"

	"google.golang.org/protobuf/proto"

	"github.com/guogeer/quasar/log"
	"github.com/guogeer/quasar/util"
)

type EnumDataReset int

const (
	_ = EnumDataReset(iota)
	DataResetPerDay
	DataResetPerWeek
)

type reseter interface {
	Reset(EnumDataReset)
}

type loadSaver interface {
	Save(any)
	Load(any)
}

// 保存游戏全局（大厅/房间）数据
type dataObj struct {
	player *Player

	loadSavers []loadSaver
	period     time.Duration
	saveTimer  *util.Timer
	items      []*gameutils.Item
	// offlinePos int32
	offline *pb.OfflineBin

	lastDayUpdateTs int64
}

func newDataObj(player *Player) *dataObj {
	obj := &dataObj{
		player: player,
		period: 109 * time.Second,
	}
	obj.Push(obj)
	return obj
}

func (obj *dataObj) Enter() errcode.Error {
	data := obj.player.enterReq.Data
	obj.loadAll(data.Bin)
	obj.updateNewDay()
	return errcode.Ok
}

func (obj *dataObj) BeforeEnter() {
	p := obj.player
	now := time.Now()

	if p.EnterReq().IsFirst() {
		util.StopTimer(obj.saveTimer)
		obj.saveTimer = p.TimerGroup.NewPeriodTimer(obj.onTime, now, obj.period)
	}
}

func (obj *dataObj) OnLeave() {
}

func (obj *dataObj) onTime() {
	uid := obj.player.Id
	bin := &pb.UserBin{}
	obj.saveAll(bin)

	bin = proto.Clone(bin).(*pb.UserBin)
	go func() {
		req := &pb.SaveBinReq{Uid: int32(uid), Bin: bin}
		rpc.CacheClient().SaveBin(context.Background(), req)
	}()
}

func (obj *dataObj) saveAll(data any) {
	p := obj.player
	bin := data.(*pb.UserBin)

	for _, h := range obj.loadSavers {
		h.Save(bin)
	}

	log.Debugf("player %d save all data", p.Id)
}

func (obj *dataObj) loadAll(data any) {
	for _, h := range obj.loadSavers {
		h.Load(data)
	}
}

func (obj *dataObj) Push(h loadSaver) {
	obj.loadSavers = append(obj.loadSavers, h)
}

func (obj *dataObj) Load(data any) {
	p := obj.player
	bin := data.(*pb.UserBin)
	gameutils.InitNilFields(bin.Global)

	obj.items = make([]*gameutils.Item, 0, 8)
	p.Level = int(bin.Global.Level)
	// p.BuildLevel = int(bin.Global.BuildLevel)
	for _, item := range bin.Global.Items {
		newItem := &gameutils.Item{}
		util.DeepCopy(newItem, item)
		obj.addItem(newItem)
	}

	// 对离线数据进行合并
	// obj.offlinePos = bin.Global.CurItemMaxPos
	obj.offline = &pb.OfflineBin{}
	for _, item := range bin.Offline.Items {
		obj.addItem(&gameutils.Item{Id: int(item.Id), Num: item.Num})
		obj.offline.Items = append(obj.offline.Items, &pb.Item{Id: item.Id, Num: -item.Num})
	}
	// obj.offlinePos = bin.OfflineBin.OfflineItemPos
	obj.lastDayUpdateTs = bin.Global.LastDayUpdateTs
}

func (obj *dataObj) addItem(newItem *gameutils.Item) {
	if isItemValid(newItem.Id) {
		obj.items = append(obj.items, newItem)
	}
}

func (obj *dataObj) updateNewDay() {
	if time.Now().Truncate(24*time.Hour) == time.Unix(obj.lastDayUpdateTs, 0).Truncate(24*time.Hour) {
		return
	}
	obj.lastDayUpdateTs = time.Now().Unix()
	for _, loadSaver := range obj.loadSavers {
		if h, ok := loadSaver.(reseter); ok {
			h.Reset(DataResetPerDay)
		}
	}
}

func (obj *dataObj) Save(data any) {
	p := obj.player
	bin := data.(*pb.UserBin)
	bin.Global = &pb.GlobalBin{}

	var items []*pb.Item
	for _, item := range obj.items {
		newItem := &pb.Item{}
		util.DeepCopy(newItem, item)
		items = append(items, newItem)
	}
	bin.Global.Items = items
	bin.Global.Level = int32(p.Level)
	// bin.Global.BuildLevel = int32(p.BuildLevel)
	bin.Offline, obj.offline = obj.offline, &pb.OfflineBin{}
	bin.Global.LastDayUpdateTs = obj.lastDayUpdateTs
}

type serviceDict struct {
	isLoad bool
	values map[string]any
}

var gServiceDict serviceDict

func (dict *serviceDict) load() {
	dict.isLoad = true
	for key, value := range dict.values {
		Dict, err := rpc.CacheClient().QueryDict(context.Background(), &pb.QueryDictReq{Key: key})
		if err != nil {
			log.Fatalf("load service dict %s error: %v", key, err)
		}
		if len(Dict.Value) == 0 {
			continue
		}
		if err := json.Unmarshal([]byte(Dict.Value), value); err != nil {
			log.Fatalf("parse service dict %s error: %v", key, err)
		}
	}
}

func (dict *serviceDict) save() {
	var reqs []*pb.UpdateDictReq
	for key, value := range dict.values {
		buf, _ := json.Marshal(value)
		reqs = append(reqs, &pb.UpdateDictReq{Key: key, Value: buf})
	}
	go func() {
		for _, req := range reqs {
			rpc.CacheClient().UpdateDict(context.Background(), req)
		}
	}()
}

func (dict *serviceDict) Add(key string, value any) {
	if dict.isLoad {
		panic("please add dict value before load")
	}
	if dict.values == nil {
		dict.values = map[string]any{}
	}
	dict.values[key] = value
}

func UpdateDict(key string, value any) {
	gServiceDict.Add(key, value)
}
