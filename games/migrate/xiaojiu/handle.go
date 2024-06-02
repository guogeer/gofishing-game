package xiaojiu

import (
	"gofishing-game/service"

	"github.com/guogeer/quasar/cmd"
)

func init() {
	cmd.BindFunc(Bet, (*xiaojiuArgs)(nil), cmd.WithServer((*XiaojiuWorld)(nil).GetName()))
}

type xiaojiuArgs struct {
	AreaId    int   `json:"areaId,omitempty"`
	SeatIndex int   `json:"seatIndex,omitempty"`
	Gold      int64 `json:"gold,omitempty"`
}

func GetPlayerByContext(ctx *cmd.Context) *XiaojiuPlayer {
	if p := service.GetGatewayPlayer(ctx.Ssid); p != nil {
		return p.GameAction.(*XiaojiuPlayer)
	}
	return nil
}

func Bet(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*xiaojiuArgs)
	ply := GetPlayerByContext(ctx)
	if ply == nil {

		return
	}
	ply.Bet(args.AreaId, args.Gold)
}
