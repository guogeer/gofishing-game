package shuiguoji

import (
	"gofishing-game/service"

	"github.com/guogeer/quasar/cmd"
)

func init() {
	cmd.Bind(Bet, (*Args)(nil))
	cmd.Bind(LookPrizePool, (*Args)(nil))
}

type Args struct {
	Line int
	Gold int64
	Chip int64
}

func GetPlayerByContext(ctx *cmd.Context) *shuiguojiPlayer {
	if p := service.GetGatewayPlayer(ctx.Ssid); p != nil {
		return p.GameAction.(*shuiguojiPlayer)
	}
	return nil
}

func Bet(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	ply.Bet(args.Line, args.Chip)
}

func LookPrizePool(ctx *cmd.Context, iArgs interface{}) {
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	ply.WriteJSON("LookPrizePool", map[string]any{"Top": prizePoolRank.top})
}
