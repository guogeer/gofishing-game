/*
	0x01~0x0e 方片2~K、A
	0x11~0x1e 梅花2~K、A
	0x21~0x2e 红桃2~K、A
	0x31~0x3e 黑桃2~K、A
	0xf0、0xf1 小鬼、大鬼
*/

package cardutils

import (
	"fmt"

	"github.com/guogeer/quasar/utils/randutils"
)

var allCardSystem = map[string]*CardSystem{}

/*****************************************************************
 * 测试用例
 *****************************************************************/

type TestCase []int

func (t *TestCase) Load(a []int) {
	*t = append([]int{}, a...)
}

/*****************************************************************
 * 发牌
 *****************************************************************/
type CardSystem struct {
	table    []int
	allCards []int
	reserve  int  // 留下多少张牌不摸
	isHide   bool // 隐藏保留的牌
	TestCase TestCase
}

func (sys *CardSystem) Table() []int {
	return sys.table
}

func AddCardSystem(name string, cards []int) *CardSystem {
	sys := &CardSystem{
		table: make([]int, 512),
	}

	for _, c := range cards {
		sys.table[c]++
	}
	for c, n := range sys.table {
		if n > 0 {
			sys.allCards = append(sys.allCards, c)
		}
	}
	allCardSystem[name] = sys
	return sys
}

func (sys *CardSystem) GetAllCards() []int {
	return sys.allCards
}

func (sys *CardSystem) IsCardValid(c int) bool {
	if idx := sys.table; c > 0 && c < len(idx) && idx[c] > 0 {
		return true
	}
	return false
}

// 保留N张牌
func (sys *CardSystem) Reserve(n int) {
	sys.reserve = n
}

// 保留N张牌并隐藏
func (sys *CardSystem) ReserveAndHide(n int) {
	sys.Reserve(n)
	sys.isHide = true
}

func GetCardSystem(name string) *CardSystem {
	return allCardSystem[name]
}

type CardSet struct {
	name string

	randCards  []int        // 洗好的牌
	extraCards []int        // 额外增加的牌
	blackList  map[int]bool // 黑名单
	dealNum    int          // 已发的牌数
}

func NewCardSet(name string) *CardSet {
	cs := &CardSet{
		name:       name,
		blackList:  make(map[int]bool),
		extraCards: make([]int, 0, 16),
	}
	cs.Shuffle()
	return cs
}

func (cs *CardSet) Total() int {
	sys := GetCardSystem(cs.name)
	n := len(cs.randCards)
	if sys.isHide {
		n = n - sys.reserve
	}
	return n
}

// 剩余牌数
func (cs *CardSet) Count() int {
	return cs.Total() - cs.dealNum
}

// 洗牌
func (cs *CardSet) Shuffle() {
	if len(cs.randCards) > 0 {
		cs.randCards = cs.randCards[:0]
	}
	cardSys := GetCardSystem(cs.name)
	for c, n := range cardSys.table {
		if _, ok := cs.blackList[c]; ok {
			continue
		}
		for i := 0; i < n; i++ {
			cs.randCards = append(cs.randCards, c)
		}
	}
	cs.randCards = append(cs.randCards, cs.extraCards...)

	randutils.Shuffle(cs.randCards)
	cs.dealNum = 0
}

func (cs *CardSet) GetRemainingCards() []int {
	var cards [512]int
	for i := cs.dealNum; i < len(cs.randCards); i++ {
		c := cs.randCards[i]
		cards[c]++
	}
	return cards[:]
}

// 发牌
func (cs *CardSet) Deal() int {
	sys := GetCardSystem(cs.name)
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

func (cs *CardSet) IsCardValid(c int) bool {
	if _, ok := cs.blackList[c]; ok {
		return false
	}
	return GetCardSystem(cs.name).IsCardValid(c)
}

func (cs *CardSet) Recover(some ...int) {
	for _, c := range some {
		delete(cs.blackList, c)
	}
	cs.Shuffle()
}

// 鬼
func IsCardGhost(c int) bool {
	return c == 0xf0 || c == 0xf1
}

// 移动到末尾去
func (cs *CardSet) MoveBack(someCards []int) {
	if GetCardSystem(cs.name).TestCase != nil {
		return
	}

	counter := 0
	cards := cs.randCards[cs.dealNum:]
	for _, c := range someCards {
		for i := range cards {
			if c == cards[i] && i+counter < len(cards) {
				k := len(cards) - 1 - counter
				cards[i], cards[k] = cards[k], cards[i]
				counter++
				break
			}
		}
	}
}

// 移动到头部去
func (cs *CardSet) MoveFront(someCards ...int) {
	if GetCardSystem(cs.name).TestCase != nil {
		return
	}

	counter := 0
	cards := cs.randCards[cs.dealNum:]
	for _, c := range someCards {
		back := len(cards) - 1
		if counter < len(cards) && c == cards[back] {
			for i := back; i > counter; i-- {
				cards[i] = cards[i-1]
			}
			cards[counter] = c
			counter++
		}
	}
}

func (cs *CardSet) Remove(some ...int) {
	for _, c := range some {
		cs.blackList[c] = true
	}
	cs.Shuffle()
}

func (sys *CardSystem) IsColorValid(color int) bool {
	return sys.IsCardValid(10*color + 1)
}

// 格式化扑克
func Format(cards []int) string {
	output := "[]int{"
	for i, c := range cards {
		if i == 0 {
			output += fmt.Sprintf("0x%02x", c)
		} else {
			output += fmt.Sprintf(",0x%02x", c)
		}
	}
	output += "}"
	return output
}

func Print(cards []int) {
	fmt.Println(Format(cards))
}
