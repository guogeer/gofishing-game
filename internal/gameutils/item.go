package gameutils

import (
	"encoding/json"
)

const (
	ItemIdGold = 1000 // 金币
	ItemIdExp  = 1001 // exp
)

type Item struct {
	Id       int   `json:"id,omitempty"`
	Num      int64 `json:"num,omitempty"`
	ExpireTs int64 `json:"expireTs,omitempty"`
	UpdateTs int64 `json:"updateTs,omitempty"`
	MaxValue int64 `json:"maxValue,omitempty"`

	params map[string]any // 自定义参数
}

// 数字型物品
func (item *Item) IsNumeric() bool {
	return len(item.params) == 0
}

func (item *Item) IsEmpty() bool {
	if item.IsNumeric() {
		return item.Num == 0
	}
	return len(item.params) == 0
}

func (item *Item) SetParams(key string, value any) {
	if item.params == nil {
		item.params = map[string]any{}
	}
	item.params[key] = value
}

func (item *Item) MarshalJSON() ([]byte, error) {
	m := map[string]any{
		"Id":  item.Id,
		"Num": item.Num,
	}

	for k, v := range item.params {
		m[k] = v
	}
	return json.Marshal(m)
}

func FormatItems(items []*Item) string {
	if len(items) == 0 {
		return ""
	}

	var values [][]int64
	for _, item := range items {
		values = append(values, []int64{int64(item.Id), item.Num})
	}
	buf, _ := json.Marshal(values)
	return string(buf)
}

// 格式：[[1000,1],[1001,2]]
func ParseItems(s string) []*Item {
	if s == "0" || s == "" {
		return nil
	}

	var items []*Item
	var a [][2]int64
	json.Unmarshal([]byte(s), &a)
	for _, v := range a {
		items = append(items, &Item{Id: int(v[0]), Num: v[1]})
	}
	return items
}

func MergeItems(items ...[]*Item) []*Item {
	set := map[int]*Item{}
	sumItems := make([]*Item, 0, 4)
	for _, item2 := range items {
		for _, item := range item2 {
			newItem := *item
			// 特殊的物品不合并
			if !newItem.IsNumeric() {
				sumItems = append(sumItems, &newItem)
			} else if oldItem, ok := set[newItem.Id]; ok {
				oldItem.Num += newItem.Num
			} else {
				set[newItem.Id] = &newItem
			}
		}
	}

	for _, item := range set {
		if !item.IsEmpty() {
			sumItems = append(sumItems, item)
		}
	}
	return sumItems
}

// 物品日志
// struct tag新增client，带给客户端的数据
type ItemLog struct {
	Way     string  `client:"Way"`
	IsTemp  bool    `json:",omitempty"`
	IsBatch bool    `json:",omitempty"`
	IsNoLog bool    `json:",omitempty"`
	IsQuiet bool    `json:",omitempty"` // 不通知客户端
	Kind    string  `json:",omitempty"` // 类型。user：玩家内部流通；sys：系统产出回收
	Uuid    string  `json:",omitempty"`
	Items   []*Item `json:",omitempty" client:"Items"`

	SubId       int    `json:",omitempty"`
	OtherId     int    `json:",omitempty"`
	IsTestPay   bool   `json:",omitempty"`
	Round       int    `json:",omitempty"` // 限时活动阶段
	IsFix       bool   `json:",omitempty"` // 修正数值，跳过增加物品
	OrderId     string `json:",omitempty"` // 付费订单ID
	RegisterFee int    `json:",omitempty"` // 当前选卡费用
	IsFirstPay  bool   `json:",omitempty"`

	// CLIENT
	ShopId    int    `json:",omitempty"`
	ClientWay string `json:",omitempty" client:"Way"` // 修正客户端Way
	PaySDK    string `json:",omitempty"`              // 支付SDK。google,coda
	IsDouble  bool   `json:",omitempty"`              // 奖励加倍
}
