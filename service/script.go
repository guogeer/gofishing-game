package service

import (
	"fmt"
	"strings"
	"sync"

	"gofishing-game/internal/env"
	"gofishing-game/internal/errcode"
	"gofishing-game/internal/gameutils"

	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
	"github.com/guogeer/quasar/script"
	"github.com/guogeer/quasar/util"
	lua "github.com/yuin/gopher-lua"
	luajson "layeh.com/gopher-json"
	luahelper "layeh.com/gopher-luar"
)

type scriptFuncEntry struct {
	FileName string
	FuncName string
}

var cmdEntries sync.Map

func scriptCreateUUID(L *lua.LState) int {
	uuid := util.GUID()
	L.Push(lua.LString(uuid))
	return 1
}

func scriptGetServerName(L *lua.LState) int {
	name := GetName()
	L.Push(lua.LString(name))
	return 1
}

func scriptGetTableCell(L *lua.LState) int {
	table := L.ToString(1)
	col := L.ToString(3)
	typ := L.ToString(4)

	rowKey := L.ToString(2)
	if ud := L.ToUserData(2); ud != nil {
		rowKey = fmt.Sprintf("%v", ud.Value)
	}

	v := lua.LNil
	switch strings.ToLower(typ) {
	case "duration", "millisecond", "milliseconds":
		if d, ok := config.Duration(table, rowKey, col); ok {
			v = lua.LNumber(d.Milliseconds())
		}
	case "second", "seconds":
		if d, ok := config.Duration(table, rowKey, col); ok {
			v = lua.LNumber(d.Seconds())
		}
	case "time":
		if t, ok := config.Time(table, rowKey, col); ok {
			v = lua.LNumber(t.Unix())
		}
	case "number":
		if n, ok := config.Float(table, rowKey, col); ok {
			v = lua.LNumber(n)
		}
	case "bool":
		if s, ok := config.String(table, rowKey, col); ok {
			s = strings.ToLower(s)
			v = lua.LBool(s != "false" && s != "0")
		}
	case "json":
		if s, ok := config.String(table, rowKey, col); ok {
			luav, err := luajson.Decode(L, []byte(s))
			if err != nil {
				log.Errorf("table cell %s:%s:%s value (%s) expect %s, error: %v", table, rowKey, col, s, typ, err)
			}
			v = luav
		}
	default:
		if s, ok := config.String(table, rowKey, col); ok {
			v = lua.LString(s)
		}
	}
	L.Push(v)
	return 1
}

func scriptLog(L *lua.LState) int {
	tag := L.ToString(1)
	msg := L.ToString(2)
	log.Printf(tag, msg)
	return 0
}

func scriptGetError(L *lua.LState) int {
	key := L.ToString(1)
	e := errcode.Get(key)
	L.Push(luahelper.New(L, e))
	return 1
}

func scriptGetTableRows(L *lua.LState) int {
	name := L.ToString(1)
	rows := config.Rows(name)
	L.Push(luahelper.New(L, rows))
	return 1
}

func scriptGetPlayer(L *lua.LState) int {
	uid := L.ToInt(1)
	p := GetPlayer(uid)
	if p == nil {
		p = &Player{}
	}
	L.Push(luahelper.New(L, p))
	return 1
}

func scriptParseItems(L *lua.LState) int {
	reward := L.ToString(1)
	items := gameutils.ParseItems(reward)
	L.Push(luahelper.New(L, items))
	return 1
}

func scriptCall(L *lua.LState) int {
	fileName := L.ToString(1)
	funcName := L.ToString(2)

	args := make([]any, 0, 4)
	for i := 3; i <= L.GetTop(); i++ {
		args = append(args, L.Get(i))
	}
	script.Call(fileName, funcName, args...)
	return 1
}

type luaArgs map[string]any

func handleScriptBind(ctx *cmd.Context, i any) {
	args := i.(*luaArgs)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}

	name := ctx.MsgId
	log.Debugf("try call lua function:%s %d", name, ply.Id)
	val, ok := cmdEntries.Load(name)
	if !ok {
		return
	}
	entry := val.(*scriptFuncEntry)
	script.Call(entry.FileName, entry.FuncName, ply.GameAction, *args)
}

func scriptBind(L *lua.LState) int {
	funcName := L.ToString(1)
	if L.GetGlobal(funcName) == nil {
		log.Errorf("bind function:%s not exists", funcName)
		return 0
	}

	fileName, ok := script.GetFileName(L)
	if !ok {
		log.Errorf("bind script:%s func:%s is invalid", fileName, funcName)
		return 0
	}
	cmdEntries.Store(funcName, &scriptFuncEntry{
		FileName: fileName,
		FuncName: funcName,
	})
	cmd.Bind(funcName, handleScriptBind, (*luaArgs)(nil))
	return 0
}

func externScript(L *lua.LState) int {
	luajson.Preload(L)

	exports := map[string]lua.LGFunction{
		"create_uuid":     scriptCreateUUID,
		"get_server_name": scriptGetServerName,
		"get_table_cell":  scriptGetTableCell,
		"log":             scriptLog,
		"get_error":       scriptGetError,
		"get_table_rows":  scriptGetTableRows,
		"get_player":      scriptGetPlayer,
		"parse_items":     scriptParseItems,
		"call":            scriptCall,
		"bind":            scriptBind,
	}
	mod := L.SetFuncs(L.NewTable(), exports)
	L.Push(mod)
	return 1
}

type scriptArgs struct {
	Name string `json:"name,omitempty"`
}

// 重新加载本地的脚本
func funcEffectLocalScript(ctx *cmd.Context, in any) {
	args := in.(*scriptArgs)
	name := args.Name
	err := script.LoadScripts(env.Config().ScriptPath + "/" + name)
	if err != nil {
		log.Errorf("load local script %s error: %v", name, err)
	}
}

func loadAllScripts() {
	path := env.Config().ScriptPath
	if err := script.LoadScripts(path); err != nil {
		log.Warnf("load scripts %s error: %v", path, err)
	}
}

func init() {
	cmd.Bind("FUNC_EffectScript", funcEffectLocalScript, (*scriptArgs)(nil)).SetPrivate()
	cmd.Bind("FUNC_EffectLocalScript", funcEffectLocalScript, (*scriptArgs)(nil)).SetPrivate()
	script.PreloadModule("game", externScript)
}
