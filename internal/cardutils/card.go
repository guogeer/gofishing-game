/*
	0x01~0x0e 方片2~K、A
	0x11~0x1e 梅花2~K、A
	0x21~0x2e 红桃2~K、A
	0x31~0x3e 黑桃2~K、A
	0xf0、0xf1 小鬼、大鬼
*/

package cardutils

import (
	"github.com/guogeer/quasar/randutil"
)

var (
	defaultCardSystem = &CardSystem{}
)

/*****************************************************************
 * 发牌
 *****************************************************************/
type CardSystem struct {
	index, allCards []int
	reserve         int // 留下多少张牌不摸
}

func (sys *CardSystem) Init(cards []int) {
	sys.allCards = nil
	sys.index = make([]int, 512)
	for _, c := range cards {
		sys.index[c]++
	}
	for c, n := range sys.index {
		if n > 0 {
			sys.allCards = append(sys.allCards, c)
		}
	}
}

func GetCardSystem() *CardSystem {
	return defaultCardSystem
}

type CardSet struct {
	randCards  []int        // 洗好的牌
	extraCards []int        // 额外增加的牌
	blackList  map[int]bool // 黑名单
	dealNum    int          // 已发的牌数
}

func NewCardSet() *CardSet {
	cs := &CardSet{
		blackList:  make(map[int]bool),
		extraCards: make([]int, 0, 16),
	}
	cs.Shuffle()
	return cs
}

// 洗牌
func (cs *CardSet) Shuffle() {
	if len(cs.randCards) > 0 {
		cs.randCards = cs.randCards[:0]
	}
	for c, n := range defaultCardSystem.index {
		if _, ok := cs.blackList[c]; ok {
			continue
		}
		for i := 0; i < n; i++ {
			cs.randCards = append(cs.randCards, c)
		}
	}
	cs.randCards = append(cs.randCards, cs.extraCards...)

	randutil.Shuffle(cs.randCards)
	cs.dealNum = 0
}

// 发牌
func (cs *CardSet) Deal() int {
	sys := GetCardSystem()
	if cs.dealNum+sys.reserve >= len(cs.randCards) {
		return -1
	}
	c := cs.randCards[cs.dealNum]
	cs.dealNum++
	return c
}

// 作弊
func (cs *CardSet) Cheat(some ...int) int {
	counter := 0
	for _, c := range some {
		cards := cs.randCards[cs.dealNum:]
		for i, c1 := range cards {
			if c == c1 {
				cards[0], cards[i] = cards[i], cards[0]
				cs.dealNum++
				counter++
				break
			}
		}
	}
	return counter
}
