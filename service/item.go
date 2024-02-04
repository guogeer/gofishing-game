package service

// 背包逻辑更新

import (
	"gofishing-game/internal/gameutils"
	"gofishing-game/internal/pb"
	"strings"

	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/util"
)

// 在item物品表出现
func isItemValid(id int) bool {
	if _, ok := config.Int("item", id, "id"); !ok {
		return false
	}
	s, _ := config.String("item", id, "visible")
	return strings.ToLower(s) != "no"
}

type ItemObj struct {
	player *Player

	items        []gameutils.Item
	offlineItems []*pb.NumericItem
}

func newItemObj(player *Player) *ItemObj {
	obj := &ItemObj{
		player: player,
	}
	return obj
}

func (obj *ItemObj) BeforeEnter() {
	obj.clearEmptyItems()
}

// 移除空物品
func (obj *ItemObj) clearEmptyItems() {
	obj.items = gameutils.MergeItems(obj.items)
}

func (obj *ItemObj) GetItems() []gameutils.Item {
	items := make([]gameutils.Item, 0, 4)
	for _, item := range obj.items {
		if item.GetNum() == 0 {
			items = append(items, item)
		}
	}
	return items
}

func (obj *ItemObj) IsEnough(id int, num int64) bool {
	return obj.NumItem(id) >= num
}

func (obj *ItemObj) Add(id int, num int64, way string) {
	obj.AddSome([]gameutils.Item{&gameutils.NumericItem{Id: id, Num: num}}, way)
}

func (obj *ItemObj) NumItem(id int) int64 {
	for _, item := range obj.items {
		if item.GetId() == id {
			return item.GetNum()
		}
	}
	return 0
}

func (obj *ItemObj) AddSome(items []gameutils.Item, way string) {
	obj.clearEmptyItems()

	obj.player.GameAction.OnAddItems(items, way)
	AddSomeItemLog(obj.player.Id, obj.items, way)
}

func (obj *ItemObj) GetItem(id int) gameutils.Item {
	for _, item := range obj.items {
		if item.GetId() == id {
			return item
		}
	}
	return nil
}

func (obj *ItemObj) LoadBin(data any) {
	bin := data.(*pb.UserBin)

	obj.items = make([]gameutils.Item, 0, 8)
	for _, item := range bin.Global.Bag.NumericItems {
		newItem := &gameutils.NumericItem{}
		util.DeepCopy(newItem, item)
		obj.addItem(newItem)
	}

	obj.offlineItems = nil
	// 对离线数据进行合并
	for _, item := range bin.Offline.Items {
		obj.addItem(&gameutils.NumericItem{Id: int(item.Id), Num: item.Num})
		obj.offlineItems = append(obj.offlineItems, &pb.NumericItem{Id: item.Id, Num: -item.Num})
	}
}

func (obj *ItemObj) SaveBin(data any) {
	bin := data.(*pb.UserBin)

	var numericItems []*pb.NumericItem
	for _, item := range obj.items {
		newItem := &pb.NumericItem{}
		util.DeepCopy(newItem, item)
		numericItems = append(numericItems, newItem)
	}
	bin.Global.Bag.NumericItems = numericItems
	bin.Offline.Items, obj.offlineItems = obj.offlineItems, nil
}

func (obj *ItemObj) addItem(newItem gameutils.Item) {
	if isItemValid(newItem.GetId()) {
		obj.items = append(obj.items, newItem)
	}
}
