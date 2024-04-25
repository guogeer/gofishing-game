package niuniu

import (
	"gofishing-game/service"

	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/log"
	"github.com/guogeer/quasar/utils"
)

func init() {
	cmd.BindFunc(ChooseTriCards, (*Args)(nil))
	cmd.BindFunc(Bet, (*Args)(nil))
	cmd.BindFunc(ChooseDealer, (*Args)(nil))
	cmd.BindFunc(DoubleAndRob, (*Args)(nil))
	cmd.BindFunc(SitDown, (*Args)(nil))
	cmd.BindFunc(EndGame, (*Args)(nil))
	cmd.BindFunc(SetAutoPlay, (*Args)(nil))
}

type Args struct {
	Ans      bool
	Times    int
	TriCards [3]int
}

func GetPlayerByContext(ctx *cmd.Context) *NiuNiuPlayer {
	if p := service.GetGatewayPlayer(ctx.Ssid); p != nil {
		return p.GameAction.(*NiuNiuPlayer)
	}
	return nil
}

func ChooseTriCards(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		log.Debug("player is nil")
		return
	}
	ply.ChooseTriCards(args.TriCards)
}

func Bet(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		log.Debug("player is nil")
		return
	}
	ply.Bet(args.Times)
}

func ChooseDealer(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		log.Debug("player is nil")
		return
	}
	ply.ChooseDealer(args.Ans)
}

func DoubleAndRob(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		log.Debug("player is nil")
		return
	}
	ply.DoubleAndRob(args.Times)
}

func SitDown(ctx *cmd.Context, iArgs interface{}) {
	// args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		log.Debug("player is nil")
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
		log.Debug("player is nil")
		return
	}
	ply.EndGame()
}

func SetAutoPlay(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		log.Debug("player is nil")
		return
	}
	chips := ply.Chips()
	if args.Times != 0 && utils.InArray(chips, args.Times) == 0 {
		return
	}
	ply.autoTimes = args.Times
	ply.WriteJSON("SetAutoPlay", map[string]any{"Times": ply.autoTimes})
}
