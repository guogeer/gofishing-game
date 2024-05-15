package texas

import (
	"gofishing-game/service"

	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/config"
)

type Args struct {
	SeatIndex int
	Gold      int64
	Auto      int
	IsShow    bool
	Level     int // 房间等级
}

type LotteryArgs struct {
	AreaId int
	Gold   int64
	UId    int
}

func init() {
	cmd.Bind(TakeAction, (*Args)(nil))
	cmd.Bind(SitDown, (*Args)(nil))
	cmd.Bind(ChooseBankroll, (*Args)(nil))
	cmd.Bind(SitUp, (*Args)(nil))       // 站起
	cmd.Bind(SetAutoPlay, (*Args)(nil)) // 托管
	cmd.Bind(ShowCard, (*Args)(nil))    // 亮牌
	cmd.Bind(Rebuy, (*Args)(nil))       // 重购
	cmd.Bind(Addon, (*Args)(nil))       // 增购

	cmd.Bind(BetLottery, (*LotteryArgs)(nil))
	cmd.Bind(LookLottery, (*LotteryArgs)(nil))
	cmd.Bind(GetLotteryHistory, (*LotteryArgs)(nil))
	cmd.Bind(GetLotteryRank, (*LotteryArgs)(nil))

	cmd.Bind(SetWallet, (*Args)(nil))
	cmd.Bind(RecommendRooms, (*Args)(nil))
}

func GetPlayerByContext(ctx *cmd.Context) *TexasPlayer {
	if p := service.GetGatewayPlayer(ctx.Ssid); p != nil {
		return p.GameAction.(*TexasPlayer)
	}
	return nil
}

func TakeAction(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {

		return
	}
	ply.TakeAction(args.Gold)
}

func SitDown(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {

		return
	}
	ply.SitDown(args.SeatIndex)
}

func ChooseBankroll(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*Args)
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
	args := iArgs.(*Args)
	ply := GetPlayerByContext(ctx)
	if ply == nil {

		return
	}
	ply.SetAutoPlay(args.Auto)
}

func ShowCard(ctx *cmd.Context, iArgs interface{}) {
	args := iArgs.(*Args)
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

func BetLottery(ctx *cmd.Context, data interface{}) {
	args := data.(*LotteryArgs)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	code := GetLotterySystem().Bet(ply, args.AreaId, args.Gold)
	ply.WriteJSON("BetLottery", map[string]any{"Code": code, "Msg": code.String(), "Gold": args.Gold})
}

func LookLottery(ctx *cmd.Context, data interface{}) {
	// args := data.(*LotteryArgs)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	sys := GetLotterySystem()
	params := map[string]any{
		"Status":    sys.Status,
		"Areas":     sys.areas,
		"PrizePool": sys.prizePool,
		"Sec":       service.GetShowTime(sys.deadline),
	}
	if e := sys.history.Back(); e != nil {
		params["LastRecord"] = e.Value.(LotteryRecord)
	}
	if user, ok := sys.users[ply.Id]; ok {
		params["MyAreas"] = user.areas
	}
	ply.WriteJSON("LookLottery", params)
}

func GetLotteryHistory(ctx *cmd.Context, data interface{}) {
	// args := data.(*LotteryArgs)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	sys := GetLotterySystem()

	last := make([]LotteryRecord, 0, 30)
	for e := sys.history.Front(); e != nil; e = e.Next() {
		last = append(last, e.Value.(LotteryRecord))
	}
	ply.WriteJSON("GetLotteryHistory", map[string]any{"Last": last})
}

func GetLotteryRank(ctx *cmd.Context, data interface{}) {
	// args := data.(*LotteryArgs)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	sys := GetLotterySystem()

	ply.WriteJSON("GetLotteryRank", map[string]any{"Yesterday": sys.yesterdayRank, "Today": sys.todayRank})
}

func SetWallet(ctx *cmd.Context, data interface{}) {
	args := data.(*LotteryArgs)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}
	ply.SetWallet(args.Gold)
}

// 推荐房间
func RecommendRooms(ctx *cmd.Context, data interface{}) {
	args := data.(*Args)
	ss := &cmd.Session{Id: ctx.Ssid, Out: ctx.Out}
	rooms := service.RecommendRooms(args.Level)

	var roomList []TexasRoomInfo
	for _, room := range rooms {
		texasRoom := room.CustomRoom().(*TexasRoom)
		minBankroll, _ := config.Int("texasroom", room.SubId, "MinBankroll")
		maxBankroll, _ := config.Int("texasroom", room.SubId, "MaxBankroll")
		info := TexasRoomInfo{
			Id:          texasRoom.Id,
			SubId:       texasRoom.GetSubId(),
			FrontBlind:  texasRoom.frontBlind,
			SmallBlind:  texasRoom.smallBlind,
			BigBlind:    texasRoom.bigBlind,
			MinBankroll: minBankroll,
			MaxBankroll: maxBankroll,
			ActiveUsers: texasRoom.CountPlayersInSeat(),
		}
		roomList = append(roomList, info)
	}
	service.WriteMessage(ss, "RecommendRooms", map[string]any{"Level": args.Level, "Rooms": roomList})
}
