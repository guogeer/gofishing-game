package utils

import (
	"math/rand"
	"sort"

	"github.com/guogeer/quasar/utils/randutils"
)

// /////////////////////////////////////////////////////////
// 庄家作弊
func HelpDealer(data sort.Interface, percent float64) {
	var rank int
	for i := 0; i+1 < data.Len(); i++ {
		if randutils.IsPercentNice(percent) {
			rank++
		}
	}
	sort.Sort(data)
	data.Swap(rank, data.Len()-1)
	for i := 0; i+1 < data.Len(); i++ {
		end := data.Len() - 1 - i
		t := rand.Intn(end)
		data.Swap(t, end-1)
	}
}

type RankUserInfo struct {
	SimpleUserInfo
	Award string
	Prize int64 `json:",omitempty"`
}

type RankList struct {
	top []RankUserInfo
	len int
}

func NewRankList(q []RankUserInfo, n int) *RankList {
	return &RankList{top: q, len: n}
}

func (lst *RankList) GetRank(uid int) int {
	for i, user := range lst.top {
		if user.Id == uid {
			return i
		}
	}
	return -1
}

func (lst *RankList) Top() []RankUserInfo {
	return lst.top
}

func (lst *RankList) Update(user *SimpleUserInfo, gold int64) *RankUserInfo {
	if gold == 0 {
		return nil
	}

	rank := lst.top
	pos := len(rank)
	uid := user.Id
	// 如果玩家已经在排行榜中，先移除
	for k := 0; k < len(rank); k++ {
		if rank[k].Id == uid {
			pos = k
			break
		}
	}
	// 前移一位
	for k := pos; k+1 < len(rank); k++ {
		rank[k] = rank[k+1]
	}
	if n := len(rank); pos < n {
		rank = rank[:n-1]
	}

	pos = len(rank)
	// 从第一名开始遍历一个金币更少的
	for k := 0; k < len(rank); k++ {
		if rank[k].Gold < gold {
			pos = k
			break
		}
	}

	if len(rank) < lst.len {
		rank = append(rank, RankUserInfo{})
	}
	// 后移一位
	for k := len(rank) - 2; k >= pos; k-- {
		rank[k+1] = rank[k]
	}
	if pos < len(rank) {
		rank[pos] = RankUserInfo{SimpleUserInfo: *user}
		rank[pos].Gold = gold
		rank[pos].Prize = gold
		lst.top = rank
		// log.Debug("update rank list", lst.top)
		return &rank[pos]
	}
	return nil
}
