package service

import (
	"gofishing-game/internal/gameutils"

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

func init() {
	cmd.Hook(hook)

	cmd.Bind("FUNC_EffectTableOk", funcEffectTableOk, (*serviceArgs)(nil))
	cmd.Bind("FUNC_AddItems", funcAddItems, (*serviceArgs)(nil))

	cmd.Bind("Leave", funcLeave, (*serviceArgs)(nil))
	cmd.Bind("Close", funcClose, (*serviceArgs)(nil))
}

func GetPlayerByContext(ctx *cmd.Context) *Player {
	return GetGatewayPlayer(ctx.Ssid)
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
		WriteMessage(ss, "", "serverClose", cmd.M{"serverName": ctx.ServerName, "cause": "not found player"})

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
	args := data.(*serviceArgs)

	var items []gameutils.Item
	for _, item := range args.Items {
		items = append(items, item)
	}
	AddItems(args.Uid, items, args.Way)
}
