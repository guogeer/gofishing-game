package threedice

import (
	"gofishing-game/service"

	"github.com/guogeer/quasar/cmd"
)

func init() {
	cmd.BindFunc(Bet, (*Args)(nil))
	cmd.BindFunc(RobDealer, (*Args)(nil))
	cmd.BindFunc(GetHistory, (*Args)(nil))
	cmd.BindFunc(SitDown, (*Args)(nil))
}

type Args struct {
	AreaId int
	SeatId int
	Gold   int64
}

func GetPlayerByContext(ctx *cmd.Context) *ThreeDicePlayer {
	if p := service.GetGatewayPlayer(ctx.Ssid); p != nil {
		return p.GameAction.(*ThreeDicePlayer)
	}
	return nil
}

func Bet(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {

		return
	}
	ply.Bet(args.AreaId, args.Gold)
}

func RobDealer(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {

		return
	}
	ply.RobDealer(args.Gold)
}

func GetHistory(ctx *cmd.Context, iArgs interface{}) {
	// args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {

		return
	}
	room := ply.Room()
	ply.WriteJSON("GetHistory", map[string]any{"Last": room.GetLastHistory(16)})
}

func SitDown(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {

		return
	}
	ply.SitDown(args.SeatId)
}
