package fruit

import (
	"gofishing-game/service"

	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/log"
)

func init() {
	// handle json

	// cmd.Bind(Enter, (*Args)(nil))
	cmd.Bind(Bet, (*Args)(nil))
	cmd.Bind(GetHistory, (*Args)(nil))
	cmd.Bind(SitUp, (*Args)(nil))
	cmd.Bind(SitDown, (*Args)(nil))
}

type Args struct {
	UId     int
	SubId   int
	Type    int
	Info    string
	Ssid    string
	Msg     string
	Token   string
	Area    int
	Gold    int64
	PageNum int
	SeatId  int
}

func GetPlayerByContext(ctx *cmd.Context) *FruitPlayer {
	if ply := service.GetGatewayPlayer(ctx.Ssid); ply != nil {
		return ply.GameAction.(*FruitPlayer)
	}
	return nil
}

func Bet(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		log.Debug("player is nil")
		return
	}
	ply.fruitObj.Bet(args.Area, args.Gold)
}

func GetHistory(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		log.Debug("player is nil")
		return
	}
	n := args.PageNum
	if n < 1 {
		n = 20
	}
	ply.fruitObj.GetHistory(n)
}

func SitDown(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	ply.SitDown(args.SeatId)
}

func SitUp(ctx *cmd.Context, iArgs interface{}) {
	// args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	ply.RoomObj.SitUp()
}
