package service

import (
	"encoding/json"
	"gofishing-game/internal/gameutils"
	"gofishing-game/service"

	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/log"
)

type serviceArgs struct {
	Id      int    `json:"id,omitempty"`
	Uid     int    `json:"uid,omitempty"`
	Answer  int    `json:"answer,omitempty"`
	OrderId string `json:"orderId,omitempty"`
	Way     string `json:"way,omitempty"`
	Msg     string `json:"msg,omitempty"`
	Type    int    `json:"type,omitempty"`
	Phone   string `json:"phone,omitempty"`
	SubId   int    `json:"subId,omitempty"`
	RoomId  int    `json:"roomId,omitempty"`
	IP      string `json:"ip,omitempty"`
	Sample  string `json:"sample,omitempty"`
	ItemId  int    `json:"itemId,omitempty"`
	ItemNum int64  `json:"itemNum,omitempty"`

	Name  string                   `json:"name,omitempty"`
	Items []*gameutils.NumericItem `json:"items,omitempty"`
}

type authArgs struct {
	Uid         int    `json:"uid,omitempty"`
	Token       string `json:"token,omitempty"`
	LeaveServer string `json:"leaveServer,omitempty"`
}

type enterArgs struct {
	authArgs

	data []byte `json:"-"`
}

func (args *enterArgs) UnmarshalJSON(buf []byte) error {
	args.data = buf
	return json.Unmarshal(buf, &args.authArgs)
}

type addItemsArgs struct {
	Uid   int                      `json:"uid,omitempty"`
	Items []*gameutils.NumericItem `json:"items,omitempty"`
	Way   string                   `json:"way,omitempty"`
}

func init() {
	cmd.Hook(hook)

	cmd.BindFunc(FUNC_EffectTableOk, nil)
	cmd.BindFunc(FUNC_AddItems, (*addItemsArgs)(nil))

	cmd.BindFunc(Leave, nil)
	cmd.BindFunc(Close, nil)

	cmd.BindFunc(FUNC_Leave, (*enterArgs)(nil)).SetPrivate()
	cmd.BindFunc(Enter, (*enterArgs)(nil))
}

type msgHandler interface {
	OnRecvMsg()
}

func hook(ctx *cmd.Context, data any) {
	// 非客户端发过来的消息或进入游戏
	if ctx.ClientAddr == "" || ctx.MsgId == "enter" {
		return
	}

	ply := service.GetPlayerByContext(ctx)
	if ply == nil {
		ss := &cmd.Session{Id: ctx.Ssid, Out: ctx.Out}
		service.WriteMessage(ss, "serverClose", cmd.M{"serverName": ctx.ServerName, "cause": "not found player"})

		ctx.Fail()
	} else {
		if mh, ok := ply.GameAction.(msgHandler); ok {
			mh.OnRecvMsg()
		}
	}
}

func Close(ctx *cmd.Context, iArgs any) {
	ply := service.GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	ply.GameAction.OnClose()
}

func Leave(ctx *cmd.Context, iArgs any) {
	ply := service.GetGatewayPlayer(ctx.Ssid)
	if ply == nil {
		return
	}
	log.Debugf("player %d leave room", ply.Id)
	ply.Leave()
}

func FUNC_EffectTableOk(ctx *cmd.Context, data any) {
}

func FUNC_AddItems(ctx *cmd.Context, data any) {
	args := data.(*addItemsArgs)

	var items []gameutils.Item
	for _, item := range args.Items {
		items = append(items, item)
	}
	service.AddItems(args.Uid, items, args.Way)
}

func Enter(ctx *cmd.Context, data any) {
	args := data.(*enterArgs)
	if args.Token == "" {
		return
	}
	// 忽略同一个链接登陆不同的账号
	ply := service.GetGatewayPlayer(ctx.Ssid)
	if ply != nil && ply.Id != args.Uid {
		return
	}

	ss, e := service.GetEnterQueue().PushBack(ctx, args.Uid, args.Token, args.LeaveServer, args.data)
	if e != nil && ss != nil {
		service.WriteMessage(ss, "enter", e)
	}
	log.Infof("player %d enter %s user+robot num %d", args.Uid, ctx.ServerName, len(service.GetAllPlayers()))
}

func FUNC_Leave(ctx *cmd.Context, data any) {
	args := data.(*enterArgs)
	uid := args.Uid
	ply := service.GetPlayer(uid)
	// log.Debugf("player %d auto leave", uid)

	if ply == nil {
		ctx.Out.WriteJSON("FUNC_Leave", cmd.M{"uid": uid})
	} else {
		ply.Leave2(ctx, nil)
	}
}
