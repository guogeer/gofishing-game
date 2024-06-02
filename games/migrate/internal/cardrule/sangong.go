// 湖南、广东地区的三公玩法
package cardrule

// "github.com/guogeer/husky/log"

// 0~9 0~9点数
// 333 3个三
// 100 三公

type SangongHelper struct {
}

func NewSangongHelper() *SangongHelper {
	return &SangongHelper{}
}

func (helper *SangongHelper) Color(c int) int {
	return c >> 4
}

// return: 类型、JQK数量
func (helper *SangongHelper) GetType(cards []int) (int, int) {
	var sum, jqk, card3 int
	for _, c := range cards {
		points := c & 0x0f
		switch points {
		case 11, 12, 13: // J、Q、K
			points = 10
		case 14: // A
			points = 1
		}
		sum += points

		if c&0x0f == 3 {
			card3++
		}

		switch c & 0x0f {
		case 11, 12, 13:
			jqk++
		}
	}
	if card3 == 3 {
		return 333, jqk
	}
	if jqk == 3 {
		return 100, jqk
	}
	return sum % 10, jqk
}

// K>Q>J...>2>A
// 黑桃>红桃>梅花>方块
func (helper *SangongHelper) LessCard(c1, c2 int) bool {
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
func (helper *SangongHelper) GetMaxCard(cards []int) int {
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

func (helper *SangongHelper) Less(first, second []int) bool {
	typ1, jqk1 := helper.GetType(first)
	typ2, jqk2 := helper.GetType(second)
	// log.Debug(typ1, jqk1)
	// log.Debug(typ2, jqk2)
	if typ1 != typ2 {
		return typ1 < typ2
	}
	if jqk1 != jqk2 {
		return jqk1 < jqk2
	}
	return helper.LessMaxCard(first, second)
}

// 比较最大的牌
func (helper *SangongHelper) LessMaxCard(first, second []int) bool {
	// sort
	max1 := helper.GetMaxCard(first)
	max2 := helper.GetMaxCard(second)
	// log.Debug(max1, max2)
	return helper.LessCard(max1, max2)
}
