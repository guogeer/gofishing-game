package cardcontrol

import (
	"fmt"
	"gofishing-game/service"

	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/utils/randutils"
)

// 暗池
type InvisiblePrizePool struct {
	subId int
	Cap   int64 `json:"cap,omitempty"`
	Tax   int64 `json:"tax,omitempty"`
}

func NewInvisiblePrizePool(subId int) *InvisiblePrizePool {
	pp := &InvisiblePrizePool{subId: subId}
	key := fmt.Sprintf("invisible_prize_pool_%d", subId)
	service.UpdateDict(key, pp)
	return pp
}

// n>0：表示玩家赢取
func (pp *InvisiblePrizePool) Add(n int64) {
	value := -n
	percent, _ := config.Float("lottery", pp.subId, "invisiblePrizePoolPercent")

	var tax int64
	if value > 0 {
		tax = int64(float64(value) * percent / 100)
		value = value - tax
	}
	pp.Cap += value
	pp.Tax += tax
}

// -1：吐分
// 0：正常
// 1：吃分
func (pp *InvisiblePrizePool) Check() int {
	subId := pp.subId
	line, _ := config.Int("lottery", subId, "warningLine")

	percent, _ := config.Float("lottery", subId, "prizePoolControlPercent")
	if randutils.IsPercentNice(percent) == false {
		return 0
	}
	n := pp.Cap
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
	line, _ := config.Int("lottery", subId, "WarningLine")

	cur := pp.Cap
	if n-cur >= line {
		return false
	}
	return true
}

// 奖池
type PrizePool struct {
	Rank      []RankUserInfo `json:"rank,omitempty"`
	subId     int
	rankLen   int
	Cap       int64 `json:"cap,omitempty"`
	LastPrize int64 `json:"lastPrize,omitempty"`
}

func NewPrizePool(subId int) *PrizePool {
	pool := &PrizePool{
		Rank:    make([]RankUserInfo, 0, 10),
		rankLen: 3,
		subId:   subId,
	}
	service.UpdateDict("lottery_prize_pool", pool)
	return pool
}

func (pool *PrizePool) Add(n int64) int64 {
	pool.Cap += n

	limit, _ := config.Int("lottery", pool.subId, "prizePoolLimit")
	if limit > 0 && pool.Cap > limit {
		pool.Cap = limit
	}
	return pool.Cap
}

func (pool *PrizePool) SetLastPrize(n int64) {
	pool.LastPrize = n
}

func (pool *PrizePool) GetLastPrize() int64 {
	return pool.LastPrize
}

func (pool *PrizePool) ClearRank() {
	if pool.Rank != nil {
		pool.Rank = pool.Rank[:0]
	}
}

func (pool *PrizePool) UpdateRank(user service.UserInfo, gold int64) {
	rankList := &RankList{top: pool.Rank, len: pool.rankLen}
	if rankList.Update(user, gold) != nil {
		pool.Rank = rankList.top
	}
}
