package shengsidu

import (
	"gofishing-game/service"

	"github.com/guogeer/quasar/cmd"
)

type Args struct {
	Cards []int
}

func init() {
	cmd.Bind(Discard, (*Args)(nil))
	cmd.Bind(Pass, (*Args)(nil))
}

func GetPlayerByContext(ctx *cmd.Context) *ShengsiduPlayer {
	if p := service.GetGatewayPlayer(ctx.Ssid); p != nil {
		return p.GameAction.(*ShengsiduPlayer)
	}
	return nil
}

func Discard(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {

		return
	}
	ply.Discard(args.Cards)
}

func Pass(ctx *cmd.Context, iArgs interface{}) {
	// args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {

		return
	}
	ply.Pass()
}
