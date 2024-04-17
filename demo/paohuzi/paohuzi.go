package paohuzi

import (
	"github.com/guogeer/quasar/log"
)

const (
	InvalidCard = -1
)

const (
	NoneCard = 50 // 必须小于MaxCard
	MaxCard  = NoneCard + 10
)

const (
	OptNone       = iota
	OptBoom       // 放炮
	OptYiDianHong // 一点红
)

func PrintCards(cards []int) {
	var a []int
	for c, n := range cards {
		for i := 0; i < n; i++ {
			a = append(a, c)
		}
	}
	log.Debug("print cards", a)

	for c, n := range cards {
		if n < 0 {
			panic(c)
		}
	}
}
