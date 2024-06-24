package internal

import (
	"gofishing-game/service"

	"github.com/guogeer/quasar/v2/cmd"
)

type Args struct {
	Cards  []int `json:"cards,omitempty"`
	Choice int   `json:"choice,omitempty"`
	Type   int   `json:"type,omitempty"`
}

func init() {
	cmd.BindFunc(Discard, (*Args)(nil), cmd.WithServer((*DoudizhuWorld)(nil).GetName()))
	cmd.BindFunc(Pass, (*Args)(nil), cmd.WithServer((*DoudizhuWorld)(nil).GetName()))
	cmd.BindFunc(Jiaodizhu, (*Args)(nil), cmd.WithServer((*DoudizhuWorld)(nil).GetName()))
	cmd.BindFunc(Jiaofen, (*Args)(nil), cmd.WithServer((*DoudizhuWorld)(nil).GetName()))
	cmd.BindFunc(Qiangdizhu, (*Args)(nil), cmd.WithServer((*DoudizhuWorld)(nil).GetName()))
	cmd.BindFunc(AutoPlay, (*Args)(nil), cmd.WithServer((*DoudizhuWorld)(nil).GetName()))
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

		return
	}
	ply.Discard(args.Cards)
}

func Pass(ctx *cmd.Context, iArgs any) {
	// args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {

		return
	}
	ply.Pass()
}

func Jiaodizhu(ctx *cmd.Context, iArgs any) {
	args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {

		return
	}
	ply.Jiaodizhu(args.Choice)
}

func Jiaofen(ctx *cmd.Context, iArgs any) {
	args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {

		return
	}
	ply.Jiaofen(args.Choice)
}

func Qiangdizhu(ctx *cmd.Context, iArgs any) {
	args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {

		return
	}
	ply.Qiangdizhu(args.Choice)
}

func AutoPlay(ctx *cmd.Context, iArgs any) {
	args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {

		return
	}
	ply.SetAutoPlay(args.Type)
}
