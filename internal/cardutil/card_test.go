package cardutil

import (
	"math/rand"
	"testing"
	"time"
)

func TestCheat(t *testing.T) {
	rand.Seed(time.Now().Unix())
	GetCardSystem().Init([]int{1, 1, 2, 2, 3, 3, 1, 2, 3})
	cs := NewCardSet()

	samples := [][]int{
		{3, 2, 1},
		{1, 1, 2},
		{3, 2, 1, 1, 1, 2},
		{1, 1, 1},
		{1, 1, 2},
		{1, 1, 1},
		{1, 1, 1},
	}

	for _, sample := range samples {
		cs.Shuffle()
		cs.Cheat(sample[:2]...)
		cs.Cheat(sample[2:]...)
	}
}
