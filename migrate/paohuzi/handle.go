package paohuzi

import (
	"gofishing-game/service"

	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/log"
)

type Args struct {
	Card      int
	ChowCards [][3]int
}

func init() {
	cmd.Bind(Discard, (*Args)(nil))
	cmd.Bind(Pass, (*Args)(nil))
	cmd.Bind(Pong, (*Args)(nil))
}

func GetPlayerByContext(ctx *cmd.Context) *PaohuziPlayer {
	if p := service.GetGatewayPlayer(ctx.Ssid); p != nil {
		return p.GameAction.(*PaohuziPlayer)
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
	ply.Discard(args.Card)
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

func Pong(ctx *cmd.Context, iArgs interface{}) {
	// args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		log.Debug("player is nil")
		return
	}
	ply.Pong()
}

func Win(ctx *cmd.Context, iArgs interface{}) {
	// args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		log.Debug("player is nil")
		return
	}
	ply.Win()
}
