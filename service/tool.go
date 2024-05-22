// 测试工具
package service

import (
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/guogeer/quasar/utils"

	"gofishing-game/internal"
	"gofishing-game/internal/env"
	"gofishing-game/internal/gameutils"

	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
)

const toolFuncPrefix = "Test_"

var (
	testToolNames    []string
	testToolHandlers []any
)

type toolArgs struct {
	Id     int    `json:"id,omitempty"`
	Params string `json:"params,omitempty"`
}

func init() {
	cmd.Bind("GetTestTools", funcGetTestTools, (*toolArgs)(nil))
	cmd.Bind("UseTestTool", funcUseTestTool, (*toolArgs)(nil))

	AddTestTool(&testTool{})
}

func AddTestTool(toolFuncs any) {
	handlers := reflect.ValueOf(toolFuncs)
	for i := 0; i < handlers.NumMethod(); i++ {
		fn := handlers.Type().Method(i)
		if strings.Index(fn.Name, toolFuncPrefix) == 0 {
			testToolNames = append(testToolNames, fn.Name[len(toolFuncPrefix):])
		}
	}
	testToolHandlers = append(testToolHandlers, toolFuncs)
	sort.Strings(testToolNames)
}

func getTestTools(uid int) []string {
	if env.Config().Environment == "test" {
		return testToolNames
	}

	uidStr := strconv.Itoa(uid)
	godStr, _ := config.String("config", "God", "Value")
	gods := strings.Split(godStr, ",")
	if internal.IndexArrayFunc(gods, func(i int) bool { return gods[i] == uidStr }) >= 0 {
		return testToolNames
	}
	return nil
}

func funcGetTestTools(ctx *cmd.Context, data any) {
	// args := data.(*ToolArgs)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	log.Debugf("player %d get test tools", ply.Id)
	tools := getTestTools(ply.Id)
	ply.WriteJSON("getTestTools", map[string]any{"tools": tools})
}

func funcUseTestTool(ctx *cmd.Context, data any) {
	args := data.(*toolArgs)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	log.Debugf("player %d use test tool %d", ply.Id, args.Id)

	tools := getTestTools(ply.Id)
	if args.Id < 0 || args.Id >= len(tools) {
		return
	}
	for _, tool := range testToolHandlers {
		fn := reflect.ValueOf(tool).MethodByName(toolFuncPrefix + tools[args.Id])
		if fn.IsValid() {
			fn.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(args.Params)})
		}
	}
}

type testTool struct{}

func (tool *testTool) Test_Q强制断线(ctx *cmd.Context, params string) {
	ply := GetPlayerByContext(ctx)
	WriteMessage(ply.session, "serverClose", cmd.M{
		"serverId": GetServerId(),
		"cause":    "test tool",
	})
}

func (tool *testTool) Test_Z增加各种数值(ctx *cmd.Context, params string) {
	ply := GetPlayerByContext(ctx)

	var items []gameutils.Item
	for _, rowId := range config.Rows("item") {
		id, _ := config.Int("item", rowId, "id")
		items = append(items, &gameutils.NumericItem{Id: int(id), Num: 9999})
	}

	ply.BagObj().AddSomeItems(items, "tool")
}

func (tool *testTool) Test_S升级到X(ctx *cmd.Context, params string) {
	ply := GetPlayerByContext(ctx)

	addExp := int64(0)
	toLevel, _ := strconv.Atoi(params)
	for i := ply.Level + 1; i < config.NumRow("level") && i <= toLevel; i++ {
		exp, _ := config.Int("level", i, "Exp")
		addExp += exp
	}

	ply.BagObj().Add(1110, addExp, utils.GUID())
}

func (tool *testTool) Test_L离开游戏(ctx *cmd.Context, params string) {
	ply := GetPlayerByContext(ctx)
	ply.Leave()
}

func (tool *testTool) Test_Z增加指定物品数量(ctx *cmd.Context, params string) {
	ply := GetPlayerByContext(ctx)

	ss := strings.Split(params+",,", ",")
	itemId, _ := strconv.Atoi(ss[0])
	num, _ := strconv.Atoi(ss[1])
	if itemId == 0 {
		ply.WriteJSON("prompt", cmd.M{"msg": "格式：物品ID,物品数量"})
	}
	ply.BagObj().Add(itemId, int64(num), utils.GUID())
}

type testReseter interface {
	TestReset()
}

func (tool *testTool) Test_Y一键重置(ctx *cmd.Context, params string) {
	ply := GetPlayerByContext(ctx)
	for _, action := range ply.enterActions {
		if h, ok := action.(testReseter); ok {
			h.TestReset()
		}
	}
}
