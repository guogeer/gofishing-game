package main

import (
	"gofishing-game/internal"
	"time"

	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
)

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
