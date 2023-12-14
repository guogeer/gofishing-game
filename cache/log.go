package cache

import (
	"time"

	"gofishing-game/internal"
	"gofishing-game/internal/dbo"

	"github.com/guogeer/quasar/config"
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

func tick1d() {
	splitItemLog()
}

func Tick() {
	nextDateStr := time.Now().Add(24 * time.Hour).Format(internal.ShortDateFmt)
	nextDate, _ := config.ParseTime(nextDateStr)
	dayTimer := time.NewTimer(time.Until(nextDate))
	for {
		<-dayTimer.C
		log.Infof("call timer %s", time.Now().Format(internal.LongDateFmt))
		dayTimer.Reset(24 * time.Hour)
		tick1d()
	}
}
