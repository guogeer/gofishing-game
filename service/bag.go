package service

// 背包逻辑更新

import (
	"gofishing-game/internal/gameutils"
	"gofishing-game/internal/pb"
	"strings"

	"github.com/guogeer/quasar/config"
)

// 在item物品表出现
func isItemValid(id int) bool {
	if _, ok := config.Int("item", id, "id"); !ok {
		return false
	}
	s, _ := config.String("item", id, "visible")
	return strings.ToLower(s) != "no"
}

type bagObj struct {
	player *Player

	items        []gameutils.Item
	offlineItems []*pb.NumericItem
}

func newBagObj(player *Player) *bagObj {
	obj := &bagObj{
		player: player,
	}
	obj.player.DataObj().Push(obj)
	return obj
}

func (obj *bagObj) BeforeEnter() {
	obj.clearEmptyItems()
}

// 移除空物品
func (obj *bagObj) clearEmptyItems() {
	obj.items = gameutils.MergeItems(obj.items)
}

func (obj *bagObj) GetItems() []gameutils.Item {
	items := make([]gameutils.Item, 0, 4)
	for _, item := range obj.items {
		if item.GetNum() != 0 {
			items = append(items, item)
		}
	}
	return items
}

func (obj *bagObj) IsEnough(id int, num int64) bool {
	return obj.NumItem(id) >= num
}

func (obj *bagObj) Add(id int, num int64, way string) {
	obj.AddSomeItems([]gameutils.Item{&gameutils.NumericItem{Id: id, Num: num}}, way)
}

func (obj *bagObj) NumItem(id int) int64 {
	for _, item := range obj.items {
		if item.GetId() == id {
			return item.GetNum()
		}
	}
	return 0
}

func (obj *bagObj) CostSomeItems(items []gameutils.Item, way string) {
	for _, item := range items {
		item.Multi(-1)
		obj.addItem(item)
	}
	obj.items = gameutils.MergeItems(obj.items)

	obj.player.GameAction.OnAddItems(items, way)
	AddSomeItemLog(obj.player.Id, obj.items, way)
}

func (obj *bagObj) AddSomeItems(items []gameutils.Item, way string) {
	for _, item := range items {
		obj.addItem(item)
	}
	obj.items = gameutils.MergeItems(obj.items)

	obj.player.GameAction.OnAddItems(items, way)
	AddSomeItemLog(obj.player.Id, obj.items, way)
}

func (obj *bagObj) GetItem(id int) gameutils.Item {
	for _, item := range obj.items {
		if item.GetId() == id {
			return item
		}
	}
	return nil
}

func (obj *bagObj) Load(data any) {
	bin := data.(*pb.UserBin)

	obj.items = make([]gameutils.Item, 0, 8)
	for _, item := range bin.Global.Bag.NumericItems {
		newItem := &gameutils.NumericItem{Id: int(item.Id), Num: item.Num}
		obj.addItem(newItem)
	}

	obj.offlineItems = nil
	// 对离线数据进行合并
	for _, item := range bin.Offline.Items {
		obj.addItem(&gameutils.NumericItem{Id: int(item.Id), Num: item.Num})
		obj.offlineItems = append(obj.offlineItems, &pb.NumericItem{Id: item.Id, Num: -item.Num})
	}
}

func (obj *bagObj) Save(data any) {
	bin := data.(*pb.UserBin)

	var numericItems []*pb.NumericItem
	for _, item := range obj.items {
		newItem := &pb.NumericItem{Id: int32(item.GetId()), Num: item.GetNum()}
		numericItems = append(numericItems, newItem)
	}
	bin.Global.Bag = &pb.Bag{
		NumericItems: numericItems,
	}
	if bin.Offline == nil {
		bin.Offline = &pb.OfflineBin{}
	}
	bin.Offline.Items, obj.offlineItems = obj.offlineItems, nil
}

func (obj *bagObj) addItem(newItem gameutils.Item) {
	if isItemValid(newItem.GetId()) {
		obj.items = append(obj.items, newItem)
	}
}
