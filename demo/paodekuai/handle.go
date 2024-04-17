package paodekuai

import (
	"gofishing-game/service"

	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/log"
)

type Args struct {
	Cards []int
}

func init() {
	cmd.Bind(Discard, (*Args)(nil))
	cmd.Bind(Pass, (*Args)(nil))
}

func GetPlayerByContext(ctx *cmd.Context) *PaodekuaiPlayer {
	if p := service.GetGatewayPlayer(ctx.Ssid); p != nil {
		return p.GameAction.(*PaodekuaiPlayer)
	}
	return nil
}

func Discard(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		log.Debug("player is nil")
		return
	}
	ply.Discard(args.Cards)
}

func Pass(ctx *cmd.Context, iArgs interface{}) {
	// args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		log.Debug("player is nil")
		return
	}
	ply.Pass()
}
