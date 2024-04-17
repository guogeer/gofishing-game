package dice

import (
	"gofishing-game/service"

	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/log"
)

func init() {
	cmd.BindFunc(Bet, (*diceArgs)(nil))
	cmd.BindFunc(GetLastHistory, (*diceArgs)(nil))
}

type diceArgs struct {
	Area    int   `json:"area,omitempty"`
	Gold    int64 `json:"gold,omitempty"`
	PageNum int   `json:"pageNum,omitempty"`
}

func GetPlayerByContext(ctx *cmd.Context) *DicePlayer {
	if ply := service.GetGatewayPlayer(ctx.Ssid); ply != nil {
		return ply.GameAction.(*DicePlayer)
	}
	return nil
}

func Bet(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*diceArgs)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		log.Debug("player is nil")
		return
	}
	e := ply.Bet(args.Area, args.Gold)
	ply.WriteErr("bet", e, "area", args.Area, "gold", args.Gold)
}

func GetLastHistory(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*diceArgs)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	room := ply.Room()
	if room == nil {
		return
	}
	ply.WriteJSON("getLastHistory", map[string]any{"last": room.GetLastHistory(args.PageNum)})
}
