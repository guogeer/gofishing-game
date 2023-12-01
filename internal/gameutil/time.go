package gameutil

import (
	"encoding/json"
	// "github.com/guogeer/quasar/log"
	"time"
)

const MaxDelayDuration = 1200 * time.Millisecond

type CD struct {
	ExpireMs int64
	PeriodMs int64
	unit     time.Duration // 单位，默认ms
}

func NewCD(d time.Duration, unit ...time.Duration) *CD {
	if d < 0 {
		d = 0
	}

	unit2 := time.Millisecond
	for _, u := range unit {
		unit2 = u
	}
	return &CD{
		ExpireMs: time.Now().Add(d).UnixNano() / 1e6,
		PeriodMs: d.Milliseconds(),
		unit:     unit2,
	}
}

func (cd *CD) IsValid() bool {
	process := cd.process()
	return process[0] > 0
}

// 当前进度[剩余时间，总时间]
func (cd *CD) process() []int {
	expireTime := time.Unix(cd.ExpireMs/1000, cd.ExpireMs%1000*1e6)
	d := time.Until(expireTime)
	if d < 0 {
		d = 0
	}
	unit := cd.unit
	if unit == 0 {
		unit = time.Millisecond
	}
	period := int(time.Duration(cd.PeriodMs) * time.Millisecond / unit)
	offset := int(float64(d+unit-1) / float64(unit))
	if offset > period {
		offset = period
	}
	return []int{offset, period}
}

func (cd *CD) MarshalJSON() ([]byte, error) {
	return json.Marshal(cd.process())
}

type Clock struct {
	t    time.Time
	unit time.Duration
}

func NewClock(d time.Duration, unit ...time.Duration) *Clock {
	unit2 := time.Millisecond
	for _, u := range unit {
		unit2 = u
	}
	return &Clock{t: time.Now().Add(d), unit: unit2}
}

func (c Clock) IsValid() bool {
	return !time.Now().After(c.t)
}

func (c Clock) MarshalJSON() ([]byte, error) {
	unit := c.unit
	d := time.Until(c.t)
	offset := int(float64(d+unit-1) / float64(unit))
	if offset < 0 {
		offset = 0
	}
	// log.Debug(d, unit, int(d/unit), float64(d)/float64(unit), offset)
	return json.Marshal(offset)
}
