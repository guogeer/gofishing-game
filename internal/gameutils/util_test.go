package gameutils

import (
	"quasar/utils"
	"testing"
)

func TestAddStructFields(t *testing.T) {
	type structA struct {
		N  int
		N2 int64
		N3 uint32
		N4 float64
		N5 string
		N6 bool
	}
	samples := [3]structA{
		{N: 3, N2: 10, N3: 300, N4: 3000.3, N5: "hi", N6: true},
		{N: 1, N2: 10, N3: 100, N4: 1000.1, N5: "hi", N6: true},
		{N: 2, N2: 0, N3: 200, N4: 2000.2, N5: "hi b", N6: true},
	}
	AddStructFields(&samples[1], &samples[2])
	if !utils.EqualJSON(samples[0], samples[1]) {
		t.Error(samples[0], samples[1])
	}
}

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

func TestConvertMap(t *testing.T) {
	type A struct {
		N int
		S string
		n int
	}
	a := &A{N: 100, S: "123", n: 2}
	m := map[string]any{
		"N": 100,
		"S": "123",
	}
	a2 := ConvertMap(a)
	m2 := ConvertMap(m)
	if !utils.EqualJSON(a, a2) {
		t.Error(a, a2)
	}
	if !utils.EqualJSON(m, m2) {
		t.Error(a, a2)
	}
}
