package utils

import (
	"fmt"

	"github.com/guogeer/quasar/config"
)

type InvisiblePrizePool struct {
	subId int
}

// n>0：表示玩家赢取
func (pp *InvisiblePrizePool) Add(n int64) {
	res := -n
	percent, _ := config.Float("entertainment", pp.subId, "InvisiblePrizePoolPercent")

	var tax int64
	if res > 0 {
		tax = int64(float64(res) * percent / 100)
		res = res - tax
	}
	key := fmt.Sprintf("invisible_prize_pool_%d", pp.subId)
	// old := ServiceConfig().Int(key)
	// log.Debug("current invisible prize pool", old, res)
	ServiceConfig().Add(key, res)
	key = fmt.Sprintf("daily.invisible_prize_pool_%d_tax", pp.subId)
	ServiceConfig().Add(key, tax)
}

// -1：吐分
//
//	0：正常
//	1：吃分
func (pp *InvisiblePrizePool) Check() int {
	subId := pp.subId
	line, _ := config.Int("entertainment", subId, "WarningLine")

	percent, _ := config.Float("entertainment", subId, "PrizePoolControlPercent")
	if randutil.IsPercentNice(percent) == false {
		return 0
	}
	key := fmt.Sprintf("invisible_prize_pool_%d", subId)
	n := ServiceConfig().Int(key)
	switch {
	case n <= -line:
		return -1
	case n >= line:
		return 1
	}
	return 0
}

// n>0：表示玩家赢取
func (pp *InvisiblePrizePool) IsValid(n int64) bool {
	subId := pp.subId
	line, _ := config.Int("entertainment", subId, "WarningLine")

	key := fmt.Sprintf("invisible_prize_pool_%d", subId)
	cur := ServiceConfig().Int(key)
	if n-cur >= line {
		return false
	}
	return true
}

// 奖池
type PrizePool struct {
	Rank         []RankUserInfo
	subId        int
	prizePoolKey string
	rankKey      string
	lastPrizeKey string
	rankLen      int
}

func NewPrizePool(subId int) *PrizePool {
	pool := &PrizePool{
		Rank:         make([]RankUserInfo, 0, 10),
		rankLen:      3,
		subId:        subId,
		prizePoolKey: fmt.Sprintf("prize_pool_gold_%d", subId),
		rankKey:      fmt.Sprintf("prize_pool_rank_%d", subId),
		lastPrizeKey: fmt.Sprintf("last_prize_gold_%d", subId),
	}
	// 兼容水果机
	if GetName() == "sgj" {
		pool.prizePoolKey = "shuiguoji_prize_pool"
	}
	return pool
}

func (pool *PrizePool) Add(n int64) int64 {
	newPrize := ServiceConfig().Add(pool.prizePoolKey, n)

	limit, _ := config.Int("entertainment", pool.subId, "PrizePoolLimit")
	if limit > 0 && newPrize > limit {
		return ServiceConfig().Add(pool.prizePoolKey, limit-newPrize)
	}
	return newPrize
}

func (pool *PrizePool) SetLastPrize(n int64) {
	ServiceConfig().Set(pool.lastPrizeKey, n)
}

func (pool *PrizePool) LastPrize() int64 {
	return ServiceConfig().Add(pool.lastPrizeKey, 0)
}

func (pool *PrizePool) ClearRank() {
	if pool.Rank != nil {
		pool.Rank = pool.Rank[:0]
	}
}

func (pool *PrizePool) UpdateRank(user *SimpleUserInfo, gold int64) {
	rankList := &RankList{top: pool.Rank, len: pool.rankLen}
	if rankList.Update(user, gold) != nil {
		pool.Rank = rankList.top
		ServiceConfig().Set(pool.rankKey, pool.Rank)
	}
}
