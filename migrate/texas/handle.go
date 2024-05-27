package texas

import (
	"gofishing-game/service"

	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/config"
)

type texasArgs struct {
	SeatIndex int   `json:"seatIndex,omitempty"`
	Gold      int64 `json:"gold,omitempty"`
	Auto      int   `json:"auto,omitempty"`
	IsShow    bool  `json:"isShow,omitempty"`
	Level     int   `json:"level,omitempty"` // 房间等级
}

func init() {
	cmd.BindFunc(TakeAction, (*texasArgs)(nil))
	cmd.BindFunc(SitDown, (*texasArgs)(nil))
	cmd.BindFunc(ChooseBankroll, (*texasArgs)(nil))
	cmd.BindFunc(SitUp, (*texasArgs)(nil))       // 站起
	cmd.BindFunc(SetAutoPlay, (*texasArgs)(nil)) // 托管
	cmd.BindFunc(ShowCard, (*texasArgs)(nil))    // 亮牌
	cmd.BindFunc(Rebuy, (*texasArgs)(nil))       // 重购
	cmd.BindFunc(Addon, (*texasArgs)(nil))       // 增购

	cmd.BindFunc(SetWallet, (*texasArgs)(nil))
	cmd.BindFunc(funcRecommendRooms, (*texasArgs)(nil))
}

func GetPlayerByContext(ctx *cmd.Context) *TexasPlayer {
	if p := service.GetGatewayPlayer(ctx.Ssid); p != nil {
		return p.GameAction.(*TexasPlayer)
	}
	return nil
}

func TakeAction(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*texasArgs)
	ply := GetPlayerByContext(ctx)
	if ply == nil {

		return
	}
	ply.TakeAction(args.Gold)
}

func SitDown(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*texasArgs)
	ply := GetPlayerByContext(ctx)
	if ply == nil {

		return
	}
	ply.SitDown(args.SeatIndex)
}

func ChooseBankroll(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*texasArgs)
	ply := GetPlayerByContext(ctx)
	if ply == nil {

		return
	}
	ply.ChooseBankroll(args.Gold)
}

func SitUp(ctx *cmd.Context, iArgs interface{}) {
	// args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {

		return
	}
	ply.SitUp()
}

func SetAutoPlay(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*texasArgs)
	ply := GetPlayerByContext(ctx)
	if ply == nil {

		return
	}
	ply.SetAutoPlay(args.Auto)
}

func ShowCard(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*texasArgs)
	ply := GetPlayerByContext(ctx)
	if ply == nil {

		return
	}
	ply.ShowCard(args.IsShow)
}

func Rebuy(ctx *cmd.Context, iArgs interface{}) {
	// args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {

		return
	}
	ply.Rebuy()
}

func Addon(ctx *cmd.Context, iArgs interface{}) {
	// args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {

		return
	}
	ply.Addon()
}

func SetWallet(ctx *cmd.Context, data interface{}) {
	args := data.(*texasArgs)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	ply.SetWallet(args.Gold)
}

// 推荐房间
func funcRecommendRooms(ctx *cmd.Context, data interface{}) {
	args := data.(*texasArgs)
	ss := &cmd.Session{Id: ctx.Ssid, Out: ctx.Out}
	rooms := RecommendRooms(args.Level)

	var roomList []TexasRoomInfo
	for _, room := range rooms {
		texasRoom := room.CustomRoom().(*TexasRoom)
		minBankroll, _ := config.Int("texasroom", room.SubId, "minBankroll")
		maxBankroll, _ := config.Int("texasroom", room.SubId, "maxBankroll")
		info := TexasRoomInfo{
			Id:          texasRoom.Id,
			SubId:       texasRoom.SubId,
			FrontBlind:  texasRoom.frontBlind,
			SmallBlind:  texasRoom.smallBlind,
			BigBlind:    texasRoom.bigBlind,
			MinBankroll: minBankroll,
			MaxBankroll: maxBankroll,
			ActiveUsers: texasRoom.NumSeatPlayer(),
		}
		roomList = append(roomList, info)
	}
	service.WriteMessage(ss, "recommendRooms", map[string]any{"level": args.Level, "rooms": roomList})
}
