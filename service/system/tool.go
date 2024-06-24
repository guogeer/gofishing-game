package system

import (
	"time"

	"gofishing-game/internal/gameutils"
	"gofishing-game/service"

	"github.com/guogeer/quasar/v2/cmd"
	"github.com/guogeer/quasar/v2/config"
)

type testTool struct {
	Id     int
	ItemId int
}

func init() {
	service.AddTestTool(&testTool{})
}

func (tool *testTool) Test_Q清理物品(ctx *cmd.Context, params string) {
	ply := service.GetPlayerByContext(ctx)

	addItems := []gameutils.Item{}
	for _, rowId := range config.Rows("item") {
		itemId, _ := config.Int("item", rowId, "shopid")
		num := ply.BagObj().NumItem(int(itemId))

		addItems = append(addItems, &gameutils.NumericItem{Id: int(itemId), Num: -num})
	}
	ply.BagObj().AddSomeItems(addItems, "tool")
}

func (tool *testTool) Test_Q签到可领(ctx *cmd.Context, params string) {
	ply := service.GetPlayerByContext(ctx)

	signInObj := getSignInObj(ply)
	signInObj.drawTime = time.Now().Add(-24 * time.Hour)
	signInObj.Look()
}
