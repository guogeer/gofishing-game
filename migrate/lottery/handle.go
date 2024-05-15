package lottery

import (
	"gofishing-game/internal/errcode"
	"gofishing-game/service"

	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
)

type lotteryArgs struct {
	Uid     int    `json:"uid,omitempty"`
	Type    int    `json:"type,omitempty"`
	Message string `json:"message,omitempty"`
	Token   string `json:"token,omitempty"`
	Area    int    `json:"area,omitempty"`
	Gold    int64  `json:"gold,omitempty"`
	PageNum int    `json:"pageNum,omitempty"`
	SeatId  int    `json:"seatIndex,omitempty"`
}

type Config struct {
}

func init() {
	cmd.BindFunc(Bet, (*lotteryArgs)(nil))
	cmd.BindFunc(GetLastHistory, (*lotteryArgs)(nil))
	cmd.BindFunc(ApplyDealer, (*lotteryArgs)(nil))
	cmd.BindFunc(CancelDealer, (*lotteryArgs)(nil))
	cmd.BindFunc(GetDealerQueue, (*lotteryArgs)(nil))
	cmd.BindFunc(ChangeDealerGold, (*lotteryArgs)(nil))

	cmd.BindFunc(Console_WhosYourDaddy, (*lotteryArgs)(nil))
}

func GetPlayerByContext(ctx *cmd.Context) *lotteryPlayer {
	if ply := service.GetGatewayPlayer(ctx.Ssid); ply != nil {
		return ply.GameAction.(*lotteryPlayer)
	}
	return nil
}

func Bet(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*lotteryArgs)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	ply.Bet(args.Area, args.Gold)
}

func GetLastHistory(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*lotteryArgs)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	ply.GetLastHistory(args.PageNum)
}

func Chat(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*lotteryArgs)
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
	args := iArgs.(*lotteryArgs)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}

	var e errcode.Error
	room := ply.Room()
	minDealerGold, _ := config.Int("lottery", room.SubId, "MinDealerGold")
	if args.Gold < minDealerGold {
		e = errcode.Retry
	}
	ply.WriteErr("ChangeDealerGold", e, "gold", args.Gold)
	if e != nil {
		return
	}
	ply.dealerLimitGold = args.Gold
}
