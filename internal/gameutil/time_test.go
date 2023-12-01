package gameutil

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/guogeer/quasar/util"
)

func TestCD(t *testing.T) {
	cd1 := NewCD(10000 * time.Millisecond)
	if !util.EqualJSON(cd1, []int{10000, 10000}) {
		t.Error("cd wrong")
	}
}

func TestClock(t *testing.T) {
	c := NewClock(1500 * time.Millisecond)
	if !util.EqualJSON(c, 1500) {
		t.Error(json.Marshal(c))
	}
}
