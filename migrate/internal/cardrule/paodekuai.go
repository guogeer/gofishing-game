package cardrule

//	"github.com/guogeer/husky/log"

const (
	PaodekuaiNone = iota
	PaodekuaiDanzhang
	PaodekuaiDuizi
	PaodekuaiSandai0
	PaodekuaiSandai1
	PaodekuaiSandai2
	PaodekuaiSandaidui
	PaodekuaiShunzi
	PaodekuaiLiandui
	PaodekuaiFeiji
	PaodekuaiFeijidaidui
	PaodekuaiZhadan
	PaodekuaiSidaisan
)

var (
	paodekuaiValues = []int{
		0, 0, // 0~1
		16,                      // 2
		3, 4, 5, 6, 7, 8, 9, 10, // 3~10
		11, 12, 13, 14, // J~A
	}
)

type PaodekuaiHelper struct {
	Sidaisan  bool
	Sandaidui bool // 三带对
}

func NewPaodekuaiHelper() *PaodekuaiHelper {
	return &PaodekuaiHelper{}
}

// 类型
func (helper *PaodekuaiHelper) Value(c int) int {
	return paodekuaiValues[c&0x0f]
}

// 类型
func (helper *PaodekuaiHelper) GetType(cards []int) (int, int, int) {
	var pair, counter int
	var values [32]int
	for _, c := range cards {
		v := helper.Value(c)
		values[v]++
	}
	// TODO 炸弹不能拆
	for _, n := range values {
		if n == 2 {
			pair++
		}
	}

	for v, n := range values {
		// 炸弹
		if n == 4 && len(cards) == 4 {
			return PaodekuaiZhadan, v, 0
		}
		if helper.Sidaisan {
			// 四带三
			if n == 4 && len(cards) > 4 {
				return PaodekuaiSidaisan, v, 0
			}
		}
		counter = 0
		if helper.Sandaidui {
			// 飞机
			for k := v; values[k] == 3; k++ {
				counter++
			}
			// 飞机带对子
			if counter > 1 && len(cards) == counter*5 && pair == counter {
				return PaodekuaiFeiji, v, counter
			}
			// 飞机带一个
			if counter > 1 && len(cards)%4 == 0 && counter*4 >= len(cards) {
				return PaodekuaiFeiji, v, len(cards) / 4
			}
			// 飞机不带
			if counter > 1 && len(cards) == counter*3 {
				return PaodekuaiFeiji, v, counter
			}
		} else {
			for k := v; values[k] == 3 && counter*5 < len(cards); k++ {
				counter++
			}
			if counter > 1 && len(cards) <= counter*5 {
				return PaodekuaiFeiji, v, counter
			}
		}
		if helper.Sandaidui {
			// 三带对
			if n == 3 && len(cards) == 5 && pair == 1 {
				return PaodekuaiSandaidui, v, 0
			}
		} else {
			// 三带二
			if n == 3 && len(cards) == 5 {
				return PaodekuaiSandai2, v, 0
			}
		}
		// 三带一
		if n == 3 && len(cards) == 4 {
			return PaodekuaiSandai1, v, 0
		}
		// 三带零
		if n == 3 && len(cards) == 3 {
			return PaodekuaiSandai0, v, 0
		}
		// 连对
		counter = 0
		for k := v; values[k] == 2; k++ {
			counter++
		}
		if counter > 1 && 2*counter == len(cards) {
			return PaodekuaiLiandui, v, counter
		}
		// 顺子，不包括4个花色的2
		counter = 0
		for k := v; values[k] == 1; k++ {
			counter++
		}
		if counter >= 5 && counter == len(cards) {
			return PaodekuaiShunzi, v, counter
		}
		if n == 1 && len(cards) == 1 {
			return PaodekuaiDanzhang, v, 0
		}
		if n == 2 && len(cards) == 2 {
			return PaodekuaiDuizi, v, 0
		}
	}
	return PaodekuaiNone, 0, 0
}

func (helper *PaodekuaiHelper) Less(first, second []int) bool {
	typ1, v1, n1 := helper.GetType(first)
	typ2, v2, n2 := helper.GetType(second)

	// log.Debug(typ1, v1, n1)
	// log.Debug(typ2, v2, n2)
	if !helper.Sandaidui {
		if typ1 == PaodekuaiSandai0 || typ1 == PaodekuaiSandai1 || typ1 == PaodekuaiSandai2 {
			typ1 = PaodekuaiSandai2
		}
		if typ2 == PaodekuaiSandai0 || typ2 == PaodekuaiSandai1 || typ2 == PaodekuaiSandai2 {
			typ2 = PaodekuaiSandai2
		}
	}

	if typ1 == typ2 {
		return v1 < v2 && n1 == n2
	}
	return typ2 == PaodekuaiZhadan
}

func (helper *PaodekuaiHelper) Match(cards, match []int) []int {
	var values [32]int
	var ans = make([]int, 0, 8)
	for _, c := range cards {
		v := helper.Value(c)
		values[v]++
	}

	typ, val, length := helper.GetType(match)
	switch typ {
	case PaodekuaiDanzhang:
		for _, c := range cards {
			if helper.Value(c) > val {
				return []int{c}
			}
		}
	case PaodekuaiDuizi:
		var pv int
		for v, n := range values {
			if n > 1 && v > val {
				pv = v
				break
			}
		}
		for _, c := range cards {
			if pv == helper.Value(c) {
				ans = append(ans, c)
			}
			if len(ans) == len(match) {
				return ans
			}
		}
	case PaodekuaiSandai0, PaodekuaiSandai1, PaodekuaiSandai2:
		var pv int
		for v, n := range values {
			if n > 2 && v > val {
				pv = v
				break
			}
		}
		for _, c := range cards {
			if pv == helper.Value(c) {
				ans = append(ans, c)
			}
			if len(ans) == 3 {
				// 额外的牌
				for v := range values {
					for _, single := range cards {
						if len(ans) >= len(match) {
							break
						}
						if pv != v && v == helper.Value(single) {
							ans = append(ans, single)
						}
					}
					if len(ans) == len(match) || len(ans) == len(cards) {
						return ans
					}
				}
			}
		}
	case PaodekuaiSandaidui:
		var mainValue int
		for v, n := range values {
			if n > 2 && v > val {
				mainValue = v
				break
			}
		}
		for _, c := range cards {
			if mainValue == helper.Value(c) {
				ans = append(ans, c)
			}
			if len(ans) == 3 {
				break
			}
		}

		if len(ans) == 3 {
			extra := len(match) - 3
			for v, n := range values {
				if n < extra {
					continue
				}
				if mainValue == v {
					continue
				}

				counter := 0
				for _, single := range cards {
					if counter >= extra {
						break
					}
					if v == helper.Value(single) {
						counter++
						ans = append(ans, single)
					}
				}
				if len(ans) == len(match) {
					return ans
				}
			}

		}
	case PaodekuaiShunzi, PaodekuaiLiandui:
		var amount = 1
		if typ == PaodekuaiLiandui {
			amount = 2
		}

		var pv int
		for start := val + 1; start < len(values); start++ {
			seq := start
			for values[seq] >= amount {
				seq++
			}
			if start+length == seq {
				pv = start
				break
			}
		}
		for start := pv; start < pv+length; start++ {
			var counter int
			for _, single := range cards {
				if start == helper.Value(single) {
					ans = append(ans, single)
					counter++
				}
				if len(ans) == len(match) {
					return ans
				}
				if counter >= amount {
					break
				}
			}
		}
	case PaodekuaiFeiji:
		var pv int
		for start := val + 1; start < len(values); start++ {
			seq := start
			for values[seq] > 2 {
				seq++
			}
			if start+length == seq {
				pv = start
				break
			}
		}

		for start := pv; start < pv+length; start++ {
			var counter int
			for _, single := range cards {
				if start == helper.Value(single) {
					ans = append(ans, single)
					counter++
				}
				if counter >= 3 {
					break
				}
			}
		}
		if len(ans) == 3*length {
			// 额外的牌
			for v := range values {
				for _, single := range cards {
					if len(ans) >= len(match) {
						break
					}
					if !(v >= pv && v < pv+length) && v == helper.Value(single) {
						ans = append(ans, single)
					}
				}
				if len(ans) == len(match) || len(ans) == len(cards) {
					return ans
				}
			}
		}
	case PaodekuaiSidaisan:
		var pv int
		for v, n := range values {
			if n > 3 && v > val {
				pv = v
				break
			}
		}
		for _, c := range cards {
			if pv == helper.Value(c) {
				ans = append(ans, c)
			}
			if len(ans) == 4 {
				// 额外的牌
				for v := range values {
					for _, single := range cards {
						if len(ans) >= len(match) {
							break
						}
						if pv != v && v == helper.Value(single) {
							ans = append(ans, single)
						}
					}
					if len(ans) == len(match) || len(ans) == len(cards) {
						return ans
					}
				}
			}
		}
	}
	var pv int
	for v, n := range values {
		if n > 3 && ((v > val && typ == PaodekuaiZhadan) || typ != PaodekuaiZhadan) {
			pv = v
			break
		}
	}
	ans = ans[:0]
	for _, c := range cards {
		if pv == helper.Value(c) {
			ans = append(ans, c)
		}
		if len(ans) == 4 {
			return ans
		}
	}
	return nil
}

func (helper *PaodekuaiHelper) MaxCard(cards []int) int {
	max := cards[0]
	for _, c := range cards {
		if helper.Value(max) < helper.Value(c) {
			max = c
		}
	}
	return max
}
