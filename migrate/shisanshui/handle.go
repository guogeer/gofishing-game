package shisanshui

import (
	"gofishing-game/service"

	"github.com/guogeer/quasar/cmd"
)

type Args struct {
	Cards []int
}

func init() {
	cmd.Bind(SplitCards, (*Args)(nil))
}

func GetPlayerByContext(ctx *cmd.Context) *ShisanshuiPlayer {
	if p := service.GetGatewayPlayer(ctx.Ssid); p != nil {
		return p.GameAction.(*ShisanshuiPlayer)
	}
	return nil
}

func SplitCards(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	ply.SplitCards(args.Cards)
}
