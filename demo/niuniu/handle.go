package niuniu

import (
	"gofishing-game/service"

	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/log"
	"github.com/guogeer/quasar/util"
)

func init() {
	cmd.Bind(ChooseTriCards, (*Args)(nil))
	cmd.Bind(Bet, (*Args)(nil))
	cmd.Bind(ChooseDealer, (*Args)(nil))
	cmd.Bind(DoubleAndRob, (*Args)(nil))
	cmd.Bind(SitDown, (*Args)(nil))
	cmd.Bind(EndGame, (*Args)(nil))
	cmd.Bind(SetAutoPlay, (*Args)(nil))
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
	if args.Times != 0 && util.InArray(chips, args.Times) == 0 {
		return
	}
	ply.autoTimes = args.Times
	ply.WriteJSON("SetAutoPlay", map[string]any{"Times": ply.autoTimes})
}
