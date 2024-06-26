package niuniu

import (
	"gofishing-game/service"

	"github.com/guogeer/quasar/v2/cmd"
	"github.com/guogeer/quasar/v2/utils"
)

type niuniuArgs struct {
	Ans      bool
	Times    int
	TriCards [3]int
}

func init() {
	cmd.BindFunc(ChooseTriCards, (*niuniuArgs)(nil), cmd.WithServer((*NiuNiuWorld)(nil).GetName()))
	cmd.BindFunc(Bet, (*niuniuArgs)(nil), cmd.WithServer((*NiuNiuWorld)(nil).GetName()))
	cmd.BindFunc(ChooseDealer, (*niuniuArgs)(nil), cmd.WithServer((*NiuNiuWorld)(nil).GetName()))
	cmd.BindFunc(DoubleAndRob, (*niuniuArgs)(nil), cmd.WithServer((*NiuNiuWorld)(nil).GetName()))
	cmd.BindFunc(SitDown, (*niuniuArgs)(nil), cmd.WithServer((*NiuNiuWorld)(nil).GetName()))
	cmd.BindFunc(EndGame, (*niuniuArgs)(nil), cmd.WithServer((*NiuNiuWorld)(nil).GetName()))
	cmd.BindFunc(SetAutoPlay, (*niuniuArgs)(nil), cmd.WithServer((*NiuNiuWorld)(nil).GetName()))
}

func GetPlayerByContext(ctx *cmd.Context) *NiuNiuPlayer {
	if p := service.GetGatewayPlayer(ctx.Ssid); p != nil {
		return p.GameAction.(*NiuNiuPlayer)
	}
	return nil
}

func ChooseTriCards(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*niuniuArgs)
	ply := GetPlayerByContext(ctx)
	if ply == nil {

		return
	}
	ply.ChooseTriCards(args.TriCards)
}

func Bet(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*niuniuArgs)
	ply := GetPlayerByContext(ctx)
	if ply == nil {

		return
	}
	ply.Bet(args.Times)
}

func ChooseDealer(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*niuniuArgs)
	ply := GetPlayerByContext(ctx)
	if ply == nil {

		return
	}
	ply.ChooseDealer(args.Ans)
}

func DoubleAndRob(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*niuniuArgs)
	ply := GetPlayerByContext(ctx)
	if ply == nil {

		return
	}
	ply.DoubleAndRob(args.Times)
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

func EndGame(ctx *cmd.Context, iArgs interface{}) {
	// args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {

		return
	}
	ply.EndGame()
}

func SetAutoPlay(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*niuniuArgs)
	ply := GetPlayerByContext(ctx)
	if ply == nil {

		return
	}
	chips := ply.Chips()
	if args.Times != 0 && utils.InArray(chips, args.Times) == 0 {
		return
	}
	ply.autoTimes = args.Times
	ply.WriteJSON("setAutoPlay", map[string]any{"times": ply.autoTimes})
}
