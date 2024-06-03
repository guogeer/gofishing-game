package cardutils

import (
	"testing"
)

func TestCheat(t *testing.T) {
	AddCardSystem("test", []int{1, 1, 2, 2, 3, 3, 1, 2, 3})
	cs := NewCardSet("test")

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
