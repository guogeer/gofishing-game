package hall

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

type Args struct {
	Id int

	Plate    string
	UId      int
	Address  string
	Version  string
	Segments []service.GameOnlineSegment
}

func GetPlayerByContext(ctx *cmd.Context) *hallPlayer {
	if p := service.GetGatewayPlayer(ctx.Ssid); p != nil {
		return p.GameAction.(*hallPlayer)
	}
	return nil
}

type updateOnlineArgs struct {
	Games []service.ClientOnline
}

func init() {
	// internal call
	cmd.Bind(FUNC_UpdateOnline, (*updateOnlineArgs)(nil))
	cmd.Bind(FUNC_SyncOnline, (*Args)(nil))
	cmd.Bind(FUNC_UpdateClientVersion, (*Args)(nil))
	cmd.Bind(S2C_GetBestGateway, (*Args)(nil)) // 网关同步
	cmd.BindWithName("FUNC_DeleteAccount", syncDeleteAccount, (*Args)(nil))
	cmd.BindWithName("FUNC_UpdateMaintain", funcUpdateMaintain, (*Args)(nil))
	cmd.BindWithName("FUNC_UpdateFakeOnline", funcUpdateFakeOnline, (*Args)(nil))
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
	args := data.(*Args)
	w := GetWorld()
	w.currentBestGateway = args.Address
}

func FUNC_UpdateClientVersion(ctx *cmd.Context, data any) {
	GetWorld().updateClientVersion()
}

// 同步删除账号
func syncDeleteAccount(ctx *cmd.Context, data any) {
	args := data.(*Args)
	log.Debug("gm delete account", args.UId)
	rpc.CacheClient().ClearAccount(context.Background(), &pb.AccountInfo{
		UId: int32(args.UId),
	})
	ctx.Out.WriteJSON("FUNC_DeleteAccount", errcode.Ok)
}

func funcUpdateMaintain(ctx *cmd.Context, data any) {
	GetWorld().updateMaintain()
}

func funcUpdateFakeOnline(ctx *cmd.Context, data any) {
	args := data.(*Args)
	GetWorld().segments = args.Segments
}
