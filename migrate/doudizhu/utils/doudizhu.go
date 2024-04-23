// 2018-11-06 Guogeer
// 重构了斗地主匹配牌型算法
// 增加斗地主AI

package utils

import (
	"fmt"
	"gofishing-game/internal/cardutils"
)

var enableDebugDoudizhu = false

const (
	DoudizhuNone = iota
	DoudizhuDanzhang
	DoudizhuDuizi
	DoudizhuSandai0
	DoudizhuSandai1
	DoudizhuSandai2 // 5
	DoudizhuSandaidui
	DoudizhuShunzi
	DoudizhuLiandui
	DoudizhuFeiji
	DoudizhuFeijidaidui // 10
	DoudizhuZhadan
	DoudizhuWangzha  // 王炸
	DoudizhuSidaier  // 四带二
	DoudizhuFeiji0   // 飞机不带
	DoudizhuSidaidui // 四带两队
	DoudizhuTypeAll
)

type doudizhuTypeAttr struct {
	pair     int
	width    int
	single   int
	score    []int
	priority []int
}

var doudizhuTypeAttrList = []*doudizhuTypeAttr{
	// DoudizhuNone
	{},
	// DoudizhuDanzhang
	{
		pair:     0,
		single:   0,
		width:    1,
		score:    []int{10, 5, -6, -6, -6, -10, -10, -10},
		priority: []int{0},
	},
	// DoudizhuDuizi
	{
		pair:     0,
		single:   0,
		width:    2,
		score:    []int{10, 5, -6, -10, -10, -10},
		priority: []int{0},
	},
	// DoudizhuSandai0
	{
		pair:     0,
		single:   0,
		width:    3,
		score:    []int{15, 10, 10, 10, 10, 10},
		priority: []int{-20},
	},
	// DoudizhuSandai1
	{
		pair:     0,
		single:   1,
		width:    3,
		score:    []int{10, 10, 10, 10, 10},
		priority: []int{-20},
	},
	// DoudizhuSandai2
	{
		pair:     0,
		single:   2,
		width:    3,
		score:    []int{10, 10, 10, 10, 10},
		priority: []int{-20},
	},
	// DoudizhuSandaidui
	{
		pair:     1,
		single:   0,
		width:    3,
		score:    []int{10, 10, 10, 10, 10, 10},
		priority: []int{-20},
	},
	// DoudizhuShunzi
	{
		pair:     0,
		single:   0,
		width:    1,
		score:    []int{10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10},
		priority: []int{-40},
	},
	// DoudizhuLiandui
	{
		pair:     0,
		single:   0,
		width:    2,
		score:    []int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		priority: []int{-30},
	},
	// DoudizhuFeiji
	{
		pair:     0,
		single:   1,
		width:    3,
		score:    []int{20, 20, 20, 20, 20, 20, 20, 20, 20, 20, 20, 20, 20},
		priority: []int{-45},
	},
	// DoudizhuFeijidaidui
	{
		pair:     1,
		single:   0,
		width:    3,
		score:    []int{20, 20, 20, 20, 20, 20, 20, 20, 20, 20, 20, 20, 20},
		priority: []int{-45},
	},
	// DoudizhuZhadan
	{
		pair:     0,
		single:   0,
		width:    4,
		score:    []int{20, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10},
		priority: []int{20},
	},
	// DoudizhuWangzha // 王炸
	{
		pair:     0,
		single:   0,
		width:    0,
		score:    []int{30},
		priority: []int{20},
	},
	// DoudizhuSidaier // 四带二
	{
		pair:     0,
		single:   2,
		width:    4,
		score:    []int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		priority: []int{20},
	},
	// DoudizhuFeiji0 // 飞机不带
	{
		pair:     0,
		single:   0,
		width:    3,
		score:    []int{20, 20, 20, 20, 20, 20, 20, 20, 20, 20, 20, 20, 20},
		priority: []int{-45},
	},
	// DoudizhuSidaidui // 四带对
	{
		pair:     2,
		single:   0,
		width:    4,
		score:    []int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		priority: []int{20},
	},
	{},
}

var (
	doudizhuValues = []int{
		0, 0, // 0~1
		16,                      // 2
		3, 4, 5, 6, 7, 8, 9, 10, // 3~10
		11, 12, 13, 14, // J~A
	}
)

var doudizhuTypeList []int

func init() {
	for i := 0; i < DoudizhuTypeAll; i++ {
		doudizhuTypeList = append(doudizhuTypeList, i)
	}
}

type DoudizhuHelper struct{}

func NewDoudizhuHelper() *DoudizhuHelper {
	return &DoudizhuHelper{}
}

// 类型
func (helper *DoudizhuHelper) Value(c int) int {
	// 小王
	if c == 0xf0 {
		return 20
	}
	// 大王
	if c == 0xf1 {
		return 21
	}
	return doudizhuValues[c&0x0f]
}

// 类型、值、长度
func (helper *DoudizhuHelper) GetType(cards []int) (int, int, int) {
	if len(cards) == 0 {
		return DoudizhuNone, 0, 0
	}

	var values [32]int
	var king, pair, counter int
	for _, c := range cards {
		v := helper.Value(c)
		values[v]++
		if c > 0 && cardutils.IsCardGhost(c) {
			king++
		}
	}
	// 炸弹
	if len(cards) == 2 && king == 2 {
		return DoudizhuWangzha, 0, 1
	}

	// 炸弹不能拆
	for _, n := range values {
		if n == 2 {
			pair++
		}
	}

	for _, c := range cards {
		v := helper.Value(c)
		n := values[v]
		// 炸弹
		if n == 4 && len(cards) == 4 {
			return DoudizhuZhadan, v, 1
		}
		// 四带二
		if n == 4 && len(cards) == 6 {
			return DoudizhuSidaier, v, 1
		}
		// 四带两对
		if n == 4 && len(cards) == 8 && pair == 2 {
			return DoudizhuSidaidui, v, 1
		}
		// 飞机
		counter = 0
		for k := v; values[k] == 3; k++ {
			counter++
		}
		// 飞机带对子
		if counter > 1 && len(cards) == counter*5 && pair == counter {
			return DoudizhuFeijidaidui, v, counter
		}
		// 飞机不带
		if counter > 1 && len(cards) == counter*3 {
			return DoudizhuFeiji0, v, counter
		}
		// 飞机
		if counter > 1 && len(cards)%4 == 0 && counter*4 >= len(cards) {
			return DoudizhuFeiji, v, len(cards) / 4
		}
		// 三带对
		if n == 3 && len(cards) == 5 && pair == 1 {
			return DoudizhuSandaidui, v, 1
		}
		// 三带一
		if n == 3 && len(cards) == 4 {
			return DoudizhuSandai1, v, 1
		}
		// 三带零
		if n == 3 && len(cards) == 3 {
			return DoudizhuSandai0, v, 1
		}
		// 连对
		counter = 0
		for k := v; values[k] == 2; k++ {
			counter++
		}
		if counter > 2 && 2*counter == len(cards) {
			return DoudizhuLiandui, v, counter
		}
		// 顺子，不包括4个花色的2
		counter = 0
		for k := v; values[k] == 1; k++ {
			counter++
		}
		if counter >= 5 && counter == len(cards) {
			return DoudizhuShunzi, v, counter
		}
		if n == 1 && len(cards) == 1 {
			return DoudizhuDanzhang, v, 1
		}
		if n == 2 && len(cards) == 2 {
			return DoudizhuDuizi, v, 1
		}
	}
	return DoudizhuNone, 0, 0
}

func (helper *DoudizhuHelper) GetSortedCards(table []int) []int {
	cards := make([]int, 0, 16)
	for v := 0; v <= helper.Value(0xf1); v++ {
		for _, c := range cardutils.GetAllCards() {
			if helper.Value(c) == v {
				for k := 0; k < table[c]; k++ {
					cards = append(cards, c)
				}
			}
		}
	}
	return cards
}

func (helper *DoudizhuHelper) Less(first, second []int) bool {
	typ1, v1, n1 := helper.GetType(first)
	typ2, v2, n2 := helper.GetType(second)

	// log.Debug(typ1, v1, n1)
	// log.Debug(typ2, v2, n2)
	if typ1 == typ2 && len(first) == len(second) {
		return v1 < v2 && n1 == n2
	}
	// 王炸特殊考虑
	if typ1 == DoudizhuWangzha {
		return false
	}
	if typ2 == DoudizhuWangzha {
		return true
	}

	return typ2 == DoudizhuZhadan
}

func (helper *DoudizhuHelper) Match(cards, match []int) []int {
	var values [32]int
	for _, c := range cards {
		v := helper.Value(c)
		values[v]++
	}

	typ, val, length := helper.GetType(match)
	attr := doudizhuTypeAttrList[typ]
	width, pair, single := attr.width, attr.pair, attr.single
	pair, single = pair*length, single*length
	if typ == DoudizhuWangzha {
		return nil
	}

	for start := val + 1; start < len(values); start++ {
		seq := start
		for width > 0 && values[seq] >= width {
			seq++
		}
		if start+length > seq {
			continue
		}
		ans := make([]int, 0, 8)
		used := make(map[int]bool)
		for v := start; v < start+length; v++ {
			var counter int
			for _, c := range cards {
				if v == helper.Value(c) {
					ans = append(ans, c)
					used[v] = true
					counter++
				}
				if counter >= width {
					break
				}
			}
		}
		var pair2 int
		for v, n := range values {
			if n > 1 && !used[v] && pair2 < pair {
				var counter int
				for _, c := range cards {
					if v == helper.Value(c) {
						ans = append(ans, c)
						used[v] = true
						counter++
					}
					if counter >= 2 {
						pair2++
						break
					}
				}
			}
		}
		var single2 int
		for v, n := range values {
			if n > 0 && !used[v] && single2 < single && n < 3 {
				for _, c := range cards {
					if v == helper.Value(c) {
						ans = append(ans, c)
						used[v] = true
						single2++
						break
					}
				}
			}
		}
		if pair2 == pair && single2 == single {
			return ans
		}
	}

	// 炸弹
	for _, c := range cards {
		v := helper.Value(c)
		if values[v] > 3 && (typ != DoudizhuZhadan || val < v) {
			c = c & 0x0f
			return []int{0x00 | c, 0x10 | c, 0x20 | c, 0x30 | c}
		}
	}
	// 王炸
	if values[helper.Value(0xf0)] > 0 && values[helper.Value(0xf1)] > 0 {
		return []int{0xf0, 0xf1}
	}
	return nil
}

type DoudizhuPart struct {
	Cards            []int
	typ, val, length int
}

type DoudizhuResult struct {
	Parts         []*DoudizhuPart
	danpai, duizi []int
	numType       [DoudizhuTypeAll]int // 长度为1时，类型的数量
}

var (
	ddzWidthOfValue  = []int{1, 1, 1, 2, 3}                                                       // [3]=2 至少拆出一个2张
	ddzLengthOfValue = []int{0, 5, 3, 2, 0}                                                       // [3]=2 飞机长度至少2
	ddzTypeOfLen1    = []int{0, DoudizhuDanzhang, DoudizhuDuizi, DoudizhuSandai0, DoudizhuZhadan} // [3] 长度为1，宽度为3时牌型
	ddzTypeOfLen2    = []int{0, DoudizhuShunzi, DoudizhuLiandui, DoudizhuFeiji0}
)

// 拆牌
func (helper *DoudizhuHelper) Split(cards []int) []*DoudizhuResult {
	var values [32]int
	for _, c := range cards {
		values[helper.Value(c)]++
	}
	rs := make([]*DoudizhuResult, 0, 16)
	parts := make([]*DoudizhuPart, 0, 16)
	king1, king2 := helper.Value(0xf0), helper.Value(0xf1)

	var dfs func(int)
	dfs = func(v int) {
		if false && len(parts) > 9 {
			return
		}
		if v > king2 {
			res := &DoudizhuResult{}
			for _, part := range parts {
				part1 := &DoudizhuPart{
					typ:    part.typ,
					val:    part.val,
					length: part.length,
				}
				part1.Cards = append(part1.Cards, part.Cards...)
				res.Parts = append(res.Parts, part1)
			}
			rs = append(rs, res)
			return
		}
		if values[v] == 0 {
			dfs(v + 1)
		}
		// 大小王特殊处理
		if v == king1 && values[king1] > 0 && values[king2] > 0 {
			part := &DoudizhuPart{
				Cards: []int{king1, king2},
				typ:   DoudizhuWangzha,
			}
			values[king1]--
			values[king2]--
			parts = append(parts, part)
			dfs(v)
			parts = parts[:len(parts)-1]
			values[king1]++
			values[king2]++
		}
		// fmt.Println("dfs", v, values[v])
		minWidth := ddzWidthOfValue[values[v]]
		if values[v] > 0 && values[v] <= values[v+1] {
			minWidth = values[v]
		}
		for width := values[v]; width >= minWidth; width-- {
			part := &DoudizhuPart{val: v}
			length := ddzLengthOfValue[width]
			for i := 0; i < width; i++ {
				part.Cards = append(part.Cards, v)
			}
			values[v] -= width
			part.typ, part.length = ddzTypeOfLen1[width], 1
			parts = append(parts, part)
			dfs(v)
			parts = parts[:len(parts)-1]
			values[v] += width
			part.Cards = part.Cards[:len(part.Cards)-width]

			end := v
			for values[end+1] >= width {
				end++
			}
			if v+length-1 > end {
				continue
			}
			end = v + length - 1
			for start := v; start < end; start++ {
				values[start] -= width
				for i := 0; i < width; i++ {
					part.Cards = append(part.Cards, start)
				}
			}
			seq := end
			for values[seq] >= width {
				for i := 0; i < width; i++ {
					part.Cards = append(part.Cards, seq)
				}
				values[seq] -= width
				part.typ, part.length = ddzTypeOfLen2[width], seq+1-v
				parts = append(parts, part)
				dfs(v)
				parts = parts[:len(parts)-1]
				seq += 1
			}
			for start := v; start < seq; start++ {
				values[start] += width
			}
		}
	}
	dfs(0)
	for _, res := range rs {
		for _, part := range res.Parts {
			for i, v := range part.Cards {
				part.Cards[i] = 0xff00 | v // 标记是否已替换
			}
		}
	}
	for _, res := range rs {
		group := make([]int, 0, 4)
		for v := range values {
			group = group[:0]
			for _, c := range cards {
				if v == helper.Value(c) {
					group = append(group, c)
				}
			}
			k := 0
			for _, part := range res.Parts {
				for i := range part.Cards {
					l := part.Cards[i] & 0x00ff
					h := part.Cards[i] & 0xff00
					if h > 0 && l == v {
						part.Cards[i] = group[k]
						k += 1
					}
				}
			}
		}
	}
	for _, res := range rs {
		for i, part := range res.Parts {
			cards := part.Cards
			if len(cards) == 1 {
				res.danpai = append(res.danpai, i)
			}
			if len(cards) == 2 &&
				helper.Value(cards[0]) == helper.Value(cards[1]) {
				res.duizi = append(res.duizi, i)
			}
			res.numType[part.typ]++
		}
	}
	return rs
}

type DoudizhuUser struct {
	Cards   []int // 玩家手牌，注意：Cards[c] = n，表示牌c有n张
	Step    []int // 本轮出牌
	CardNum int
}

// 斗地主AI
type DoudizhuAI struct {
	MySeat    int             // 座位
	Dizhu     int             // 地主座位
	UsedCards []int           // 已打出去的牌
	Users     []*DoudizhuUser // 当前的玩家
	Helper    *DoudizhuHelper
	isInit    bool // 初始化
	ranks     [DoudizhuTypeAll][32]int
}

func (ai *DoudizhuAI) String() string {
	s := fmt.Sprintf("MySeat: %d, Dizhu: %d", ai.MySeat, ai.Dizhu)
	s += fmt.Sprintf(",UsedCards: %s", cardutils.Format(ai.UsedCards))

	users := ""
	for _, user := range ai.Users {
		var cards []int
		for c, n := range user.Cards {
			for k := 0; k < n; k++ {
				cards = append(cards, c)
			}
		}
		if users != "" {
			users += ","
		}
		users += fmt.Sprintf("&DoudizhuUser{Cards: %s, Step: %s, CardNum: %d}", cardutils.Format(cards), cardutils.Format(user.Step), user.CardNum)
	}
	s += fmt.Sprintf(",Users: []*DoudizhuUser{%s}", users)
	return s
}

func (ai *DoudizhuAI) init() {
	if ai.isInit {
		return
	}
	ai.isInit = true
	/* for _, user := range ai.Users {
		for c := range user.Cards {
			user.CardNum += user.Cards[c]
		}
	} */

	var unused, own [32]int
	var help = ai.Helper
	var sys = cardutils.GetCardSystem()
	// fmt.Println(ai.MySeat)
	for _, c := range sys.GetAllCards() {
		own[help.Value(c)] += ai.Users[ai.MySeat].Cards[c]
	}
	for c, n := range sys.Table() {
		if n > 0 {
			unused[help.Value(c)] += n
		}
	}
	for _, c := range ai.UsedCards {
		unused[help.Value(c)]--
	}
	for i := range ai.ranks {
		for j := range ai.ranks[i] {
			ai.ranks[i][j] = -1
			ai.ranks[DoudizhuWangzha][j] = 0
		}
	}
	for v1 := len(unused) - 1; v1 > 0; v1-- {
		if unused[v1] == 0 {
			continue
		}
		for i := range ai.ranks {
			ai.ranks[i][v1] = 0
		}

		ai.ranks[DoudizhuDanzhang][v1] = 1
		ai.ranks[DoudizhuDuizi][v1] = 1
		ai.ranks[DoudizhuSandai0][v1] = 1
		// 炸弹默认最大
		ai.ranks[DoudizhuZhadan][v1] = 0
		if unused[v1] <= own[v1] {
			ai.ranks[DoudizhuDanzhang][v1] = 0
		}
		for v2 := range unused {
			if unused[v2] > 0 && v1 < v2 {
				ai.ranks[DoudizhuDanzhang][v1] += unused[v2]
			}
			if own[v1] == 4 && unused[v2] == 4 && v1 < v2 {
				ai.ranks[DoudizhuZhadan][v1]++
			}
			if own[v1] > 1 && unused[v2] > 1 && v1 < v2 {
				ai.ranks[DoudizhuDuizi][v1] += unused[v2] / 2
			}
			if own[v1] > 2 && unused[v2] > 2 && v1 < v2 {
				ai.ranks[DoudizhuSandai0][v1]++
			}
			if false && own[v1] > 3 && unused[v2] > 3 && v1 < v2 {
				ai.ranks[DoudizhuZhadan][v1]++
			}
		}
		n := ai.ranks[DoudizhuDuizi][v1]
		if unused[v1] < own[v1]+2 && own[v1] > 1 && n == 1 {
			ai.ranks[DoudizhuDuizi][v1] = 0
		}
	}
	for i := range ai.ranks {
		for j := range ai.ranks[i] {
			score := doudizhuTypeAttrList[i].score
			if ai.ranks[i][j] >= len(score) {
				ai.ranks[i][j] = len(score) - 1
			}
		}
	}
}

// 给予当前手牌打分
func (ai *DoudizhuAI) Score(res *DoudizhuResult) int {
	ai.init()

	sum := 0
	for _, part := range res.Parts {
		rank := ai.ranks[part.typ][part.val]
		// fmt.Println(rank, part.typ, part.val)
		s := doudizhuTypeAttrList[part.typ].score[rank]
		sum += s
	}
	return sum
}

// TODO 先考虑两个玩家的情况
func (ai *DoudizhuAI) Turn() []int {
	ai.init()

	var lastStep, friendStep []int
	for i := 1; i < len(ai.Users); i++ {
		seat := (ai.MySeat + len(ai.Users) - i) % len(ai.Users)
		step := ai.Users[seat].Step
		if len(lastStep) == 0 {
			lastStep = step
		}
		if ai.MySeat != ai.Dizhu && seat != ai.Dizhu && len(friendStep) == 0 {
			friendStep = step
		}
	}

	maxScore := int(-1e6)
	curScore, priority := maxScore, maxScore
	curStep := make([]int, 0, 4)

	otherCards := make([]int, 0, 8)
	sortedCards := make([]int, 0, 8)
	validTable := make(map[int]int)
	for c, n := range ai.Users[ai.MySeat].Cards {
		for k := 0; k < n; k++ {
			sortedCards = append(sortedCards, c)
			validTable[c]++
		}
	}

	// fmt.Println("turn", Format(sortedCards))
	dizhu := ai.Users[ai.Dizhu]
	rs := ai.Helper.Split(sortedCards)
	typ, val, length := ai.Helper.GetType(lastStep)
	ftype, fval, _ := ai.Helper.GetType(friendStep)

	// fmt.Println("*******", Format(sortedCards), rs)
	for _, res := range rs {
		step := make([]int, 0, 4)     // 可选的出牌
		goodStep := make([]int, 0, 4) // 和对手手牌牌数不一样多的出牌

		minCost := 100                           // 出牌的代价最小
		score := ai.Score(res)                   // 拆牌的价值
		usedParts := make([]*DoudizhuPart, 0, 4) // 使用中的特征牌
		// fmt.Println("=================", Format(lastStep))
		allowTypes := doudizhuTypeList
		if len(lastStep) > 0 {
			allowTypes = []int{DoudizhuZhadan, DoudizhuWangzha, typ}
			if typ == DoudizhuWangzha {
				allowTypes = allowTypes[1:]
			}
		}

		for _, t := range allowTypes {
			for _, part := range res.Parts {
				if len(lastStep) == 0 {
					val, length = -1, part.length
				}

				if false && enableDebugDoudizhu {
					fmt.Println("turn part", cardutils.Format(part.Cards))
				}
				wingsN, wings := 0, []int{}
				attr := doudizhuTypeAttrList[t]
				if attr.pair > 0 {
					wingsN, wings = length*attr.pair, res.duizi
				}
				if attr.single > 0 {
					wingsN, wings = length*attr.single, res.danpai
				}
				// TODO 单张、对子
				// 该类型的牌超过3搭，就尽量不出最大的
				size := part.length * attr.width
				if size < 3 && ai.ranks[part.typ][part.val] == 0 &&
					res.numType[part.typ] > 2 {
					continue
				}

				rawType := t
				fakeVal, fakeLen := val, length
				step, usedParts = step[:0], usedParts[:0]
				switch t {
				case DoudizhuSandai1, DoudizhuSandaidui:
					rawType = DoudizhuSandai0
				case DoudizhuFeiji0, DoudizhuFeiji, DoudizhuFeijidaidui:
					rawType = DoudizhuFeiji0
				case DoudizhuSidaier:
					rawType = DoudizhuZhadan
				case DoudizhuZhadan, DoudizhuWangzha:
					if typ != t {
						fakeVal, fakeLen, wingsN = -1, part.length, 0
					}
				}
				if part.typ != rawType {
					continue
				}
				// fmt.Println("!!!", t, typ, val, length, part.val, part.length, wingsN, fakeVal, rawType)
				if fakeVal < part.val && fakeLen == part.length && wingsN <= len(wings) {
					end := wingsN
					step = append(step[:0], part.Cards...)
					for _, ref := range wings[:end] {
						step = append(step, res.Parts[ref].Cards...)
						usedParts = append(usedParts, res.Parts[ref])
					}
				}

				if len(step) == 0 {
					continue
				}

				usedParts = append(usedParts, part)

				var cost int
				var isMax = true
				// 尽量不要出和对手手牌相同数量的牌
				var isSimilar = false // 和地主手牌一样多
				// fmt.Println("++++++++", dizhu.CardNum, Format(step))
				partRanks := ai.ranks[part.typ]
				for v := part.val + 1; isMax && v < len(partRanks); v++ {
					if partRanks[part.val] > 0 && partRanks[v] >= 0 {
						isMax = false
					}
				}
				if ai.MySeat != ai.Dizhu && dizhu.CardNum == len(step) {
					isSimilar = true
				}
				if ai.MySeat == ai.Dizhu {
					for seat, user := range ai.Users {
						if ai.MySeat != seat && len(step) == user.CardNum {
							isSimilar = true
							break
						}
					}
				}
				if isSimilar && !isMax {
					cost += 30
				}

				otherTable := make(map[int]int)
				for _, c := range sortedCards {
					otherTable[c]++
				}
				for _, c := range step {
					otherTable[c]--
				}
				otherCards = otherCards[:0]
				for _, c := range sortedCards {
					if otherTable[c] < 0 {
						panic("system error")
					}
					for k := 0; k < otherTable[c]; k++ {
						otherCards = append(otherCards, c)
					}
				}
				otherType, _, _ := ai.Helper.GetType(otherCards)
				if isMax && otherType > 0 {
					cost -= 200
				}
				if isMax && score >= 0 {
					cost -= 21
				}
				for _, usedPart := range usedParts {
					rank := ai.ranks[usedPart.typ][usedPart.val]
					partAttr := doudizhuTypeAttrList[usedPart.typ]
					cost += partAttr.score[rank]
					cost += partAttr.priority[0]
				}
				// 一次可以出完，修正为好牌
				// 排除四带二
				if len(step) == ai.Users[ai.MySeat].CardNum && (part.typ != DoudizhuZhadan || len(usedParts) == 1) {
					cost -= 1000
				}
				if enableDebugDoudizhu {
					fmt.Printf("test part step:%s,minCost:%d,score:%d,cost:%d,similar:%v,max:%v\n", cardutils.Format(step), minCost, score, cost, isSimilar, isMax)
				}

				if part.typ == DoudizhuZhadan && cost >= -100 {
					continue
				}
				if part.typ == DoudizhuWangzha && cost >= -100 {
					continue
				}
				// 优先代价低
				if minCost > cost || (minCost == cost && isSimilar) {
					minCost = cost
					goodStep = append(goodStep[:0], step...)
				}

				// 某种牌型数量不小于3个时，优先出最小
				if res.numType[part.typ] > 2 && !isSimilar {
					break
				}
			}
		}

		step = append(step[:0], goodStep...)
		// fmt.Println("friend", ftype, fval, score, step)
		if maxScore < score {
			maxScore = score
		}

		// 一次可以出完，修正为好牌
		if minCost < -100 {
			score += 1000
		}
		// 队友出牌
		if len(friendStep) > 0 {
			dizhu := ai.Users[ai.Dizhu]
			rank := ai.ranks[ftype][fval]
			scorelist := doudizhuTypeAttrList[ftype].score
			if enableDebugDoudizhu {
				fmt.Printf("friend step:%s,rank:%d\n", cardutils.Format(friendStep), rank)
			}
			// 出大牌或者地主要不起
			if rank < len(scorelist) || len(dizhu.Step) == 0 {
				score -= 15
			}
		}

		if enableDebugDoudizhu {
			fmt.Printf("turn result minCost:%d,score:%d,step:%s\n", minCost, score, cardutils.Format(step))
		}
		if len(step) > 0 && (curScore < score ||
			(curScore == score && priority < score-minCost)) {
			curScore, priority = score, score-minCost
			curStep = append(curStep[:0], step...)
		}
	}
	h := ai.Helper.GetSortedCards(ai.Users[ai.MySeat].Cards)
	if enableDebugDoudizhu {
		fmt.Printf("turn final result curStep:%s,curScore:%d,maxScore:%d,MyCards:%s\n", cardutils.Format(curStep), curScore, maxScore, cardutils.Format(h))
	}
	if len(lastStep) > 0 && len(curStep) > 0 && curScore+20 <= maxScore {
		curStep = curStep[:0]
	}
	return curStep
}
