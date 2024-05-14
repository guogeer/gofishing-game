package cardrule

/*
	0x01~0x0a 一到十
	0x21~0x2a 壹到拾
*/

import (
	"gofishing-game/internal/cardutils"
	"sort"

	"github.com/guogeer/quasar/log"
)

// 操作
const (
	PaohuziWin       = iota + 1
	PaohuziDraw      // 摸牌
	PaohuziChow      // 吃
	PaohuziPong      // 碰
	PaohuziKong      // 杠
	PaohuziReadyHand // 听牌
)

const (
	PaohuziNone             = iota
	PaohuziSequence         // 顺子
	PaohuziStraightKong     // 直杠
	PaohuziBentKong         // 弯杠
	PaohuziVisibleKong      // 明杠
	PaohuziInvisibleKong    // 暗杠
	PaohuziVisibleTriplet   // 明刻
	PaohuziInvisibleTriplet // 暗刻
)

type PaohuziMeld struct {
	Cards []int
	Type  int
	Score int
}

type PaohuziHelper struct{}

func NewPaohuziHelper() *PaohuziHelper {
	return &PaohuziHelper{}
}

func (helper *PaohuziHelper) IsAbleChow(cards []int, sample [][3]int, chow int) bool {
	var cardSet [256]int
	for _, c := range cards {
		cardSet[c]++
	}
	for _, tri := range sample {
		found := false
		for _, c := range tri {
			if cardutils.IsCardValid(c) == false {
				return false
			}
			if c == chow {
				found = true
			}
		}
		if found == false {
			return false
		}
	}
	for _, c := range cards {
		cardSet[c]++
	}
	for _, tri := range sample {
		m := helper.NewMeld(tri[:])
		if m.Type != PaohuziSequence {
			return false
		}
		for _, c := range tri {
			cardSet[c]--
		}
	}
	for _, c := range cardutils.GetAllCards() {
		if cardSet[c] < 0 {
			return false
		}
	}
	return cardSet[chow] == 0
}

func (helper *PaohuziHelper) TryChow(cards []int, chowCard int) [][]PaohuziMeld {
	var cardSet [256]int
	for _, c := range cards {
		cardSet[c]++
	}
	cardSet[chowCard]++

	melds := make([]PaohuziMeld, 0, 4)
	samples := make([][]PaohuziMeld, 0, 1)
	chowMelds := helper.GetChowMelds(chowCard)
	log.Debug(cards)

	var dfs func(int)
	dfs = func(n int) {
		if n == len(chowMelds) {
			if cardSet[chowCard] == 0 {
				copyMelds := make([]PaohuziMeld, len(melds))
				copy(copyMelds, melds)
				samples = append(samples, copyMelds)
			}
			return
		}
		for _, c := range chowMelds[n].Cards {
			if cardutils.IsCardValid(c) == false || cardSet[c] < 0 || cardSet[c] > 2 {
				return
			}
		}

		dfs(n + 1)

		for _, c := range chowMelds[n].Cards {
			cardSet[c]--
		}
		melds = append(melds, chowMelds[n])
		dfs(n)
		melds = melds[:len(melds)-1]
		for _, c := range chowMelds[n].Cards {
			cardSet[c]++
		}
	}
	dfs(0)
	return samples
}

func (helper *PaohuziHelper) GetChowMelds(chowCard int) []PaohuziMeld {
	oppositeCard := chowCard & 0x0f // 一=>壹、壹=>一
	if oppositeCard == chowCard {
		oppositeCard = chowCard | 0x10
	}

	options := [][]int{
		[]int{0x02, 0x07, 0x0a},
		[]int{0x12, 0x17, 0x1a},
		[]int{chowCard - 2, chowCard - 1, chowCard},
		[]int{chowCard - 1, chowCard, chowCard + 1},
		[]int{chowCard, chowCard + 1, chowCard + 2},
		[]int{chowCard & 0x0f, chowCard | 0x10, chowCard},
		[]int{oppositeCard, oppositeCard, chowCard},
	}

	melds := make([]PaohuziMeld, 0, 8)
	for _, opt := range options {
		var found, invalid bool
		for _, c := range opt {
			if c == chowCard {
				found = true
			}
			if cardutils.IsCardValid(c) == false {
				invalid = true
			}
		}
		if found && invalid == false {
			for _, opt := range options {
				sort.Ints(opt)
				for i, c := range opt {
					if c == chowCard {
						opt[0], opt[i] = opt[i], opt[0]
					}
				}
			}

			melds = append(melds, helper.NewMeld(opt))
		}
	}

	return melds
}

func (helper *PaohuziHelper) NewMeld(cards []int) PaohuziMeld {
	c, score, typ := cards[0], 0, PaohuziNone
	if len(cards) == 4 && cards[1] == 0 {
		score = 12
		if c&0xf0 == 0x00 {
			score = 9
		}
		typ = PaohuziInvisibleKong
	} else if len(cards) == 4 && cards[0] == cards[1] {
		score = 9
		if c&0xf0 == 0x00 {
			score = 6
		}
		typ = PaohuziVisibleKong
	} else if len(cards) == 3 && cards[1] == 0 {
		score = 6
		if c&0xf0 == 0x00 {
			score = 3
		}
		typ = PaohuziInvisibleTriplet
	} else if len(cards) == 3 && cards[0] == cards[1] {
		score = 3
		if c&0xf0 == 0x00 {
			score = 1
		}
		typ = PaohuziVisibleTriplet
	} else if len(cards) == 3 {
		var tri [3]int
		copy(tri[:], cards)
		// 先将三张牌按大小排序
		sort.Ints(tri[:])
		// 三四五
		if tri[0]+1 == tri[1] && tri[0]+2 == tri[2] {
			score = 0
		}
		// 一一壹
		if tri[0] == tri[1] && tri[0]&0x0f == tri[2]&0x0f {
			score = 0
		}
		// 一壹壹
		if tri[1] == tri[2] && tri[0]&0x0f == tri[1]&0x0f {
			score = 0
		}
		// 二七十
		if tri[0] == 0x02 && tri[1] == 0x07 && tri[2] == 0x0a {
			score = 3
		}
		// 贰柒拾
		if tri[0] == 0x12 && tri[1] == 0x17 && tri[2] == 0x1a {
			score = 6
		}
		// 一二三
		if tri[0] == 0x01 && tri[1] == 0x02 && tri[2] == 0x03 {
			score = 3
		}
		// 壹贰叁
		if tri[0] == 0x11 && tri[1] == 0x12 && tri[2] == 0x13 {
			score = 6
		}
		typ = PaohuziSequence
	}
	return PaohuziMeld{Cards: cards, Type: typ, Score: score}
}

func (helper *PaohuziHelper) Sum(melds []PaohuziMeld) int {
	sum := 0
	for _, meld := range melds {
		sum += meld.Score
	}
	return sum
}

type PaohuziSplitOption struct {
	Melds []PaohuziMeld
	Pair  int
}

// 刻、杠已提前预处理
// 剩下一对或者不剩，忽略跑胡
func (helper *PaohuziHelper) TryWin(cards []int) (opt *PaohuziSplitOption) {
	var cardSet [256]int
	for _, c := range cards {
		cardSet[c]++
	}

	var pair = -1
	var melds = make([]PaohuziMeld, 0, 4)
	var allCards = cardutils.GetAllCards()
	var total = len(allCards)

	var dfs func(int)
	dfs = func(k int) {
		if k >= total {
			if opt == nil || helper.Sum(opt.Melds) < helper.Sum(melds) {
				opt = &PaohuziSplitOption{
					Pair: pair,
				}
				opt.Melds = append(opt.Melds, melds...)
			}
			return
		}
		c := allCards[k]
		if cardSet[c] == 0 {
			dfs(k + 1)
			return
		}
		if cardSet[c] > 2 {
			meld := helper.NewMeld([]int{c, 0, 0})
			cardSet[c] -= 3
			melds = append(melds, meld)
			dfs(k)
			cardSet[c] += 3
			melds = melds[:len(melds)-1]
		}
		// 顺子
		for _, meld := range helper.GetChowMelds(c) {
			enough := true
			for _, seq := range meld.Cards {
				if cardutils.IsCardValid(seq) == false || cardSet[seq] < 1 || cardSet[seq] == 3 {
					enough = false
				}
			}
			if enough == true {
				for _, seq := range meld.Cards {
					cardSet[seq]--
				}
				melds = append(melds, meld)
				dfs(k)
				melds = (melds)[:len(melds)-1]
				for _, seq := range meld.Cards {
					cardSet[seq]++
				}
			}
		}
	}

	if mod := len(cards) % 3; mod == 0 {
		pair = -1
		dfs(0)
	} else if mod == 2 {
		for c, n := range cardSet {
			if n == 2 {
				pair, cardSet[c] = c, 0
				dfs(0)
				pair, cardSet[c] = -1, 2
			}
		}
	} else {
		panic("unkown error")
	}
	return
}
