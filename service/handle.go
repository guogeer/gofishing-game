package service

import (
	"gofishing-game/internal/gameutils"

	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/log"
)

type serviceArgs struct {
	Id      int
	UId     int
	Answer  int
	OrderId string
	Guid    string
	Way     string
	Msg     string
	Type    int
	Phone   string
	SubId   int
	RoomId  int
	IP      string
	Sample  string
	ItemId  int
	ItemNum int64

	Name      string
	Items     []*gameutils.Item
	IsWatchAd bool
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
	if ctx.ClientAddr == "" || ctx.MsgId == "Enter" {
		return
	}

	ply := GetPlayerByContext(ctx)
	if ply == nil {
		ss := &cmd.Session{Id: ctx.Ssid, Out: ctx.Out}
		WriteMessage(ss, "", "ServerClose", cmd.M{"ServerName": ctx.ServerName})

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

	AddItems(args.UId, &gameutils.ItemLog{
		Kind:  "sys",
		Items: args.Items,
		Way:   args.Way,
	})
}
