package gameutils

import (
	"quasar/utils"
	"testing"
)

func TestInitNilFields(t *testing.T) {
	type AA struct {
		NA int
	}
	type A struct {
		M   map[string]int32
		AA  *AA
		AA2 AA
		MA  map[string]*AA
		N   *int
	}
	a := &A{}
	n := 0
	ans := &A{
		M:   map[string]int32{},
		AA:  &AA{},
		AA2: AA{},
		MA:  map[string]*AA{},
		N:   &n,
	}
	InitNilFields(a)
	if !utils.EqualJSON(a, ans) {
		t.Error(a, ans)
	}
}
