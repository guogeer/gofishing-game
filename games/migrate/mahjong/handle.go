package mahjong

import (
	"gofishing-game/service"

	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/log"
)

type Args struct {
	UId         int
	Token       string
	SubId       int
	Card        int
	Type        int
	Info        string
	ItemId      int
	ToUId       int
	IsReadyHand bool
	Color       int
	Index       int
	TriCards    [3]int
}

func GetPlayer(id int) *MahjongPlayer {
	if p := service.GetPlayer(id); p != nil {
		return p.GameAction.(*MahjongPlayer)
	}
	return nil
}

func GetPlayerByContext(ctx *cmd.Context) *MahjongPlayer {
	if p := service.GetGatewayPlayer(ctx.Ssid); p != nil {
		return p.GameAction.(*MahjongPlayer)
	}
	return nil
}

func AddHandlers(serverName string) {
	// cmd.BindFunc(Leave, (*Args)(nil))
	// cmd.BindFunc(Ready, (*Args)(nil))
	cmd.BindFunc(ExchangeTriCards, (*Args)(nil), cmd.WithServer(serverName))
	cmd.BindFunc(ChooseColor, (*Args)(nil), cmd.WithServer(serverName))
	cmd.BindFunc(Chow, (*Args)(nil), cmd.WithServer(serverName))
	cmd.BindFunc(Pong, (*Args)(nil), cmd.WithServer(serverName))
	cmd.BindFunc(Kong, (*Args)(nil), cmd.WithServer(serverName))
	cmd.BindFunc(Pass, (*Args)(nil), cmd.WithServer(serverName))
	cmd.BindFunc(Discard, (*Args)(nil), cmd.WithServer(serverName))
	cmd.BindFunc(Win, (*Args)(nil), cmd.WithServer(serverName))
	// cmd.BindFunc(ChangeRoom, (*Args)(nil))
	cmd.BindFunc(AutoPlay, (*Args)(nil), cmd.WithServer(serverName))
	cmd.BindFunc(GetChipHistory, (*Args)(nil), cmd.WithServer(serverName))
	cmd.BindFunc(GetWinOptions, (*Args)(nil), cmd.WithServer(serverName))
	cmd.BindFunc(BuyHorse, (*Args)(nil), cmd.WithServer(serverName)) // 湖南麻将买马
	cmd.BindFunc(Fail, (*Args)(nil), cmd.WithServer(serverName))
	cmd.BindFunc(ChoosePao, (*Args)(nil), cmd.WithServer(serverName))  // 郑州麻将带跑
	cmd.BindFunc(ChoosePiao, (*Args)(nil), cmd.WithServer(serverName)) // 湖北麻将选漂
	cmd.BindFunc(Double, (*Args)(nil), cmd.WithServer(serverName))     // 加倍
}

/*func ChangeRoom(ctx *cmd.Context, iArgs any) {
	// req := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	log.Debugf("player %d change room", ply.Id)
	ply.ChangeRoom()
}
*/

/*
func Leave(ctx *cmd.Context, iArgs any) {
	// req := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	log.Debugf("player %d leave room", ply.Id)
	ply.Leave()
}
*/

func ExchangeTriCards(ctx *cmd.Context, iArgs any) {
	args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	ply.ExchangeTriCards(args.TriCards)
}

func ChooseColor(ctx *cmd.Context, iArgs any) {
	args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	ply.ChooseColor(args.Color)
}

func Discard(ctx *cmd.Context, iArgs any) {
	req := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	room := ply.Room()
	if !room.CanPlay(OptBaoTing) && req.IsReadyHand {
		return
	}
	ply.expectReadyHand = req.IsReadyHand && ply.expectReadyHand
	ply.Discard(req.Card)
}

func Chow(ctx *cmd.Context, iArgs any) {
	req := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	ply.Chow(req.Card)
}

func Pong(ctx *cmd.Context, iArgs any) {
	// req := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	ply.Pong()
}

func Kong(ctx *cmd.Context, iArgs any) {
	req := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	ply.Kong(req.Card)
}

func Pass(ctx *cmd.Context, iArgs any) {
	// req := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	log.Debugf("player %d send pass", ply.Id)
	ply.Pass()
}

func Win(ctx *cmd.Context, iArgs any) {
	// req := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	ply.Win()
}

/*
func Close(ctx *cmd.Context, iArgs any) {
	// req := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {

		return
	}

	ply.Close()
}
*/

func AutoPlay(ctx *cmd.Context, iArgs any) {
	req := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	ply.AutoPlay(req.Type)
}

func GetChipHistory(ctx *cmd.Context, iArgs any) {
	// req := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}

	ply.WriteJSON("getChipHistory", map[string]any{
		"detail": ply.chipHistory,
	})
}

func GetWinOptions(ctx *cmd.Context, iArgs any) {
	// req := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	ply.GetWinOptions()
}

func BuyHorse(ctx *cmd.Context, iArgs any) {
	data := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	index := ((data.Index%4 + 4) % 4)
	if obj, ok := ply.localObj.(*HunanObj); ok {
		obj.BuyHorse(index)
	}
}

// 放弃领取破产补助或充值
func Fail(ctx *cmd.Context, iArgs any) {
	// data := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	ply.Fail()
}

func ChoosePao(ctx *cmd.Context, iArgs any) {
	data := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	if obj, ok := ply.localObj.(*ZhengzhouObj); ok {
		obj.ChoosePao(data.Index)
	}
}

func ChoosePiao(ctx *cmd.Context, iArgs any) {
	data := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	if obj, ok := ply.localObj.(HubeiObj); ok {
		obj.ChoosePiao(data.Index)
	}
}

func Double(ctx *cmd.Context, iArgs any) {
	// data := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	ply.Double()
}
