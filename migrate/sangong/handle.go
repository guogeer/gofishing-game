package sangong

import (
	"gofishing-game/service"

	"github.com/guogeer/quasar/cmd"
)

func init() {
	cmd.Bind(Finish, (*Args)(nil))
	cmd.Bind(Bet, (*Args)(nil))
	cmd.Bind(ChooseDealer, (*Args)(nil))
	cmd.Bind(SitDown, (*Args)(nil))
}

type Args struct {
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
	args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {

		return
	}
	ply.Bet(args.Chip)
}

func ChooseDealer(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {

		return
	}
	ply.ChooseDealer(args.Ans)
}

func SitDown(ctx *cmd.Context, iArgs interface{}) {
	// args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {

		return
	}
	room := ply.Room()
	seatId := room.GetEmptySeat()
	ply.SitDown(seatId)
}
