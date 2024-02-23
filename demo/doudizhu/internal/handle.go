package internal

import (
	"gofishing-game/service"

	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/log"
)

type Args struct {
	Cards  []int
	Choice int
	Type   int
}

func init() {
	cmd.BindFunc(Discard, (*Args)(nil))
	cmd.BindFunc(Pass, (*Args)(nil))
	cmd.BindFunc(Jiaodizhu, (*Args)(nil))
	cmd.BindFunc(Jiaofen, (*Args)(nil))
	cmd.BindFunc(Qiangdizhu, (*Args)(nil))
	cmd.BindFunc(AutoPlay, (*Args)(nil))
}

func GetPlayerByContext(ctx *cmd.Context) *DoudizhuPlayer {
	if p := service.GetGatewayPlayer(ctx.Ssid); p != nil {
		return p.GameAction.(*DoudizhuPlayer)
	}
	return nil
}

func Discard(ctx *cmd.Context, iArgs any) {
	args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		log.Debug("player is nil")
		return
	}
	ply.Discard(args.Cards)
}

func Pass(ctx *cmd.Context, iArgs any) {
	// args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		log.Debug("player is nil")
		return
	}
	ply.Pass()
}

func Jiaodizhu(ctx *cmd.Context, iArgs any) {
	args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		log.Debug("player is nil")
		return
	}
	ply.Jiaodizhu(args.Choice)
}

func Jiaofen(ctx *cmd.Context, iArgs any) {
	args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		log.Debug("player is nil")
		return
	}
	ply.Jiaofen(args.Choice)
}

func Qiangdizhu(ctx *cmd.Context, iArgs any) {
	args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		log.Debug("player is nil")
		return
	}
	ply.Qiangdizhu(args.Choice)
}

func AutoPlay(ctx *cmd.Context, iArgs any) {
	args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		log.Debug("player is nil")
		return
	}
	ply.SetAutoPlay(args.Type)
}
