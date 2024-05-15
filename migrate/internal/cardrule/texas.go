package cardrule

// 2017-8-21
// 德州扑克牌型
// 同花大顺（Royal Flush）：最高为Ace（一点）的同花顺。
// 同花顺（Straight Flush）：同一花色，顺序的牌。
// 四条（Four of a Kind）：有四张同一点数的牌。
// 葫芦（Fullhouse）：三张同一点数的牌，加一对其他点数的牌。
// 同花（Flush)：五张同一花色的牌。
// 顺子（Straight）：五张顺连的牌。
// 三条（Three of a kind）：有三张同一点数的牌。
// 两对（Two Pairs）：两张相同点数的牌，加另外两张相同点数的牌。
// 一对（One Pair）：两张相同点数的牌。
// 高牌（High Card）：不符合上面任何一种牌型的牌型，由单牌且不连续不同花的组成，以点数决定大小。

//	"github.com/guogeer/husky/log"

const (
	_                  = iota
	TexasHighCard      // 高牌
	TexasOnePair       // 对子
	TexasTwoPair       // 两队
	TexasThreeOfAKind  // 三条
	TexasStraight      // 顺子
	TexasFlush         // 同花
	TexasFullHouse     // 葫芦
	TexasFourOfAKind   // 四条
	TexasStraightFlush // 同花顺
	TexasRoyalFlush    // 皇家同花顺
	TexasTypeAll
)

type TexasHelper struct {
	size int
}

func NewTexasHelper() *TexasHelper {
	return &TexasHelper{size: 5}
}

func (helper *TexasHelper) Size() int {
	return helper.size
}

func (helper *TexasHelper) Value(c int) int {
	return c & 0x0f
}

func (helper *TexasHelper) Color(c int) int {
	return c >> 4
}

func (helper *TexasHelper) GetType(cards []int) (int, []int) {
	var color int
	var values [32]int
	for _, c := range cards {
		v := helper.Value(c)
		values[v]++

		color |= 1 << uint(helper.Color(c))
	}

	var pairs, kind, straight int
	for v, n := range values {
		if n == 2 {
			pairs++
		}
		if kind < n {
			kind = n
		}
		if straight == 0 && n > 0 {
			straight = v
		}
	}

	size := helper.Size()
	for v := straight; v < straight+size; v++ {
		if values[v] != 1 {
			straight = 0
			break
		}
	}
	if straight > 0 {
		straight += len(cards) - 1
	} else {
		// A2345
		straight = 0x5
		for _, v := range []int{0x2, 0x3, 0x4, 0x5, 0xe} {
			if values[v] != 1 {
				straight = 0
			}
		}
	}

	if straight > 0 {
		if color&(color-1) == 0 {
			if straight == 0x0e {
				return TexasRoyalFlush, []int{straight}
			}
			return TexasStraightFlush, []int{straight}
		}
		return TexasStraight, []int{straight}
	}

	rank := make([]int, 0, 8)
	for n := len(cards); n > 0; n-- {
		for v := len(values) - 1; v > 0; v-- {
			if values[v] == n {
				rank = append(rank, v)
			}
		}
	}

	if color&(color-1) == 0 {
		return TexasFlush, rank
	}
	if kind == 4 {
		return TexasFourOfAKind, rank
	}
	if kind == 3 && pairs == 1 {
		return TexasFullHouse, rank
	}
	if kind == 3 && pairs == 0 {
		return TexasThreeOfAKind, rank
	}
	if pairs == 2 {
		return TexasTwoPair, rank
	}
	if pairs == 1 {
		return TexasOnePair, rank
	}
	return TexasHighCard, rank
}

func (helper *TexasHelper) Equal(first, second []int) bool {
	return !helper.Less(first, second) &&
		!helper.Less(second, first)
}

func (helper *TexasHelper) Less(first, second []int) bool {
	typ1, rank1 := helper.GetType(first)
	typ2, rank2 := helper.GetType(second)
	// log.Debug(typ1, rank1)
	// log.Debug(typ2, rank2)
	if typ1 == typ2 {
		for i := 0; i < len(rank1); i++ {
			if rank1[i] != rank2[i] {
				return rank1[i] < rank2[i]
			}
		}
		return false
	}
	return typ1 < typ2
}

func (helper *TexasHelper) Match(cards []int) []int {
	size := helper.Size()
	ans := make([]int, size)
	sample := make([]int, size)
	for bitMap := 0; bitMap < 1<<uint(len(cards)); bitMap++ {
		var counter int
		for k := bitMap; k != 0 && counter < len(cards); k = k & (k - 1) {
			counter++
		}
		if counter == size {
			sample = sample[:0]
			for k := 0; k < len(cards); k++ {
				if bitMap&(1<<uint(k)) > 0 {
					sample = append(sample, cards[k])
				}
			}
			if ans[0] == 0 || helper.Less(ans, sample) {
				copy(ans, sample)
			}
		}
	}
	return ans
}
