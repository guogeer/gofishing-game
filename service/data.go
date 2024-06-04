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

	"github.com/guogeer/quasar/v2/log"
	"github.com/guogeer/quasar/v2/utils"
	"google.golang.org/protobuf/proto"
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

	loadSavers      []loadSaver
	period          time.Duration
	saveTimer       *utils.Timer
	offline         *pb.OfflineBin
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

func (obj *dataObj) TryEnter() errcode.Error {
	request := GetEnterQueue().GetRequest(obj.player.Id)
	obj.loadAll(request.EnterGameResp.Bin)
	obj.updateNewDay()
	return nil
}

func (obj *dataObj) BeforeEnter() {
	p := obj.player
	now := time.Now()

	if !obj.saveTimer.IsValid() {
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

	p.Level = int(bin.Global.Level)
	obj.lastDayUpdateTs = bin.Global.LastDayUpdateTs
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

	bin.Global.Level = int32(p.Level)
	bin.Offline, obj.offline = obj.offline, &pb.OfflineBin{}
	bin.Global.LastDayUpdateTs = obj.lastDayUpdateTs
}

type globalDictItem struct {
	Data            any   `json:"data,omitempty"`
	LastDayUpdateTs int64 `json:"lastDayUpdateTs,omitempty"`
}

// 全局数据
type GlobalDict struct {
	values map[string]*globalDictItem
}

var globalData GlobalDict

func (gdata *GlobalDict) load() {
	for key, value := range gdata.values {
		resp, err := rpc.CacheClient().QueryDict(context.Background(), &pb.QueryDictReq{Key: key})
		if err != nil {
			log.Fatalf("load service dict %s error: %v", key, err)
		}
		if len(resp.Value) == 0 {
			continue
		}
		if err := json.Unmarshal([]byte(resp.Value), value); err != nil {
			log.Fatalf("parse service dict %s error: %v", key, err)
		}
	}
	gdata.updateNewDay()
}

func (gdata *GlobalDict) save() {
	var reqs []*pb.UpdateDictReq
	for key, value := range gdata.values {
		buf, _ := json.Marshal(value)
		reqs = append(reqs, &pb.UpdateDictReq{Key: key, Value: buf})
	}
	go func() {
		for _, req := range reqs {
			rpc.CacheClient().UpdateDict(context.Background(), req)
		}
	}()
}

func (gdata *GlobalDict) Add(key string, value any) {
	if gdata.values == nil {
		gdata.values = map[string]*globalDictItem{}
	}
	gdata.values[key] = &globalDictItem{
		Data: value,
	}
}

func (gdata *GlobalDict) updateNewDay() {
	for _, value := range gdata.values {
		if time.Now().Truncate(24*time.Hour) != time.Unix(value.LastDayUpdateTs, 0).Truncate(24*time.Hour) {
			value.LastDayUpdateTs = time.Now().Unix()
			if h, ok := value.Data.(reseter); ok {
				h.Reset(DataResetPerDay)
			}
		}
	}
}

func UpdateDict(key string, value any) {
	globalData.Add(key, value)
}
