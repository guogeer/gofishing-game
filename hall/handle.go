package main

import (
	"context"

	"gofishing-game/internal/pb"
	"gofishing-game/internal/rpc"
	"gofishing-game/service"

	"github.com/guogeer/quasar/v2/cmd"
	"github.com/guogeer/quasar/v2/log"
)

type hallArgs struct {
	Id       int                         `json:"id,omitempty"`
	Plate    string                      `json:"plate,omitempty"`
	Uid      int                         `json:"uid,omitempty"`
	Address  string                      `json:"address,omitempty"`
	Version  string                      `json:"version,omitempty"`
	Segments []service.GameOnlineSegment `json:"segments,omitempty"`
}

func GetPlayerByContext(ctx *cmd.Context) *hallPlayer {
	if p := service.GetGatewayPlayer(ctx.Ssid); p != nil {
		return p.GameAction.(*hallPlayer)
	}
	return nil
}

type updateOnlineArgs struct {
	Games []service.ServerOnline `json:"games,omitempty"`
}

func init() {
	// internal call
	cmd.BindFunc(FUNC_UpdateOnline, (*updateOnlineArgs)(nil), cmd.WithPrivate())
	cmd.BindFunc(FUNC_SyncOnline, (*hallArgs)(nil), cmd.WithPrivate())
	cmd.BindFunc(S2C_GetBestGateway, (*hallArgs)(nil), cmd.WithPrivate())

	cmd.BindFunc(FUNC_DeleteAccount, (*hallArgs)(nil), cmd.WithPrivate(), cmd.WithoutQueue())
	cmd.BindFunc(FUNC_UpdateMaintain, (*hallArgs)(nil), cmd.WithPrivate())
	cmd.BindFunc(FUNC_UpdateFakeOnline, (*hallArgs)(nil), cmd.WithPrivate())
}

// 更新在线人数
func FUNC_UpdateOnline(ctx *cmd.Context, data any) {
	w := GetWorld()
	args := data.(*updateOnlineArgs)
	for _, g := range args.Games {
		w.onlines[g.Id] = g
	}
}

// 后台获取用户实时的在线数据
func FUNC_SyncOnline(ctx *cmd.Context, data any) {
	// req := data.(*service.Args)
	online := GetWorld().GetCurrentOnline()
	ctx.Out.WriteJSON("func_syncOnline", online)
}

func S2C_GetBestGateway(ctx *cmd.Context, data any) {
	args := data.(*hallArgs)
	w := GetWorld()
	w.currentBestGateway = args.Address
}

// 同步删除账号
func FUNC_DeleteAccount(ctx *cmd.Context, data any) {
	args := data.(*hallArgs)
	log.Debug("gm delete account", args.Uid)
	rpc.CacheClient().ClearAccount(context.Background(), &pb.ClearAccountReq{
		Uid: int32(args.Uid),
	})
	ctx.Out.WriteJSON("func_deleteAccount", struct{}{})
}

func FUNC_UpdateMaintain(ctx *cmd.Context, data any) {
	GetWorld().updateMaintain()
}

func FUNC_UpdateFakeOnline(ctx *cmd.Context, data any) {
	args := data.(*hallArgs)
	GetWorld().segments = args.Segments
}
