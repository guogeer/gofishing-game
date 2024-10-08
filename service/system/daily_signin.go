package system

import (
	"gofishing-game/internal/errcode"
	"gofishing-game/internal/gameutils"
	"gofishing-game/internal/pb"
	"gofishing-game/service"
	"time"

	"github.com/guogeer/quasar/v2/cmd"
	"github.com/guogeer/quasar/v2/config"
)

const maxSignInDay = 7

var (
	errSignInAlready = errcode.New("sign_in_already", "sign in already")
)

type dailySignInObj struct {
	player *service.Player

	drawTime  time.Time
	startTime time.Time
	drawState int
}

var _ service.EnterAction = (*dailySignInObj)(nil)

func newDailySignInObj(player *service.Player) service.EnterAction {
	obj := &dailySignInObj{
		player: player,
	}
	player.DataObj().Push(obj)
	return obj
}

func (obj *dailySignInObj) BeforeEnter() {
	obj.update(time.Now())
}

func (obj *dailySignInObj) update(current time.Time) {
	dayNum := countIntervalDay(current, obj.startTime)
	if dayNum >= maxSignInDay {
		obj.drawState = 0
		obj.startTime = current
	}
}

type signInState struct {
	IsDraw    bool `json:"isDraw,omitempty"`
	DrawState int  `json:"drawState,omitempty"`
	DayIndex  int  `json:"dayIndex,omitempty"`
}

func (obj *dailySignInObj) currentState(current time.Time) signInState {
	obj.update(current)

	dayIndex := countIntervalDay(current, obj.startTime)
	dayIndex = dayIndex % maxSignInDay
	return signInState{
		IsDraw:    countIntervalDay(obj.drawTime, current) == 0,
		DrawState: obj.drawState,
		DayIndex:  dayIndex,
	}
}

func (obj *dailySignInObj) Draw() errcode.Error {
	now := time.Now()

	state := obj.currentState(now)
	if state.IsDraw {
		return errSignInAlready
	}

	obj.drawState |= 1 << state.DayIndex
	obj.drawTime = now

	reward, _ := config.String("signin", config.RowId(state.DayIndex+1), "reward")
	obj.player.BagObj().AddSomeItems(gameutils.ParseNumbericItems(reward), "sign_in")
	return nil
}

func (obj *dailySignInObj) Load(data any) {
	bin := data.(*pb.UserBin)

	obj.drawTime = time.Unix(bin.Global.SignIn.DrawTs, 0)
	obj.startTime = time.Unix(bin.Global.SignIn.StartTs, 0)
	obj.drawState = int(bin.Global.SignIn.DrawState)
}

func (obj *dailySignInObj) Save(data any) {
	bin := data.(*pb.UserBin)

	bin.Global.SignIn = &pb.DailySignIn{
		StartTs:   obj.startTime.Unix(),
		DrawTs:    obj.drawTime.Unix(),
		DrawState: int32(obj.drawState),
	}
}

func (obj *dailySignInObj) Look() {
	obj.player.WriteJSON("lookSignIn", obj.currentState(time.Now()))
}

// 计算间隔天数
func countIntervalDay(t1, t2 time.Time) int {
	if t1.After(t2) {
		t2, t1 = t1, t2
	}
	y, m, d := t1.Date()
	date1 := time.Date(y, m, d, 0, 0, 0, 0, t1.Location())
	duration := t2.Sub(date1)

	return int(duration.Hours() / 24)
}

const enterActionDailySignIn = "dailySignIn"

func init() {
	service.AddAction(enterActionDailySignIn, newDailySignInObj)
	cmd.Bind("drawSignIn", funcDrawSignIn, (*signInArgs)(nil))
	cmd.Bind("lookSignIn", funcLookSignIn, (*signInArgs)(nil))
}

type signInArgs struct{}

func getSignInObj(player *service.Player) *dailySignInObj {
	return player.GetAction(enterActionDailySignIn).(*dailySignInObj)
}

func funcDrawSignIn(ctx *cmd.Context, data any) {
	ply := service.GetPlayerByContext(ctx)
	if ply == nil {
		return
	}

	e := getSignInObj(ply).Draw()
	ply.WriteErr("drawSignIn", e)
}

func funcLookSignIn(ctx *cmd.Context, data any) {
	ply := service.GetPlayerByContext(ctx)
	if ply == nil {
		return
	}

	getSignInObj(ply).Look()
}
