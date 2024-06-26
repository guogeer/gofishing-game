package paodekuai

import (
	"gofishing-game/service"

	"github.com/guogeer/quasar/v2/cmd"
)

type paodekuaiArgs struct {
	Cards []int
}

func init() {
	cmd.BindFunc(Discard, (*paodekuaiArgs)(nil), cmd.WithServer((*PaodekuaiWorld)(nil).GetName()))
	cmd.BindFunc(Pass, (*paodekuaiArgs)(nil), cmd.WithServer((*PaodekuaiWorld)(nil).GetName()))
}

func GetPlayerByContext(ctx *cmd.Context) *PaodekuaiPlayer {
	if p := service.GetGatewayPlayer(ctx.Ssid); p != nil {
		return p.GameAction.(*PaodekuaiPlayer)
	}
	return nil
}

func Discard(ctx *cmd.Context, data interface{}) {
	args := data.(*paodekuaiArgs)
	ply := GetPlayerByContext(ctx)
	ply.Discard(args.Cards)
}

func Pass(ctx *cmd.Context, data interface{}) {
	// args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	ply.Pass()
}
