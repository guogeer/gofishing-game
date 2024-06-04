package sangong

import (
	"gofishing-game/service"

	"github.com/guogeer/quasar/v2/cmd"
)

func init() {
	cmd.BindFunc(Finish, (*sangongArgs)(nil), cmd.WithServer((*SangongWorld)(nil).GetName()))
	cmd.BindFunc(Bet, (*sangongArgs)(nil), cmd.WithServer((*SangongWorld)(nil).GetName()))
	cmd.BindFunc(ChooseDealer, (*sangongArgs)(nil), cmd.WithServer((*SangongWorld)(nil).GetName()))
}

type sangongArgs struct {
	Ans  bool
	Chip int
}

func GetPlayerByContext(ctx *cmd.Context) *SangongPlayer {
	if p := service.GetGatewayPlayer(ctx.Ssid); p != nil {
		return p.GameAction.(*SangongPlayer)
	}
	return nil
}

func Finish(ctx *cmd.Context, iArgs interface{}) {
	ply := GetPlayerByContext(ctx)
	if ply == nil {

		return
	}
	ply.Finish()
}

func Bet(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*sangongArgs)
	ply := GetPlayerByContext(ctx)
	if ply == nil {

		return
	}
	ply.Bet(args.Chip)
}

func ChooseDealer(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*sangongArgs)
	ply := GetPlayerByContext(ctx)
	if ply == nil {

		return
	}
	ply.ChooseDealer(args.Ans)
}
