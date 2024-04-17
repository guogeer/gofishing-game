package zhajinhua

import (
	"gofishing-game/service"

	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/log"
)

func init() {
	cmd.Bind(TakeAction, (*Args)(nil))
	cmd.Bind(SitDown, (*Args)(nil))
	cmd.Bind(SitUp, (*Args)(nil))       // 站起
	cmd.Bind(LookCard, (*Args)(nil))    // 亮牌
	cmd.Bind(ShowCard, (*Args)(nil))    // 亮牌
	cmd.Bind(CompareCard, (*Args)(nil)) // 比牌
	cmd.Bind(SetAutoPlay, (*Args)(nil)) // 托管
}

type Args struct {
	SeatId int
	Gold   int64
	Auto   int
}

func GetPlayerByContext(ctx *cmd.Context) *ZhajinhuaPlayer {
	if p := service.GetGatewayPlayer(ctx.Ssid); p != nil {
		return p.GameAction.(*ZhajinhuaPlayer)
	}
	return nil
}

func TakeAction(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		log.Debug("player is nil")
		return
	}
	ply.TakeAction(args.Gold)
}

func SitDown(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		log.Debug("player is nil")
		return
	}
	ply.SitDown(args.SeatId)
}

func SitUp(ctx *cmd.Context, iArgs interface{}) {
	// args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		log.Debug("player is nil")
		return
	}
	ply.SitUp()
}

func LookCard(ctx *cmd.Context, iArgs interface{}) {
	// args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		log.Debug("player is nil")
		return
	}
	ply.LookCard()
}

func ShowCard(ctx *cmd.Context, iArgs interface{}) {
	// args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		log.Debug("player is nil")
		return
	}
	ply.ShowCard()
}

func CompareCard(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		log.Debug("player is nil")
		return
	}
	ply.CompareCard(args.SeatId)
}

func SetAutoPlay(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		log.Debug("player is nil")
		return
	}
	ply.auto = args.Auto
	ply.AutoPlay()
	ply.WriteJSON("SetAutoPlay", map[string]any{"UId": ply.Id, "Auto": args.Auto})
}
