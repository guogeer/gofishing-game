package cardrule

import (
	"gofishing-game/internal/cardutils"
	"math/rand"
)

const (
	_                    = iota
	ZhajinhuaSanpai      // 散牌
	ZhajinhuaDuizi       // 对子
	ZhajinhuaShunzi      // 顺子
	ZhajinhuaTonghua     // 金花
	ZhajinhuaTonghuashun // 顺金
	ZhajinhuaBaozi       // 豹子
	ZhajinhuaAAA         // AAA
	ZhajinhuaTypeAll
)

type ZhajinhuaHelper struct {
	name    string
	size    int
	options map[string]bool
}

func NewZhajinhuaHelper(name string) *ZhajinhuaHelper {
	return &ZhajinhuaHelper{
		name:    name,
		size:    3,
		options: make(map[string]bool),
	}
}

func (helper *ZhajinhuaHelper) SetOption(option string) {
	helper.options[option] = true
}

func (helper *ZhajinhuaHelper) Size() int {
	return helper.size
}

func (helper *ZhajinhuaHelper) Color(c int) int {
	return c >> 4
}

func (helper *ZhajinhuaHelper) Value(c int) int {
	return c & 0x0f
}

func (helper *ZhajinhuaHelper) GetType(cards []int) (int, []int) {
	var color int
	var values [32]int
	for _, c := range cards {
		v := helper.Value(c)
		values[v]++
		color |= 1 << uint(helper.Color(c))
	}

	var kind, straight int
	for v, n := range values {
		if kind < n {
			kind = n
		}
		if straight == 0 && n > 0 {
			straight = v
		}
	}
	for i := straight; i < straight+len(cards); i++ {
		if values[i] != 1 {
			straight = 0
		}
	}
	if straight > 0 {
		straight += len(cards) - 1
	} else {
		// A23
		straight = 0x3
		for _, v := range []int{0xe, 0x2, 0x3} {
			if values[v] != 1 {
				straight = 0
			}
		}
	}

	rank := make([]int, 0, 4)
	if straight > 0 {
		rank = append(rank, straight)
	} else {
		for n := len(cards); n > 0; n-- {
			for v := len(values) - 1; v > 0; v-- {
				if values[v] == n {
					rank = append(rank, v)
				}
			}
		}
	}

	if kind == 3 {
		// AAA
		if _, ok := helper.options["AAA"]; ok && helper.Value(cards[0]) == 0x0e {
			return ZhajinhuaAAA, rank
		}
		return ZhajinhuaBaozi, rank
	}
	if kind == 2 {
		return ZhajinhuaDuizi, rank
	}

	isSameColor := color&(color-1) == 0
	if isSameColor && straight > 0 {
		return ZhajinhuaTonghuashun, rank
	}
	if straight > 0 {
		return ZhajinhuaShunzi, rank
	}
	if isSameColor {
		return ZhajinhuaTonghua, rank
	}
	return ZhajinhuaSanpai, rank
}

func (helper *ZhajinhuaHelper) Less(first, second []int) bool {
	typ1, rank1 := helper.GetType(first)
	typ2, rank2 := helper.GetType(second)
	// log.Debug(typ1, rank1, typ2, rank2)
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

// 作弊，生成指定的牌型
func (helper *ZhajinhuaHelper) Cheat(typ int, table []int) []int {
	validCards := make([]int, 0, 64)
	resultSet := make([]int, 0, 1024)
	for _, c := range cardutils.GetCardSystem(helper.name).GetAllCards() {
		for i := 0; i < table[c]; i++ {
			validCards = append(validCards, c)
		}
	}

	for i := 0; i < len(validCards); i++ {
		for j := i + 1; j < len(validCards); j++ {
			for k := j + 1; k < len(validCards); k++ {
				c1, c2, c3 := validCards[i], validCards[j], validCards[k]
				if typ1, _ := helper.GetType([]int{c1, c2, c3}); typ == typ1 {
					h := (c1 << 16) | (c2 << 8) | c3
					resultSet = append(resultSet, h)
				}
			}
		}
	}
	if len(resultSet) == 0 {
		return nil
	}
	h := resultSet[rand.Intn(len(resultSet))]
	return []int{h >> 16, h >> 8 & 0xff, h & 0xff}
}
