package service

// 背包逻辑更新

import (
	"strings"

	"gofishing-game/internal/gameutils"

	"github.com/guogeer/quasar/util"

	"github.com/guogeer/quasar/config"
)

// 在item物品表出现
func isItemValid(id int) bool {
	if _, ok := config.Int("item", id, "ShopID"); !ok {
		return false
	}
	s, _ := config.String("item", id, "Visible")
	return strings.ToLower(s) != "no"
}

type ItemObj struct {
	player *Player
}

func newItemObj(player *Player) *ItemObj {
	obj := &ItemObj{
		player: player,
	}
	return obj
}

func (obj *ItemObj) BeforeEnter() {
	obj.update()
}

// 更新背包，清理无效的物品
func (obj *ItemObj) update() {
	p := obj.player

	var cur int
	for k, item := range p.dataObj.items {
		for _, item2 := range p.dataObj.items[k:] {
			if item.IsNumeric() && item2.IsNumeric() && item != item2 && item.Id == item2.Id {
				item.Num, item2.Num = item.Num+item2.Num, 0
			}
		}
		if item.Num != 0 {
			p.dataObj.items[cur] = item
			cur++
		}
	}
	p.dataObj.items = p.dataObj.items[:cur]
}

func (obj *ItemObj) GetItems() []*gameutils.Item {
	p := obj.player
	items := make([]*gameutils.Item, 0, 4)
	for _, item := range p.dataObj.items {
		if item.Num > 0 {
			items = append(items, item)
		}
	}
	return items
}

func (obj *ItemObj) IsEnough(id int, num int64) bool {
	return obj.NumItem(id) >= num
}

func (obj *ItemObj) Add(id int, num int64, way string) {
	obj.AddSome([]*gameutils.Item{{Id: id, Num: num}}, way)
}

func (obj *ItemObj) NumItem(id int) (sum int64) {
	p := obj.player
	for _, item := range p.dataObj.items {
		if item.Id == id {
			sum += item.Num
		}
	}
	return
}

func (obj *ItemObj) AddSome(items []*gameutils.Item, way string) {
	kindAndWay := strings.Split(way, ".")
	if len(kindAndWay) == 1 {
		kindAndWay = []string{"", kindAndWay[0]}
	}
	obj.AddByLog(&gameutils.ItemLog{
		Items: items,
		Way:   kindAndWay[1],
		Uuid:  util.GUID(),
		Kind:  kindAndWay[0],
	})
}

func (obj *ItemObj) AddByLog(itemLog *gameutils.ItemLog) {
	p := obj.player
	if itemLog.Uuid == "" {
		itemLog.Uuid = util.GUID()
	}
	if itemLog.Kind == "" {
		itemLog.Kind = "sys"
	}

	itemLog.Items = gameutils.MergeItems(itemLog.Items)
	//需要区分离线数据
	for _, item := range itemLog.Items {
		n := obj.NumItem(item.Id)
		if item.MaxValue > 0 && item.Num+n > item.MaxValue {
			item.Num = item.MaxValue - n
		}
		if item.Num+n <= 0 {
			item.Num = -n
		}

		if !itemLog.IsTemp && item.Num != 0 {
			if obj.GetItem(item.Id) == nil {
				newItem := *item // copy
				p.dataObj.addItem(&newItem)
			} else {
				bagItem := obj.GetItem(item.Id)
				bagItem.Num += item.Num
			}
		}
	}
	itemLog.Items = gameutils.MergeItems(itemLog.Items)
	//log.Debugf("ply %v add items %v", p.Id, itemLog.Items)

	p.GameAction.OnAddItems(itemLog)
	//添加日志
	if !itemLog.IsTemp && !itemLog.IsNoLog {
		AddSomeItemLog(p.Id, itemLog)
	}
}

func (obj *ItemObj) GetItem(id int) *gameutils.Item {
	p := obj.player
	for _, item := range p.dataObj.items {
		if item.Id == id {
			return item
		}
	}
	return nil
}
