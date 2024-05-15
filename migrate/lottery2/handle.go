package lottery

import (
	"gofishing-game/service"
	"gofishing-game/service/roomutils"

	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/log"
)

type Args struct {
	Area      int
	Area2     int
	SeatIndex int
	Gold      int64
	Type      int
}

func init() {
	cmd.Bind(Bet, (*Args)(nil))

	cmd.Bind(SitUp, (*Args)(nil))
	cmd.Bind(SitDown, (*Args)(nil))
	cmd.Bind(GetBetHistory, (*Args)(nil))
	cmd.Bind(Console_WhosYourDaddy, (*Args)(nil))
}

func GetPlayerByContext(ctx *cmd.Context) *lotteryPlayer {
	if p := service.GetGatewayPlayer(ctx.Ssid); p != nil {
		return p.GameAction.(*lotteryPlayer)
	}
	return nil
}

func Bet(ctx *cmd.Context, data interface{}) {
	args := data.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	area := args.Area + 2
	if args.Area2 > 0 {
		area = args.Area2
	}
	ply.Bet(area, args.Gold)
}

func SitDown(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	ply.SitDown(args.SeatIndex)
}

func SitUp(ctx *cmd.Context, iArgs interface{}) {
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	roomutils.GetRoomObj(ply.Player).SitUp()
}

func GetBetHistory(ctx *cmd.Context, iArgs interface{}) {
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	room := ply.Room()
	var last []UserRecord
	for e := room.awards.Front(); e != nil; e = e.Next() {
		awardData := e.Value.(*AwardRecord)
		userData := awardData.Users[ply.Id]
		userData.Ts = awardData.Ts
		userData.Type = awardData.Type
		last = append(last, userData)
	}
	ply.WriteJSON("GetBetHistory", map[string]any{"Last": last})
}

func Console_WhosYourDaddy(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*Args)
	log.Debug("console whos you daddy", args.Type)
	gNextTurnType = args.Type
}
