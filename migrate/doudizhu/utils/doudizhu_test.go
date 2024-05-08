package utils

import (
	"gofishing-game/internal/cardutils"
	"testing"
)

func checkSameCards(from, to []int) bool {
	helper := NewDoudizhuHelper()
	if len(from) != len(to) {
		return false
	}

	tb1 := make(map[int]int)
	tb2 := make(map[int]int)
	for _, c := range from {
		v := helper.Value(c)
		tb1[v]++
	}
	for _, c := range to {
		v := helper.Value(c)
		tb2[v]++
	}
	if len(tb1) != len(tb2) {
		return false
	}
	for k, n := range tb1 {
		if tb2[k] != n {
			return false
		}
	}
	return true
}

func TestDoudizhuLess(t *testing.T) {
	helper := NewDoudizhuHelper()
	samples := [][][]int{
		{
			{0xf0, 0xf1},
			{0x03, 0x13, 0x04, 0x14, 0x24, 0x05, 0x15, 0x25, 0x06, 0x16, 0x26, 0x07, 0x17, 0x27, 0x08, 0x18, 0x28, 0x09, 0x19, 0x29},
		},
		{
			{0x03, 0x13, 0x23, 0x33, 0x04, 0x14},
			{0x03, 0x13, 0x23, 0x33, 0x04, 0x15},
		},
		{
			{0x03, 0x13, 0x23, 0x33, 0x04, 0x15},
			{0x05, 0x15, 0x25, 0x35, 0x06, 0x16, 0x07, 0x17},
		},
	}
	for _, sample := range samples {
		if helper.Less(sample[0], sample[1]) {
			t.Error("less", sample)
		}
	}
}

func TestDoudizhuMatch(t *testing.T) {
	helper := NewDoudizhuHelper()
	samples := [][][]int{
		{
			{0xf1}, // cards
			{0xf0}, // match
			{0xf1}, // answer
		},
		{
			{0xf1},
			{0x02},
			{0xf1},
		},
		{
			{0xf0, 0xf1},
			{0x02, 0x12},
			{0xf0, 0xf1},
		},
		{
			{0x06, 0x16, 0x26, 0x03, 0x1a, 0x0a},
			{0x04, 0x14, 0x24, 0x05, 0x15},
			{0x06, 0x16, 0x26, 0x1a, 0x0a},
		},
		{
			{0x06, 0x16, 0x26, 0x36, 0x03, 0x1a, 0x0a, 0x13},
			{0x04, 0x14, 0x24, 0x34, 0x05, 0x15, 0x06, 0x16},
			{0x06, 0x16, 0x26, 0x36, 0x03, 0x1a, 0x0a, 0x13},
		},
		{
			{0x02, 0x12},
			{0x0e},
			{0x02},
		},
		{
			{0x07, 0x17, 0x27, 0x37, 0x08, 0x18, 0x28, 0x38},
			{0x03, 0x13, 0x23, 0x1a, 0x04, 0x14, 0x24, 0x0a},
			{0x07, 0x17, 0x27, 0x37},
		},
		{
			{0x07, 0x17, 0x27, 0x37, 0x08, 0x18},
			{0x03, 0x13, 0x23, 0x0a, 0x1a},
			{0x07, 0x17, 0x27, 0x08, 0x18},
		},
		{
			{0x04, 0x14, 0x24, 0x34, 0x05},
			{0x03, 0x13, 0x23, 0x06, 0x16},
			{0x04, 0x14, 0x24, 0x34},
		},
		{
			{0x13, 0x04, 0x05, 0x15, 0x06, 0x07, 0x18, 0x0a},
			{0x03, 0x04, 0x05, 0x06, 0x17},
			{0x04, 0x05, 0x06, 0x07, 0x18},
		},
		{
			{0x02, 0x12, 0x22, 0x32},
			{0xf0, 0xf1},
			{},
		},
		{
			{0x07, 0x17, 0x27, 0x3a, 0x08, 0x18, 0x28, 0x3b, 0x0d},
			{0x03, 0x13, 0x23, 0x1a, 0x04, 0x14, 0x24, 0x0a},
			{0x07, 0x17, 0x27, 0x3a, 0x08, 0x18, 0x28, 0x3b},
		},
		{
			{51, 4, 5, 56, 9, 57, 10, 42, 58, 11, 43, 59, 45, 61, 14, 2, 240},
			{8, 24, 40, 23, 39, 55, 38, 54, 19, 35},
			{0x0a, 0x2a, 0x3a, 0x0b, 0x2b, 0x3b, 0x09, 0x39, 0x2d, 0x3d},
		},
	}
	for i, sample := range samples[:] {
		ans := helper.Match(sample[0], sample[1])
		if !checkSameCards(ans, sample[2]) {
			t.Error("match", i, sample, cardutils.Format(ans))
		}
	}
}

func TestDoudizhuSplit(t *testing.T) {
	helper := NewDoudizhuHelper()
	samples := [][]int{
		// {0x03, 0x04, 0xf0, 0xf1},
		// {0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		// {0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		// {0x03, 0x04, 0x14, 0x05, 0x15, 0x06, 0x16, 0x07, 0x17, 0x08},
		// {0x03, 0x04, 0x14, 0x05, 0x15, 0x06, 0x16, 0x07, 0x17, 0x08},
		{0x03, 0x13, 0x23, 0x33, 0x04, 0x14, 0x24, 0x34, 0x05, 0x15, 0x25, 0x35, 0x06, 0x16, 0x26, 0x36, 0x07, 0x17, 0x27, 0x37},
		{0x03, 0x13, 0x04, 0x14, 0x05, 0x15, 0x06, 0x16, 0x07, 0x17, 0x08, 0x18, 0x09, 0x19, 0x0a, 0x1a, 0x0b, 0x1b, 0x0c, 0x1c},
		{0x04, 0x05, 0x06, 0x07, 0x08, 0x18},
	}
	for _, sample := range samples[:0] {
		rs := helper.Split(sample)
		t.Logf("============== split %d %v", len(rs), cardutils.Format(sample))
		for i, res := range rs {
			t.Logf("result %d size %d:", i, len(res.Parts))
			for j, part := range res.Parts {
				t.Logf("part %d %v:", j, cardutils.Format(part.Cards))
			}
		}
	}
}

func TestDoudizhuScore(t *testing.T) {
	helper := NewDoudizhuHelper()
	defaultCards := []int{0xf0, 0xf1}
	for i := 0x02; i <= 0x0e; i++ {
		defaultCards = append(defaultCards, i, i&0x10, i&0x20, i&0x30)
	}
	samples := [][][]int{
		{
			{0x03, 0x04, 0x05},
			{0x04},
		},
		{
			{0x03, 0x04, 0x05, 0x06, 0x16},
			{0x06},
		},
		{
			{0x03, 0x13, 0x04, 0x05, 0x06, 0x16, 0x07, 0x08},
			{0x03, 0x13},
		},
		{
			{0x03, 0x04, 0x05, 0x06, 0x16},
			{0x06, 0x16},
		},
		{
			defaultCards,
			{0x05, 0x15, 0x25, 0x06, 0x16, 0x26, 0x07, 0x08},
		},
		// 5
		{
			defaultCards,
			{0x03, 0x06, 0x13, 0x15, 0x1b, 0x1c, 0x1d, 0x22, 0x23, 0x26, 0x2a, 0x2b, 0x2c, 0x33, 0x35, 0x38, 0x3e},
		},
		{
			defaultCards,
			{0x05, 0x08, 0x0a, 0x19, 0x1b, 0x1c, 0x22, 0x23, 0x29, 0x2b, 0x32, 0x33, 0x3b, 0x3c, 0x3e, 0xf0, 0xf1},
		},
	}
	for i, sample := range samples[:0] {
		cardutils.GetCardSystem().Init(sample[0])
		var cards [255]int
		for _, c := range sample[1] {
			cards[c]++
		}
		ai := &DoudizhuAI{
			Users: []*DoudizhuUser{
				{
					Cards:   cards[:],
					CardNum: len(cards),
				},
			},
			Helper: helper,
		}
		rs := helper.Split(sample[1])
		for _, res := range rs {
			t.Logf("result %d: %d", i, ai.Score(res))
		}
	}
}

// 两个人先手出牌
func TestDoudizhuSimpleTurn2(t *testing.T) {
	helper := NewDoudizhuHelper()
	samples := [][][]int{
		{
			{0x03, 0x05},
			{0x04},
			{0x05},
		},
		{
			{0x03, 0x4},
			{0x06},
			{0x03},
		},
		{
			{0x05, 0x04, 0x14},
			{0x08},
			{0x04, 0x14},
		},
		{
			{0x05, 0x04, 0x14},
			{0x08},
			{0x04, 0x14},
		},
		{
			{0x03, 0x04, 0x05, 0x06, 0x16},
			{0x06, 0x16},
			{0x03},
		},
		// 5
		{
			{0x05, 0x04, 0x14, 0x09},
			{0x08, 0x18},
			{0x05},
		},
		{
			{0x03, 0x13},
			{0x08, 0x18},
			{0x03, 0x13},
		},
		{
			{0x04, 0x17},
			{0x08, 0x18},
			{0x04},
		},
		{
			{0x04, 0x14, 0x24, 0x05, 0x15, 0x25, 0x06, 0x16, 0x07, 0x08},
			{0x0a, 0x1a},
			{0x04, 0x14, 0x24, 0x05, 0x15, 0x25, 0x07, 0x08},
		},
		{
			{0x04, 0x05, 0x06, 0x07, 0x08, 0x14},
			{0x08, 0x18},
			{0x14, 0x05, 0x06, 0x07, 0x08},
		},
		// 10
		{
			{0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x03, 0x13, 0x23, 0x14},
			{0x08, 0x18},
			{0x04, 0x05, 0x06, 0x07, 0x08},
		},
		{
			{0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x19, 0x29, 0x13},
			{0x0a, 0x1a},
			{0x04, 0x05, 0x06, 0x07, 0x08},
		},

		{
			{0x03, 0x13, 0x23, 0x04, 0x14, 0x05, 0x15, 0x19, 0x29},
			{0x08, 0x18},
			{0x03, 0x13, 0x04, 0x14, 0x05, 0x15},
		},
		{
			{0x05, 0x15, 0x09},
			{0x08, 0x18},
			{0x09},
		},
		{
			{0x03, 0x05, 0x15, 0x09},
			{0x08, 0x18},
			{0x03},
		},
		// 15
		{
			{0x04, 0x07},
			{0x05, 0x06, 0x7},
			{0x07},
		},
		{
			{0x04, 0xf0},
			{0x05, 0x06, 0x7, 0x8, 0x8, 0x9},
			{0xf0},
		},
		{
			{0x03, 0x13, 0x23, 0x04, 0x14, 0x24},
			{0x05, 0x06, 0x7, 0x8, 0x8, 0x9},
			{0x03, 0x13, 0x23, 0x04, 0x14, 0x24},
		},
		{
			{0x03, 0x13, 0x23, 0x04, 0x14, 0x24, 0x05, 0x06},
			{0x05, 0x06, 0x7, 0x8, 0x8, 0x9},
			{0x03, 0x13, 0x23, 0x04, 0x14, 0x24, 0x05, 0x06},
		},
		{
			{0x03, 0x13, 0x23, 0x04, 0x14, 0x24, 0x05, 0x15, 0x06, 0x16, 0x8},
			{0x05, 0x06, 0x7, 0x8, 0x8, 0x9},
			{0x03, 0x13, 0x23, 0x04, 0x14, 0x24, 0x05, 0x15, 0x06, 0x16},
		},
	}
	enableDebugDoudizhu = false
	for i, sample := range samples[:1] {
		var sortedCards []int
		var users []*DoudizhuUser
		for _, a := range sample[:2] {
			sortedCards = append(sortedCards, a...)
			cards := make([]int, 256)
			for _, c := range a {
				cards[c]++
			}
			user := &DoudizhuUser{Cards: cards, CardNum: len(a)}
			users = append(users, user)
		}
		cardutils.GetCardSystem().Init(sortedCards)
		ai := &DoudizhuAI{
			Users:  users,
			Helper: helper,
		}
		turn := ai.Turn()
		if !checkSameCards(sample[2], turn) {
			t.Errorf("result %d: %s", i, cardutils.Format(turn))
		}
	}
	enableDebugDoudizhu = false
}

// 两个人后手出牌
func TestDoudizhuTurn2(t *testing.T) {
	helper := NewDoudizhuHelper()
	samples := [][][]int{
		{
			{0x04, 0x08},
			{0x05},
			{0x03},
			{0x08},
		},
		{
			{0x04, 0x08},
			{0x05, 0x08},
			{0x03},
			{0x08},
		},
		{
			{0x04, 0x14, 0x08, 0x18},
			{0x05, 0x06, 0x7, 0x17, 0x16},
			{0x03, 0x13},
			{0x08, 0x18},
		},
		{
			{0x04, 0x14, 0x08, 0x18},
			{0x05, 0x06, 0x7, 0x17, 0x16},
			{0x03, 0x13},
			{0x08, 0x18},
		},
		{
			{0x04, 0x14, 0x24, 0x34},
			{0x05, 0x06, 0x7, 0x17, 0x16},
			{0x03, 0x13},
			{0x04, 0x14, 0x24, 0x34},
		},
		{
			{51, 4, 5, 56, 9, 57, 10, 42, 58, 11, 43, 59, 45, 61, 14, 2, 240},
			{0xf0},
			{8, 24, 40, 23, 39, 55, 38, 54, 19, 35},
			{0x0a, 0x2a, 0x3a, 0x0b, 0x2b, 0x3b, 0x09, 0x39, 0x2d, 0x3d},
		},
	}
	for i, sample := range samples[:] {
		var sortedCards []int
		var users []*DoudizhuUser
		for _, a := range sample[:2] {
			sortedCards = append(sortedCards, a...)
			cards := make([]int, 256)
			for _, c := range a {
				cards[c]++
			}
			user := &DoudizhuUser{Cards: cards, CardNum: len(a)}
			users = append(users, user)
		}
		users[1].Step = sample[2]
		cardutils.GetCardSystem().Init(sortedCards)
		ai := &DoudizhuAI{
			Users:  users,
			Helper: helper,
		}
		turn := ai.Turn()
		if !checkSameCards(sample[3], turn) {
			t.Errorf("result %d: %v", i, cardutils.Format(turn))
		}
	}
}

type doudizhuAIResult struct {
	ai   *DoudizhuAI
	turn []int
}

// 三个人出牌
func TestDoudizhuTurn3(t *testing.T) {
	helper := NewDoudizhuHelper()
	samples := []doudizhuAIResult{
		{ai: &DoudizhuAI{
			MySeat: 0, Dizhu: 0, UsedCards: []int{0x14, 0x35, 0x16, 0x17, 0x08, 0x38, 0x09, 0x2a, 0x3b, 0x1c}, Users: []*DoudizhuUser{{Cards: []int{0x02, 0x03, 0x0b, 0x12, 0x1a, 0x1b, 0x22, 0x26, 0x27, 0x28, 0x2b, 0x2c, 0x33, 0x39, 0x3d}, Step: []int{}, CardNum: 15}, {Cards: []int{}, Step: []int{0x38, 0x09, 0x2a, 0x3b, 0x1c}, CardNum: 12}, {Cards: []int{}, Step: []int{}, CardNum: 17}}},
			turn: []int{}},
		{ai: &DoudizhuAI{MySeat: 0, Dizhu: 2, UsedCards: []int{0x19, 0x38, 0x37, 0x06, 0x15, 0x34, 0x33, 0x1c, 0x3b, 0x0a, 0x09, 0x08, 0x05, 0x28, 0x3a, 0x0b, 0x3c, 0x3d, 0x32, 0xf0, 0xf1, 0x16, 0x36, 0x02, 0x12, 0x07, 0x18, 0x29, 0x2a, 0x1b, 0x2c, 0x0d, 0x2e}, Users: []*DoudizhuUser{{Cards: []int{0x03, 0x04, 0x23, 0x2d}, Step: []int{}, CardNum: 4}, {Cards: []int{}, Step: []int{}, CardNum: 15}, {Cards: []int{}, Step: []int{}, CardNum: 2}}},
			turn: []int{0x04}},
		{ai: &DoudizhuAI{MySeat: 0, Dizhu: 1, UsedCards: []int{0x05, 0x25, 0x35, 0x06, 0x26, 0x36, 0x03, 0x24, 0x0d, 0x1d, 0x3d, 0x2c, 0x08, 0x09, 0x0a, 0x2b, 0x0c, 0x2a, 0x0b, 0x3c, 0x2d, 0x0e, 0x07, 0x17, 0x27, 0x13, 0x38, 0x2e, 0x12, 0xf0, 0xf1, 0x23, 0x33, 0x18, 0x28, 0x1a, 0x3a, 0x34}, Users: []*DoudizhuUser{{Cards: []int{0x04, 0x14, 0x1b, 0x22, 0x32, 0x3b, 0x3e}, Step: []int{}, CardNum: 7}, {Cards: []int{}, Step: []int{}, CardNum: 1}, {Cards: []int{}, Step: []int{0x34}, CardNum: 8}}},
			turn: []int{0x32},
		},
		// 小王没出
		{ai: &DoudizhuAI{MySeat: 2, Dizhu: 2, UsedCards: []int{0x03, 0x33, 0x14, 0x24, 0x05, 0x15, 0x0b, 0x3b, 0x1c, 0x2c, 0x0d, 0x3d, 0x12, 0x22, 0x32, 0x13, 0x35, 0xf1, 0x07, 0x38, 0x09, 0x3a, 0x1b, 0x0c, 0x1d, 0x2e}, Users: []*DoudizhuUser{{Cards: []int{}, Step: []int{}, CardNum: 17}, {Cards: []int{}, Step: []int{}, CardNum: 6}, {Cards: []int{0x02, 0x06, 0x16, 0x26, 0x2d}, Step: []int{}, CardNum: 5}}}, turn: []int{0x06, 0x16, 0x26, 0x2d}},
		{ai: &DoudizhuAI{MySeat: 2, Dizhu: 2, UsedCards: []int{0x03, 0x33, 0x14, 0x24, 0x05, 0x15, 0x0b, 0x3b, 0x1c, 0x2c, 0x0d, 0x3d, 0x12, 0x22, 0x32, 0x13, 0x35}, Users: []*DoudizhuUser{{Cards: []int{}, Step: []int{}, CardNum: 17}, {Cards: []int{}, Step: []int{0x35}, CardNum: 6}, {Cards: []int{0x02, 0x06, 0x07, 0x09, 0x0c, 0x16, 0x1b, 0x1d, 0x26, 0x2d, 0x2e, 0x38, 0x3a, 0xf1}, Step: []int{}, CardNum: 14}}}, turn: []int{0x2d}},
		// 5
		{ai: &DoudizhuAI{MySeat: 0, Dizhu: 1, UsedCards: []int{0x02, 0x22, 0x32, 0x05, 0x14, 0x25, 0x26, 0x17, 0x08, 0x38, 0x39, 0x2a, 0x2b, 0x3c, 0x19, 0x3a, 0x0b, 0x1c, 0x1d, 0x04, 0x36, 0x3b, 0x2c}, Users: []*DoudizhuUser{{Cards: []int{0x06, 0x09, 0x23, 0x28, 0x2d, 0x34, 0x35, 0x37, 0x3d, 0x3e, 0xf1}, Step: []int{}, CardNum: 11}, {Cards: []int{}, Step: []int{0x2c}, CardNum: 4}, {Cards: []int{}, Step: []int{}, CardNum: 16}}}, turn: []int{0xf1}},
		{ai: &DoudizhuAI{MySeat: 1, Dizhu: 1, UsedCards: []int{}, Users: []*DoudizhuUser{{Cards: []int{}, Step: []int{}, CardNum: 17}, {Cards: []int{0x02, 0x04, 0x05, 0x08, 0x0b, 0x0e, 0x14, 0x17, 0x19, 0x1c, 0x1d, 0x1e, 0x22, 0x25, 0x26, 0x27, 0x2c, 0x32, 0x3a, 0xf0}, Step: []int{}, CardNum: 20}, {Cards: []int{}, Step: []int{}, CardNum: 17}}}, turn: []int{0x25, 0x26, 0x17, 0x08, 0x19}},
		{ai: &DoudizhuAI{MySeat: 1, Dizhu: 1, UsedCards: []int{0x25, 0x26, 0x17, 0x08, 0x19}, Users: []*DoudizhuUser{{Cards: []int{}, Step: []int{}, CardNum: 17}, {Cards: []int{0x02, 0x04, 0x05, 0x0b, 0x0e, 0x14, 0x1c, 0x1d, 0x1e, 0x22, 0x27, 0x2c, 0x32, 0x3a, 0xf0}, Step: []int{}, CardNum: 20}, {Cards: []int{}, Step: []int{}, CardNum: 17}}}, turn: []int{0x3a, 0x0b, 0x1c, 0x1d, 0x0e}},
		{ai: &DoudizhuAI{MySeat: 1, Dizhu: 1, UsedCards: []int{0x25, 0x26, 0x17, 0x08, 0x19, 0x3a, 0x0b, 0x1c, 0x1d, 0x0e}, Users: []*DoudizhuUser{{Cards: []int{}, Step: []int{}, CardNum: 17}, {Cards: []int{0x02, 0x04, 0x05, 0x14, 0x1e, 0x22, 0x27, 0x2c, 0x32, 0xf0}, Step: []int{}, CardNum: 20}, {Cards: []int{}, Step: []int{}, CardNum: 17}}}, turn: []int{0x05}},
		{ai: &DoudizhuAI{MySeat: 0, Dizhu: 1, UsedCards: []int{0x1b, 0x2b, 0x3b, 0x0c, 0x1c, 0x3c, 0x03, 0x14, 0x36, 0x07, 0x08, 0x19, 0x1a, 0x28, 0x09, 0x0a, 0x0b, 0x2c, 0x23, 0x17, 0x12}, Users: []*DoudizhuUser{{Cards: []int{0x02, 0x04, 0x22, 0x24, 0x25, 0x2d, 0x32, 0x37, 0x39, 0x3d, 0xf0}, Step: []int{}, CardNum: 11}, {Cards: []int{}, Step: []int{0x17}, CardNum: 6}, {Cards: []int{}, Step: []int{0x12}, CardNum: 16}}}, turn: []int{}},
		// 10
		{ai: &DoudizhuAI{MySeat: 1, Dizhu: 1, UsedCards: []int{0x04, 0x14, 0x34, 0x15, 0x25, 0x35, 0x23, 0x06, 0x07, 0x17, 0x27, 0x37, 0x33, 0x2a, 0x1b, 0x3c, 0x1e, 0x22, 0x24, 0x2b, 0x1d, 0x12, 0xf0, 0x26, 0x0d, 0x02, 0x0c, 0x1c, 0x2c, 0x05, 0x0e, 0x2e, 0x3e, 0x39, 0x1a, 0x0b, 0x2d, 0x32, 0x03, 0x13, 0x0a, 0x3a}, Users: []*DoudizhuUser{{Cards: []int{}, Step: []int{0x0a, 0x3a}, CardNum: 3}, {Cards: []int{0x08, 0x09, 0x18, 0x19, 0x28, 0x38}, Step: []int{}, CardNum: 6}, {Cards: []int{}, Step: []int{0x03, 0x13}, CardNum: 3}}}, turn: []int{0x08, 0x18, 0x28, 0x38}},
	}

	var cards = []int{0xf0, 0xf1}
	for color := 0; color < 4; color++ {
		for value := 0x02; value <= 0x0e; value++ {
			c := (color << 4) | value
			cards = append(cards, c)
		}
	}
	cardutils.GetCardSystem().Init(cards)

	enableDebugDoudizhu = false
	for i, sample := range samples[:] {
		ai := sample.ai
		ai.Helper = helper
		for _, user := range ai.Users {
			var cards [256]int
			for _, c := range user.Cards {
				cards[c]++
			}
			user.Cards = cards[:]
		}
		turn := ai.Turn()
		if !checkSameCards(turn, sample.turn) {
			t.Error("match", i, cardutils.Format(sample.turn), cardutils.Format(turn))
		}
	}
	enableDebugDoudizhu = false
}