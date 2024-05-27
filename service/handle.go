package service

import (
	"encoding/json"
	"gofishing-game/internal/gameutils"

	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/log"
)

type addItemsArgs struct {
	Uid   int                      `json:"uid,omitempty"`
	Items []*gameutils.NumericItem `json:"items,omitempty"`
	Way   string                   `json:"way,omitempty"`
}

type leaveArgs struct {
	Uid int `json:"uid,omitempty"`
}

func init() {
	cmd.Hook(hook)

	cmd.Bind("func_effectTableOk", funcEffectTableOk, nil)
	cmd.Bind("func_addItems", funcAddItems, (*addItemsArgs)(nil))

	cmd.Bind("Leave", funcLeave, nil)
	cmd.Bind("Close", funcClose, nil)

	cmd.Bind("func_leave", funcSysLeave, (*leaveArgs)(nil)).SetPrivate()
	cmd.Bind("Enter", funcEnter, (*json.RawMessage)(nil))
}

type msgHandler interface {
	OnRecvMsg()
}

func hook(ctx *cmd.Context, data any) {
	// 非客户端发过来的消息或进入游戏
	if ctx.ClientAddr == "" || ctx.MsgId == "enter" {
		return
	}

	ply := GetPlayerByContext(ctx)
	if ply == nil {
		ss := &cmd.Session{Id: ctx.Ssid, Out: ctx.Out}
		WriteMessage(ss, "serverClose", cmd.M{"serverName": ctx.ServerName, "cause": "not found player"})

		ctx.Fail()
	} else {
		if mh, ok := ply.GameAction.(msgHandler); ok {
			mh.OnRecvMsg()
		}
	}
}

func funcClose(ctx *cmd.Context, iArgs any) {
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	ply.GameAction.OnClose()
}

func funcLeave(ctx *cmd.Context, iArgs any) {
	ply := GetGatewayPlayer(ctx.Ssid)
	if ply == nil {
		return
	}
	log.Debugf("player %d leave room", ply.Id)
	ply.Leave()
}

func funcEffectTableOk(ctx *cmd.Context, data any) {
}

func funcAddItems(ctx *cmd.Context, data any) {
	args := data.(*addItemsArgs)

	var items []gameutils.Item
	for _, item := range args.Items {
		items = append(items, item)
	}
	AddItems(args.Uid, items, args.Way)
}

func funcEnter(ctx *cmd.Context, data any) {
	rawData := *(data.(*json.RawMessage))

	args := &enterArgs{}
	json.Unmarshal(rawData, args)

	if args.Token == "" {
		return
	}
	if args.LeaveServer == GetServerId() {
		args.LeaveServer = ""
	}

	ss, e := GetEnterQueue().PushBack(ctx, args.Token, args.LeaveServer, rawData)
	if e != nil && ss != nil {
		WriteMessage(ss, "enter", e)
	}
}

type enterArgs struct {
	Token       string `json:"token,omitempty"`
	LeaveServer string `json:"leaveServer,omitempty"`
}

func funcSysLeave(ctx *cmd.Context, data any) {
	args := data.(*leaveArgs)

	uid := args.Uid
	ply := GetPlayer(uid)
	log.Debugf("player %d auto leave", uid)

	if ply == nil {
		ctx.Out.WriteJSON("func_leave", cmd.M{"uid": uid})
	} else {
		ply.Leave2(ctx, nil)
	}
}
