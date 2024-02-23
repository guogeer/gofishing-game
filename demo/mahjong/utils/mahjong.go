package utils

import (
	"fmt"
	"gofishing-game/internal/cardutils"
	"strconv"
	"strings"

	"github.com/guogeer/quasar/log"
	"github.com/guogeer/quasar/util"
)

const (
	MahjongGhostCard = 250
	MaxMahjongCard   = MahjongGhostCard + 10
)

// 操作
const (
	_                = iota
	OperateWin       // 胡
	OperateKong      // 杠
	OperatePong      // 碰
	OperateHuaZhu    // 花猪
	OperateDaJiao    // 大叫
	OperateBackKong  // 返还
	OperateMoveKong  // 呼叫转移
	OperateReadyHand // 听牌
	OperateDouble    // 加倍
	OperateDraw      // 摸牌
	OperateFlower    // 补花
	OperateDiscard   // 出牌
	OperateChow      // 吃
)

// 组合类型
const (
	MeldTriplet = iota + 16
	MeldSequence
	MeldStraightKong
	MeldBentKong
	MeldInvisibleKong
	MeldVisibleTriplet
	MeldInvisibleTriplet
)

func IsValueSame(c int, some ...int) bool {
	for _, v := range some {
		if v == c%10 {
			return true
		}
	}
	return false
}

type WinOption struct {
	WinCard int
	Score   int `json:",omitempty"`
	Points  int `json:",omitempty"`

	Qidui      bool `json:"-"` // 七对
	Shisanyao  bool `json:"-"` // 十三幺
	Yitiaolong bool `json:"-"` // 一条龙
	Jiangyise  bool `json:"-"` // 将一色
	Qingyise   bool `json:"-"` // 清一色
	Duiduihu   bool `json:"-"` // 对对胡

	Pair2   int `json:"-"` // 手牌中某一个牌有两对
	KongNum int `json:"-"` // 杠
}

type Meld struct {
	Card, Type int

	SeatId int // 对方座位ID
}

type SplitOption struct {
	Extra []int
	Melds []Meld
}

type MahjongHelper struct {
	AnyCards       []int // 赖子
	ReserveCardNum int   // 最后保留N张牌

	Qidui            bool // 七对
	Qiduilaizizuodui bool // 七对赖子做对
	Shisanyao        bool // 十三幺
	Jiangyise        bool // 将一色

	Qingyise bool // 清一色
	Duiduihu bool // 碰碰胡
}

// 拆牌
func (helper *MahjongHelper) SplitN(cards []int, most int) []SplitOption {
	var extra = make([]int, 0, 8)
	var melds = make([]Meld, 0, 4)
	var opts = make([]SplitOption, 0, 8)
	var allCards = cardutils.GetAllCards()
	var total = len(allCards)

	var dfs func(int)
	dfs = func(k int) {
		if len(extra) > most {
			return
		}
		if k >= total {
			opt := SplitOption{}
			opt.Extra = append(opt.Extra, extra...)
			for i := 0; i+len(extra) < most && i < cards[MahjongGhostCard]; i++ {
				opt.Extra = append(opt.Extra, MahjongGhostCard)
			}
			opt.Melds = append(opt.Melds, melds...)
			opts = append(opts, opt)
			return
		}

		c := allCards[k]
		if n := cards[c]; n >= 0 && len(extra)+n <= most {
			cards[c] = 0
			for i := 0; i < n; i++ {
				extra = append(extra, c)
			}
			dfs(k + 1)
			cards[c] = n
			extra = extra[:len(extra)-n]
		}

		// TODO 优先组成刻子
		if cards[c] > 0 && cards[c]+cards[MahjongGhostCard] > 2 {
			miss := 0
			if cards[c] < 3 {
				miss = 3 - cards[c]
			}
			cards[c] -= 3 - miss
			cards[MahjongGhostCard] -= miss
			melds = append(melds, Meld{Card: c, Type: MeldTriplet})
			dfs(k)
			melds = (melds)[:len(melds)-1]
			cards[c] += 3 - miss
			cards[MahjongGhostCard] += miss
		}

		// 顺子
		var seq, miss int
		for i := 0; i < 3; i++ {
			if cards[c+i] == 0 || c+i == MahjongGhostCard {
				seq = c + i
				miss++
			}
		}
		if miss < 2 && miss <= cards[MahjongGhostCard] {
			cards[c]--
			cards[c+1]--
			cards[c+2]--
			if seq > 0 {
				cards[seq]++
			}
			cards[MahjongGhostCard] -= miss
			melds = append(melds, Meld{Card: c, Type: MeldSequence})
			dfs(k)
			melds = (melds)[:len(melds)-1]
			cards[c]++
			cards[c+1]++
			cards[c+2]++
			if seq > 0 {
				cards[seq]--
			}
			cards[MahjongGhostCard] += miss
		}
	}

	dfs(0)
	return opts
}
func (helper *MahjongHelper) Split(cards []int) []SplitOption {
	return helper.SplitN(cards, 0)
}

func (helper *MahjongHelper) PrintCards(cards []int) {
	a := make([]string, 0, 16)
	for c, n := range cards {
		for k := 0; k < n; k++ {
			a = append(a, strconv.Itoa(c))
		}
	}
	s := fmt.Sprintf("[%s]", strings.Join(a, " "))
	log.Debug(s)
}

// 胡牌
func (helper *MahjongHelper) Win(cards []int, melds []Meld) *WinOption {
	isAbleWin := false
	winOpt := &WinOption{}

	table := make(map[int]int)
	for _, c := range helper.AnyCards {
		table[c] = cards[c]
		cards[MahjongGhostCard] += table[c]
		cards[c] = 0
	}

	duiduihu := true
	for _, meld := range melds {
		if meld.Type == MeldSequence {
			duiduihu = false
		}
	}

	for _, pair := range cardutils.GetAllCards() {
		if cards[pair] > 0 && cards[pair]+cards[MahjongGhostCard] > 1 {
			miss := 0
			if cards[pair] < 2 {
				miss = 2 - cards[pair]
			}
			cards[pair] -= 2 - miss
			cards[MahjongGhostCard] -= miss

			opts := helper.Split(cards)
			// 对对胡
			for _, opt := range opts {
				duiduihu2 := true
				for _, meld := range opt.Melds {
					if meld.Type == MeldSequence {
						duiduihu2 = false
					}
				}
				if duiduihu && duiduihu2 {
					winOpt.Duiduihu = true
				}

				// 一条龙
				seqs := make(map[int]bool)
				for _, m := range melds {
					if m.Type == MeldSequence {
						seqs[m.Card] = true
					}
				}
				for _, m := range opt.Melds {
					if m.Type == MeldSequence {
						seqs[m.Card] = true
					}
				}
				for _, c := range []int{1, 21, 41} {
					if seqs[c] && seqs[c+3] && seqs[c+6] {
						winOpt.Yitiaolong = true
					}
				}
			}

			if len(opts) > 0 {
				isAbleWin = true
			}
			cards[pair] += 2 - miss
			cards[MahjongGhostCard] += miss
		}
	}

	// 七对
	if helper.Qidui && len(melds) == 0 {
		if helper.Qiduilaizizuodui && cards[MahjongGhostCard] > 0 {
			cards[MahjongGhostCard] = 0
			for _, c := range helper.AnyCards {
				cards[c] = table[c]
			}
		}

		pairNum := 0
		for _, c := range cardutils.GetAllCards() {
			pairNum += cards[c] / 2
		}
		if pairNum+cards[MahjongGhostCard] > 6 {
			isAbleWin = true
			winOpt.Qidui = true
		}
		cards[MahjongGhostCard] = 0
		for _, c := range helper.AnyCards {
			cards[MahjongGhostCard] += table[c]
			cards[c] = 0
		}
	}

	// 将一色
	if helper.Jiangyise {
		jiangyise := true
		for _, m := range melds {
			if m.Type == MeldSequence || !IsValueSame(m.Card, 2, 5, 8) {
				jiangyise = false
				break
			}
		}
		if jiangyise {
			for _, c := range cardutils.GetAllCards() {
				if cards[c] > 0 && !IsValueSame(c, 2, 5, 8) && util.InArray(helper.AnyCards, c) == 0 {
					jiangyise = false
					break
				}
			}
		}
		if jiangyise {
			isAbleWin = true
			winOpt.Jiangyise = true
		}
	}
	// 十三幺
	if helper.Shisanyao {
		counter := 0
		for _, c := range []int{60, 70, 80, 90, 100, 110, 120, 1, 9, 21, 29, 41, 49} {
			if cards[c] > 0 {
				counter++
			}
		}
		if counter+cards[MahjongGhostCard] >= 13 {
			isAbleWin = true
			winOpt.Shisanyao = true
		}
	}

	var pair2, kongNum, color int
	for _, c := range cardutils.GetAllCards() {
		if cards[c] > 0 {
			color = color | int(1<<uint(c/10))
		}
		pair2 += cards[c] / 4
	}
	for _, m := range melds {
		color = color | int(1<<uint(m.Card/10))
		switch m.Type {
		case MeldInvisibleKong, MeldStraightKong, MeldBentKong:
			kongNum++
		}
	}
	// reset cards
	cards[MahjongGhostCard] = 0
	for _, c := range helper.AnyCards {
		cards[c] = table[c]
	}

	if !isAbleWin {
		return nil
	}

	winOpt.Pair2 = pair2
	winOpt.KongNum = kongNum
	winOpt.Qingyise = (color&(color-1) == 0) // 清一色
	return winOpt
}

type Context struct {
	Cards      []int
	Melds      []Meld
	ValidCards []int
	DrawNum    int // 摸牌次数
}

var (
	// 期望胡清一色时，其他花色的牌存在N张时差异值
	gQingyisechayizhi = []int{0, 1, 2, 6, 8, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64}

	// 期望胡清一色时，第N次摸牌时候，打出去一张其他花色的牌后，手牌剩余的其他花色的牌
	gQingyisezapai = []int{4, 4, 4, 3, 3, 3, 3, 2, 2, 2, 2, 1, 1, 1, 1}

	gPaixing3  = []int{-2, -1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	gPaixing34 = []int{
		0, 1, 3, 3, 5, 6, 7, 8, // 0-7
		9, 10, 11, 12, 13, 14, 15, 16, // 8-15
		17, 18, 19, 20, 21, 22, 23, 24, 25,
	}
	gPaixing344 = [][]int{
		{2, 3, 5, 6, 6, 7, 8, 8, 8, 8, 8, 8},
		{4, 4, 6, 8, 8, 10, 10, 10, 10, 10, 10, 10},
		{10, 10, 10, 10, 12, 12, 12, 12, 12, 12, 12, 12},
	}
)

// 计算权重
func (helper *MahjongHelper) Weight(ctx *Context) (int, int) {
	cards := ctx.Cards
	validCards := ctx.ValidCards

	best := 0
	expectColors := 0
	result := make(map[int]bool)
	extraCards := make(map[int]int)
	for _, m := range ctx.Melds {
		expectColors |= 1 << uint(m.Card/10)
	}

	if expectColors == 0 {
		for _, color := range []int{0, 2, 4} {
			counter := 0
			for _, c := range cardutils.GetAllCards() {
				if color == c/10 {
					counter += cards[c]
				}
			}
			// 手牌至少7张才选择清一色
			if counter >= 7 {
				expectColors = 1 << uint(color)
			}
		}
	}

	for _, dc := range cardutils.GetAllCards() {
		if cards[dc] >= 3 || cards[dc] <= 0 || dc == MahjongGhostCard {
			continue
		}
		diff := 0
		colors := expectColors
		prices := gQingyisechayizhi
		for _, c := range cardutils.GetAllCards() {
			if cards[c] > 0 && colors != 1<<uint(c/10) {
				diff += prices[cards[c]]
			}
		}
		maxDiff := 0
		if ctx.DrawNum < len(gQingyisezapai) {
			maxDiff = gQingyisezapai[ctx.DrawNum]
		}

		addition0 := 0
		if helper.Qingyise && (diff == 0 || diff <= maxDiff) && colors != 1<<uint(dc/10) {
			addition0 += 300
		}
		for _, m := range ctx.Melds {
			if m.Type == MeldVisibleTriplet {
				addition0 += 10
			}
		}

		cards[dc]--
		opts := helper.SplitN(cards, 64)
		cards[dc]++

		for _, opt := range opts {
			addition1 := 0
			for _, m := range opt.Melds {
				if m.Type == MeldTriplet {
					addition1 += 30
				}
			}
			for i := range extraCards {
				extraCards[i] = 0
			}
			for _, c := range opt.Extra {
				extraCards[c]++
			}
			weight := 0
			firstPair := 0
			extraNum := len(opt.Extra)
			// 牌型例子
			// (0) 3
			// (1) 3 3
			// (2) 3 4/1 2/8 9/3 5
			// (3) 3 4 4/3 3 4/3 5 5
			for _, c := range cardutils.GetAllCards() {
				if extraCards[c-1]*extraCards[c+1] == 0 && extraCards[c] == 2 {
					firstPair = 90
				}
			}
			for _, c := range cardutils.GetAllCards() {
				if extraCards[c] <= 0 || extraCards[c] > 2 {
					continue
				}

				// 3 4 4/3 3 4/3 5 5/3 3 5
				// log.Debug(c, extraCards[c], extraCards[c+1], weight)
				if extraCards[c]+extraCards[c+1]+extraCards[c+2] > 2 {
					pair := c
					if extraCards[c+2] > 1 {
						pair = c + 2
					}
					if extraCards[c+1] > 1 {
						pair = c + 1
					}
					if extraCards[c] > 1 {
						pair = c
					}
					single := 0
					if extraCards[c+1] == 0 {
						single = validCards[c+1]
					}
					if extraCards[c+1] > 0 {
						single = validCards[c-1] + validCards[c+2]
					}
					// log.Debug(pair, single)
					if extraCards[pair] == 2 {
						weight += gPaixing344[validCards[pair]][single]
						extraCards[pair] = 1 // 对子下次不考虑
					}
				} else if extraCards[c] == 2 {
					weight += gPaixing344[validCards[c]][0]
				} else if extraCards[c]+extraCards[c+1]+extraCards[c+2] == 2 {
					single := 0
					// 3 5
					if extraCards[c+1] == 0 {
						single = validCards[c+1]
					}
					// 3 4
					if extraCards[c+1] > 0 {
						single = validCards[c-1] + validCards[c+2]
					}
					// 3 4 5 6 7
					if extraCards[c]+extraCards[c+1] == 2 {
						for _, m := range opt.Melds {
							if m.Type == MeldSequence && m.Card == c+2 {
								// log.Debug("========", validCards[c+5])
								single = validCards[c-1] + validCards[c+2]
								single += validCards[c+5]
							}
						}
					}
					weight += gPaixing34[single]
				} else if extraCards[c] == 1 {
					weight += gPaixing3[c%10]
				}
			}
			sum := (16-extraNum)*30 + firstPair + addition0 + addition1 + weight
			if best < sum {
				for c := range result {
					result[c] = false
				}
			}
			// log.Debug(dc, sum, opt.Extra, firstPair, addition0, addition1, weight)
			if best <= sum {
				result[dc] = true
				best = sum
			}
		}
	}

	// 内置map数组随机
	for c, b := range result {
		if b {
			return c, best
		}
	}
	return -1, 0
}
