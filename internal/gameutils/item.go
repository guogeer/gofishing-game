package gameutils

import (
	"encoding/json"
	"sort"
)

const (
	ItemIdGold = 1001
	ItemIdExp  = 1002
)

type Item interface {
	GetId() int
	GetNum() int64
	Merge(Item) bool
	Multi(n int)
}

type NumericItem struct {
	Id  int   `json:"id,omitempty"`
	Num int64 `json:"num,omitempty"`
}

func (item *NumericItem) Merge(mergeItem Item) bool {
	if item.Id != mergeItem.GetId() {
		return false
	}
	item.Num += mergeItem.GetNum()
	return true
}

func (item *NumericItem) GetId() int {
	return item.Id
}

func (item *NumericItem) GetNum() int64 {
	return item.Num
}

func (item *NumericItem) Multi(n int) {
	item.Num *= int64(n)
}

// 格式：[[1000,1],[1001,2]]
func ParseNumbericItems(s string) []Item {
	var items []Item
	var a [][2]int64
	json.Unmarshal([]byte(s), &a)
	for _, v := range a {
		items = append(items, &NumericItem{Id: int(v[0]), Num: v[1]})
	}
	return items
}

func MergeItems(items []Item) []Item {
	sort.Slice(items, func(i, j int) bool {
		return items[i].GetId() < items[j].GetId()
	})

	var mergeItems []Item
	for i, j := 0, 0; i < len(items); i = j {
		for j = i + 1; j < len(items) && items[i].GetId() == items[j].GetId(); j++ {
			if items[j].GetNum() != 0 && !items[i].Merge(items[j]) {
				mergeItems = append(mergeItems, items[j])
			}
		}
		if items[i].GetNum() != 0 {
			mergeItems = append(mergeItems, items[i])
		}
	}

	return mergeItems
}

func CountItems(items []Item, itemId int) int64 {
	var num int64
	for _, item := range items {
		if item.GetId() == itemId {
			num += item.GetNum()
		}
	}
	return num
}
