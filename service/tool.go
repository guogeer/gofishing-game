// 测试工具
package service

import (
	"encoding/json"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"gofishing-game/internal"
	"gofishing-game/internal/env"
	"gofishing-game/internal/gameutils"

	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
	"github.com/guogeer/quasar/util"
)

const toolFuncPrefix = "Test_"

var (
	testToolNames    []string
	testToolHandlers []any
)

type toolArgs struct {
	Id     int
	Params string
}

func init() {
	cmd.BindWithName("GetTestTools", funcGetTestTools, (*toolArgs)(nil))
	cmd.BindWithName("UseTestTool", funcUseTestTool, (*toolArgs)(nil))

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
	ply.WriteJSON("GetTestTools", map[string]any{"Tools": tools})
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
	WriteMessage(ply.enterReq.session, "", "ServerClose", cmd.M{
		"ServerName": ply.enterReq.ServerName,
	})
}

func (tool *testTool) Test_Z增加各种数值(ctx *cmd.Context, params string) {
	ply := GetPlayerByContext(ctx)
	items := [][2]int{
		{1107, 10000}, {1104, 10000}, {10001, 10}, {10003, 10},
		{11002, 10}, {11004, 10}, {12002, 10}, {12003, 10}, {20002, 1}, {1110, 1000000},
		{20001, 1}, {1300, 10000}, {1401, 10000}, {1111, 100},
	}

	buf, _ := json.Marshal(items)
	ply.ItemObj().AddSome(gameutils.ParseItems(string(buf)), util.GUID())
}

func (tool *testTool) Test_S升级到X(ctx *cmd.Context, params string) {
	ply := GetPlayerByContext(ctx)

	addExp := int64(0)
	toLevel, _ := strconv.Atoi(params)
	for i := ply.Level + 1; i < config.NumRow("level") && i <= toLevel; i++ {
		exp, _ := config.Int("level", i, "Exp")
		addExp += exp
	}

	ply.ItemObj().Add(1110, addExp, util.GUID())
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
		ply.WriteJSON("Prompt", cmd.M{"Msg": "格式：物品ID,物品数量"})
	}
	ply.ItemObj().Add(itemId, int64(num), util.GUID())
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
