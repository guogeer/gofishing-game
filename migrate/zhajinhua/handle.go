package zhajinhua

import (
	"gofishing-game/service"

	"github.com/guogeer/quasar/cmd"
)

func init() {
	cmd.BindFunc(TakeAction, (*zhajinhuaArgs)(nil))
	cmd.BindFunc(SitUp, (*zhajinhuaArgs)(nil))       // 站起
	cmd.BindFunc(LookCard, (*zhajinhuaArgs)(nil))    // 亮牌
	cmd.BindFunc(ShowCard, (*zhajinhuaArgs)(nil))    // 亮牌
	cmd.BindFunc(CompareCard, (*zhajinhuaArgs)(nil)) // 比牌
	cmd.BindFunc(SetAutoPlay, (*zhajinhuaArgs)(nil)) // 托管
}

type zhajinhuaArgs struct {
	SeatId int   `json:"seatId,omitempty"`
	Gold   int64 `json:"gold,omitempty"`
	Auto   int   `json:"auto,omitempty"`
}

func GetPlayerByContext(ctx *cmd.Context) *ZhajinhuaPlayer {
	if p := service.GetGatewayPlayer(ctx.Ssid); p != nil {
		return p.GameAction.(*ZhajinhuaPlayer)
	}
	return nil
}

func TakeAction(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*zhajinhuaArgs)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	ply.TakeAction(args.Gold)
}

func SitUp(ctx *cmd.Context, iArgs interface{}) {
	// args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	ply.SitUp()
}

func LookCard(ctx *cmd.Context, iArgs interface{}) {
	// args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	ply.LookCard()
}

func ShowCard(ctx *cmd.Context, iArgs interface{}) {
	// args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	ply.ShowCard()
}

func CompareCard(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*zhajinhuaArgs)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	ply.CompareCard(args.SeatId)
}

func SetAutoPlay(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*zhajinhuaArgs)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	ply.auto = args.Auto
	ply.AutoPlay()
	ply.WriteJSON("SetAutoPlay", map[string]any{"UId": ply.Id, "Auto": args.Auto})
}
