package xiaojiu

import (
	"gofishing-game/service"

	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/log"
)

func init() {
	cmd.Bind(Bet, (*Args)(nil))
}

type Args struct {
	AreaId int
	SeatId int
	Gold   int64
}

func GetPlayerByContext(ctx *cmd.Context) *XiaojiuPlayer {
	if p := service.GetGatewayPlayer(ctx.Ssid); p != nil {
		return p.GameAction.(*XiaojiuPlayer)
	}
	return nil
}

func Bet(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		log.Debug("player is nil")
		return
	}
	ply.Bet(args.AreaId, args.Gold)
}
