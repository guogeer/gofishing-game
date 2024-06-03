package cardrule

import (
	"gofishing-game/internal/cardutils"
	"testing"
)

func TestMahjong(t *testing.T) {
	var cards []int
	for _, c := range []int{
		1, 2, 3, 4, 5, 6, 7, 8, 9,
		21, 22, 23, 24, 25, 26, 27, 28, 29,
		41, 42, 43, 44, 45, 46, 47, 48, 49,
		100, 110, 120,
	} {
		cards = append(cards, c, c, c, c)
	}

	samples := []int{1, 1, 1, 2, 2, 2, 3, 3, 3, 21, 22, 23, 22, 22}
	// samples = []int{4, 5, 6, 7, 7, 7, 25, 26, 28, 45, 46, 47, 100, 100}
	// samples = []int{2, 3, 4, 22, 22, 26, 27, 28, 41, 42, 43, 100}
	// samples = []int{1, 4, 5, 7, 8, 8, 9, 41, 43, 43, 47}
	cardutils.AddCardSystem("mahjong", cards)

	/*cs := NewCardSet()
	cs.Shuffle()
	cs.MoveBack([]int{1, 2, 3, 4, 5, 2, 2, 2})
	cs.MoveFront(1)
	c1 := cs.Deal()
	cs.MoveFront(2)
	c2 := cs.Deal()
	cs.MoveFront(1)
	c3 := cs.Deal()
	cs.MoveFront(3)
	t.Log(c1, c2, c3, cs.randCards)
	return
	*/

	cards = make([]int, MaxMahjongCard)
	for _, c := range samples {
		cards[c]++
	}
	helper := &MahjongHelper{Qingyise: true, name: "mahjong"}
	// opts := helper.SplitN(cards, 30)
	// t.Log(opts)

	remainingCards := make([]int, 255)
	for _, c := range cardutils.GetCardSystem("mahjong").GetAllCards() {
		remainingCards[c] = 4
	}
	ctx := &Context{
		ValidCards: remainingCards,
	}
	/*
		samples = []int{1, 1, 2, 2, 3, 3, 4, 4, 9, 21, 22, 23, 24}
		samples = []int{9, 8, 8, 7, 7, 7, 6, 5}
		samples = []int{3, 4, 5, 6, 7, 8, 44, 44, 46, 46, 49}
		samples = []int{4, 6, 7, 7, 9}
		samples = []int{2, 2, 3, 3, 4}
		samples = []int{2, 4, 4, 6, 7}
		samples = []int{3, 3, 26, 22, 23, 24, 24, 25}
	*/
	samples = []int{3, 3, 3, 4, 7}
	cards = make([]int, MaxMahjongCard)
	for _, c := range cardutils.GetCardSystem("mahjong").GetAllCards() {
		remainingCards[c] = 4
	}
	for _, c := range samples {
		cards[c]++
		remainingCards[c]--
	}
	remainingCards[46] = 0
	ctx.Cards = cards
	dc, w := helper.Weight(ctx)
	t.Log(dc, w)
}
