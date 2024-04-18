package internal

import (
	"gofishing-game/service"

	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
)

type Args struct {
	UId     int
	Type    int
	Message string
	Token   string
	Area    int
	Gold    int64
	PageNum int
	SeatId  int
}

type Config struct {
}

func init() {
	// handle json

	cmd.Bind(Bet, (*Args)(nil))
	cmd.Bind(GetLastHistory, (*Args)(nil))
	cmd.Bind(SitUp, (*Args)(nil))
	cmd.Bind(SitDown, (*Args)(nil))
	// cmd.Bind(Chat, (*Args)(nil))
	cmd.Bind(ApplyDealer, (*Args)(nil))
	cmd.Bind(CancelDealer, (*Args)(nil))
	cmd.Bind(GetRank, (*Args)(nil))
	cmd.Bind(GetDealerQueue, (*Args)(nil))
	cmd.Bind(ChangeDealerGold, (*Args)(nil))

	cmd.Bind(Console_WhosYourDaddy, (*Args)(nil))
}

func GetPlayerByContext(ctx *cmd.Context) *entertainmentPlayer {
	if ply := service.GetGatewayPlayer(ctx.Ssid); ply != nil {
		return ply.GameAction.(*entertainmentPlayer)
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
	ply.Bet(args.Area, args.Gold)
}

func GetLastHistory(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		log.Debug("player is nil")
		return
	}
	ply.GetLastHistory(args.PageNum)
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

func Chat(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	ply.Chat(args.Type, args.Message)
}

func ApplyDealer(ctx *cmd.Context, iArgs interface{}) {
	// args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	ply.ApplyDealer()
}

func CancelDealer(ctx *cmd.Context, iArgs interface{}) {
	// args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	ply.CancelDealer()
}

func GetRank(ctx *cmd.Context, iArgs interface{}) {
	// args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	todayGold := ply.BaseObj().Int64("daily." + service.GetName() + "_win_gold")
	prizePool := ply.Room().GetDayRank()
	ply.WriteJSON("GetRank", map[string]any{
		"Rank":      prizePool.Rank(),
		"LastRank":  prizePool.LastRank(),
		"TodayGold": todayGold,
	})
}

func GetDealerQueue(ctx *cmd.Context, iArgs interface{}) {
	// args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}

	ply.WriteJSON("GetDealerQueue", ply.dealerQueue())
}

func Console_WhosYourDaddy(ctx *cmd.Context, iArgs interface{}) {
	log.Debug("console whos you daddy")
	isNextTurnSystemControl = true
}

// 修改上庄金币
func ChangeDealerGold(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}

	e := NewError(Ok)
	room := ply.Room()
	minDealerGold, _ := config.Int("entertainment", room.GetSubId(), "MinDealerGold")
	if args.Gold < minDealerGold {
		e = NewError(Retry)
	}
	response := map[string]any{
		"Code": e.Code,
		"Msg":  e.Msg,
		"Gold": args.Gold,
	}
	ply.WriteJSON("ChangeDealerGold", response)
	if e.Code != Ok {
		return
	}
	ply.dealerLimitGold = args.Gold
}
