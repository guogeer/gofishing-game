package cardrule

import (
	"gofishing-game/internal/cardutils"
	"math/rand"

	"github.com/guogeer/quasar/utils/randutils"
)

var (
	niuniuWeights = []int{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, // 没牛~牛牛
		50, // 五小牛
		40, // 炸弹牛
		30, // 五花牛
		29, // 四花牛
	}
)

const (
	NNWuXiaoNiu = iota + 10
	NNSiHuaNiu
	NNWuHuaNiu
	NNZhaDanNiu
)

type NiuNiuHelper struct {
	options map[int]bool
}

func NewNiuNiuHelper() *NiuNiuHelper {
	return &NiuNiuHelper{
		options: make(map[int]bool),
	}
}

func (helper *NiuNiuHelper) SetOption(opt int) {
	helper.options[opt] = true
}

func (helper *NiuNiuHelper) GetCardWeight(c int) int {
	w := c & 0x0f

	switch w {
	// J、Q、K
	case 11, 12, 13:
		return 10
	// A
	case 14:
		return 1
	case 15:
		panic("unkown card")
	}
	return w
}

func (helper *NiuNiuHelper) Weight(cards []int) (int, []int) {
	var tri [3]int
	var sum, weight int
	for _, c := range cards {
		sum += helper.GetCardWeight(c)
	}

	num := len(cards)
	for i := 0; i < num; i++ {
		for j := i + 1; j < num; j++ {
			for k := j + 1; k < num; k++ {
				triSum := 0
				for _, x := range []int{i, j, k} {
					triSum += helper.GetCardWeight(cards[x])
				}

				w := (sum - triSum) % 10
				// 牛牛
				if w == 0 {
					w = 10
				}
				if triSum%10 == 0 && weight < w {
					weight = w
					copy(tri[:], []int{i, j, k})
				}
			}
		}
	}

	copyCards := make([]int, len(cards))
	copy(copyCards, cards)
	if weight > 0 {
		for i, v := range tri {
			copyCards[i], copyCards[v] = copyCards[v], copyCards[i]
		}
	}

	extra := make([]int, 0, 4)
	// 四花牛
	if helper.isSiHuaNiu(cards) {
		extra = append(extra, 14)
	}
	// 五花牛
	if helper.isWuHuaNiu(cards) {
		extra = append(extra, 13)
	}
	// 炸弹牛
	if helper.isZhaDanNiu(cards) {
		extra = append(extra, 12)
	}
	// 五小牛
	if helper.isWuXiaoNiu(cards) {
		extra = append(extra, 11)
	}
	for _, n := range extra {
		if helper.LessWeight(weight, n) {
			weight = n
		}
	}
	return weight, copyCards
}

func (helper *NiuNiuHelper) isWuXiaoNiu(cards []int) bool {
	if _, ok := helper.options[NNWuXiaoNiu]; !ok {
		return false
	}

	sum := 0
	for _, c := range cards {
		v := helper.GetCardWeight(c)
		if v >= 5 {
			return false
		}
		sum += v
	}
	return sum <= 10
}

func (helper *NiuNiuHelper) isWuHuaNiu(cards []int) bool {
	if _, ok := helper.options[NNWuHuaNiu]; !ok {
		return false
	}
	for _, c := range cards {
		switch c & 0x0f {
		// J、Q、K
		case 11, 12, 13:
			// do nothing
		default:
			return false
		}
	}
	return true
}

func (helper *NiuNiuHelper) isSiHuaNiu(cards []int) bool {
	if _, ok := helper.options[NNSiHuaNiu]; !ok {
		return false
	}
	var counter int
	for _, c := range cards {
		switch c & 0x0f {
		case 10:
		// J、Q、K
		case 11, 12, 13:
			counter++
		default:
			return false
		}
	}
	return counter == 4
}

// 炸弹牛
// 有四张相同的牌
func (helper *NiuNiuHelper) isZhaDanNiu(cards []int) bool {
	if _, ok := helper.options[NNZhaDanNiu]; !ok {
		return false
	}
	for i := 0; i < len(cards); i++ {
		counter := 0
		for j := 0; j < len(cards); j++ {
			if cards[i]&0x0f == cards[j]&0x0f {
				counter++
			}
		}
		if counter >= 4 {
			return true
		}
	}
	return false
}

// K>Q>J...>2>A
// 黑桃>红桃>梅花>方块
func (helper *NiuNiuHelper) LessCard(c1, c2 int) bool {
	if c1 == c2 {
		panic("unkown error")
	}

	scoreList := []int{
		0, 0,
		2, 3, 4, 5, 6, 7, 8, 9, 10, // 2~10
		11, 12, 13, //  J、Q、K
		1, // A
	}
	// 牌面相同，花色不同
	if c1&0x0f == c2&0x0f && c1 < c2 {
		return true
	}
	// 牌面小于
	if scoreList[c1&0x0f] < scoreList[c2&0x0f] {
		return true
	}
	return false
}

// 最大的牌
func (helper *NiuNiuHelper) GetMaxCard(cards []int) int {
	n := len(cards)

	c := cards[0]
	for i := 1; i < n; i++ {
		if helper.LessCard(c, cards[i]) {
			// log.Debug(c, cards[i])
			c = cards[i]
		}
	}
	return c
}

// 比较牛
// 牛相同，依次比较最大的牌
func (helper *NiuNiuHelper) Less(first, second []int) bool {
	w1, _ := helper.Weight(first)
	w2, _ := helper.Weight(second)
	// 牛相同
	if w1 != w2 {
		return helper.LessWeight(w1, w2)
	}
	return helper.LessMaxCard(first, second)
}

func (helper *NiuNiuHelper) LessWeight(first, second int) bool {
	return niuniuWeights[first] < niuniuWeights[second]
}

// 比较最大的牌
func (helper *NiuNiuHelper) LessMaxCard(first, second []int) bool {
	// sort
	max1 := helper.GetMaxCard(first)
	max2 := helper.GetMaxCard(second)
	// log.Debug(max1, max2)
	return helper.LessCard(max1, max2)
}

// 作弊，生成指定的牌型
func (helper *NiuNiuHelper) Cheat(typ int, table []int) []int {
	cards := make([]int, 0, 5)
	validCards := make([]int, 0, 64)

	switch typ {
	case 11: // 五小牛
		// TODO
		// 4、1、1、1、1
		// 3、1、1、1、1
		// 2、1、1、1、1
		// 3、3、1、1、1
		// 3、2、1、1、1
		// 2、2、1、1、1
		// 2、2、2、1、1
		// 2、2、2、2、1
		panic("unkown type")
	case 12: // 炸弹牛
		for value := 0x2; value <= 0xe; value++ {
			if table[value] > 0 && table[value|0x10] > 0 && table[value|0x20] > 0 && table[value|0x30] > 0 {
				validCards = append(validCards, value)
			}
		}
		if len(validCards) == 0 {
			return nil
		}
		c := validCards[rand.Intn(len(validCards))]

		validCards = validCards[:0]
		cards = append(cards, c, c|0x10, c|0x20, c|0x30)
		for _, c := range cardutils.GetAllCards() {
			for i := 0; i < table[c] && c&0x0f != c; i++ {
				validCards = append(validCards, c)
			}
		}
		if len(validCards) == 0 {
			return nil
		}
		c = validCards[rand.Intn(len(validCards))]
		cards = append(cards, c)
		return cards
	case 13, 14: // 五花牛、四花牛
		for value := 0xb; value <= 0xe; value++ {
			for color := 0; color < 4; color++ {
				c := color<<4 | value
				for i := 0; i < table[c]; i++ {
					validCards = append(validCards, c)
				}
			}
		}
		n := 4
		if typ == 12 {
			n = 5
		}
		randutils.Shuffle(validCards)
		if len(validCards) < n {
			return nil
		}
		cards = append(cards, validCards[:n]...)

		validCards = validCards[:0]
		for _, c := range []int{0x0a, 0x1a, 0x2a, 0x3a} {
			for i := 0; i < table[c]; i++ {
				validCards = append(validCards, c)
			}
		}
		randutils.Shuffle(validCards)
		if len(cards)+len(validCards) < 5 {
			return nil
		}
		cards = append(cards, validCards[:5-len(validCards)]...)
		return cards
	}

	for _, c := range cardutils.GetAllCards() {
		for i := 0; i < table[c]; i++ {
			validCards = append(validCards, c)
		}
	}

	resultSet := make([]int, 0, 1024)
	for i := 0; i < len(validCards); i++ {
		for j := i + 1; j < len(validCards); j++ {
			for k := j + 1; k < len(validCards); k++ {
				c1, c2, c3 := validCards[i], validCards[j], validCards[k]
				sum := helper.GetCardWeight(c1) + helper.GetCardWeight(c2) + helper.GetCardWeight(c3)
				if (typ != 0 && sum%10 == 0) || (typ == 0 && sum%10 != 0) {
					resultSet = append(resultSet, (c1<<16)|(c2<<8)|c3)
				}
			}
		}
	}
	if len(resultSet) == 0 {
		return nil
	}

	cards = cards[:0]
	validCards = validCards[:0]
	n := resultSet[rand.Intn(len(resultSet))]
	cards = append(cards, n>>16, n>>8&0xff, n&0xff)
	for _, c := range cards {
		table[c]--
	}
	for _, c := range cardutils.GetAllCards() {
		for i := 0; i < table[c]; i++ {
			validCards = append(validCards, c)
		}
	}
	for _, c := range cards {
		table[c]++
	}

	resultSet = resultSet[:0]
	for i := 0; i < len(validCards); i++ {
		for j := i + 1; j < len(validCards); j++ {
			cards = append(cards, validCards[i], validCards[j])
			t, _ := helper.Weight(cards)
			if t == typ {
				resultSet = append(resultSet, validCards[i]<<8|validCards[j])
			}
			cards = cards[:3]
		}
	}
	if len(resultSet) == 0 {
		return nil
	}
	n = resultSet[rand.Intn(len(resultSet))]
	cards = append(cards, n>>8, n&0xff)
	return cards
}
