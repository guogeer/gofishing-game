package cache

import (
	"context"
	"time"

	"gofishing-game/internal/dbo"
	"gofishing-game/internal/pb"

	"github.com/guogeer/quasar/log"
)

// 每日定时拆分item_log
func splitItemLog() {
	db := dbo.Get()
	table := "item_log"
	lastTable := table + "_" + time.Now().Add(-23*time.Hour).Format("20060102")
	removeTable := table + "_" + time.Now().Add(-90*24*time.Hour).Format("20060102")
	// 保留N天日志
	log.Info("split table " + table + " drop " + removeTable)
	db.Exec("drop table if exists " + removeTable)
	db.Exec("create table if not exists " + lastTable + " like " + table)
	// 例如：rename table item_log to item_log_temp,item_log_20210102 to item_log, item_log_temp to item_log_20210102
	db.Exec("rename table " + table + " to " + table + "_temp, " + lastTable + " to " + table + ", " + table + "_temp to " + lastTable)
}

// 批量增加物品日志
func (cc *Cache) AddSomeItemLog(ctx context.Context, req *pb.AddSomeItemLogReq) (*pb.EmptyResp, error) {
	uid := req.Uid
	way := req.Way
	uuid := req.Uuid
	db := dbo.Get()
	for _, item := range req.Items {
		db.Exec("insert item_log(uid,way,guid,item_id,item_num,balance,extra) values(?,?,?,?,?,?,?)",
			uid, way, uuid, item.Id, item.Num, item.Balance, dbo.JSON(req.Extra))
	}
	return &pb.EmptyResp{}, nil
}

// 批量增加物品
func (cc *Cache) AddSomeItem(ctx context.Context, req *pb.AddSomeItemReq) (*pb.EmptyResp, error) {
	uid := req.Uid
	return cc.SaveBin(ctx, &pb.SaveBinReq{
		Uid: uid,
		Bin: &pb.UserBin{
			Offline: &pb.OfflineBin{Items: req.Items},
		},
	})
}
