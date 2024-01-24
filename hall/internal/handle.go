package internal

import (
	"context"
	"strconv"
	"strings"

	"gofishing-game/internal/errcode"
	"gofishing-game/internal/pb"
	"gofishing-game/internal/rpc"
	"gofishing-game/service"

	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/log"
)

type hallArgs struct {
	Id int `json:"id,omitempty"`

	Plate    string                      `json:"plate,omitempty"`
	UId      int                         `json:"uId,omitempty"`
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
	Games []service.ClientOnline `json:"games,omitempty"`
}

func init() {
	// internal call
	cmd.BindFunc(FUNC_UpdateOnline, (*updateOnlineArgs)(nil)).SetPrivate()
	cmd.BindFunc(FUNC_SyncOnline, (*hallArgs)(nil)).SetPrivate()
	cmd.BindFunc(S2C_GetBestGateway, (*hallArgs)(nil)).SetPrivate()

	cmd.Bind("FUNC_DeleteAccount", syncDeleteAccount, (*hallArgs)(nil)).SetPrivate()
	cmd.Bind("FUNC_UpdateMaintain", funcUpdateMaintain, (*hallArgs)(nil)).SetPrivate()
	cmd.Bind("FUNC_UpdateFakeOnline", funcUpdateFakeOnline, (*hallArgs)(nil)).SetPrivate()
}

// 更新在线人数
func FUNC_UpdateOnline(ctx *cmd.Context, data any) {
	w := GetWorld()
	args := data.(*updateOnlineArgs)
	for _, g := range args.Games {
		keys := strings.Split(g.ServerName+":", ":")
		subId, _ := strconv.Atoi(keys[1])
		w.onlines[subId] = service.ClientOnline{ServerName: keys[0], Online: g.Online}
	}
}

// 后台获取用户实时的在线数据
func FUNC_SyncOnline(ctx *cmd.Context, data any) {
	// req := data.(*service.Args)
	online := GetWorld().GetCurrentOnline()
	ctx.Out.WriteJSON("FUNC_SyncOnline", online)
}

func S2C_GetBestGateway(ctx *cmd.Context, data any) {
	args := data.(*hallArgs)
	w := GetWorld()
	w.currentBestGateway = args.Address
}

// 同步删除账号
func syncDeleteAccount(ctx *cmd.Context, data any) {
	args := data.(*hallArgs)
	log.Debug("gm delete account", args.UId)
	rpc.CacheClient().ClearAccount(context.Background(), &pb.ClearAccountReq{
		Uid: int32(args.UId),
	})
	ctx.Out.WriteJSON("FUNC_DeleteAccount", errcode.Ok)
}

func funcUpdateMaintain(ctx *cmd.Context, data any) {
	GetWorld().updateMaintain()
}

func funcUpdateFakeOnline(ctx *cmd.Context, data any) {
	args := data.(*hallArgs)
	GetWorld().segments = args.Segments
}
