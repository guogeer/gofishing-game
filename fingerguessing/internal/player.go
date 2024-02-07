package internal

import (
	"gofishing-game/internal/errcode"
	"gofishing-game/internal/gameutils"
	"gofishing-game/internal/pb"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
	"slices"

	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/config"
)

var errInvalidGesture = errcode.New("invalid_shape", "错误的手势")

type fingerGuessingPlayer struct {
	*service.Player

	gesture   string
	winPlay   int
	losePlay  int
	totalPlay int
}

var _ service.EnterAction = (*fingerGuessingPlayer)(nil)

func newFingerGuessingPlayer(player *service.Player) service.EnterAction {
	return &fingerGuessingPlayer{
		Player: player,
	}
}

func (ply *fingerGuessingPlayer) TryEnter() errcode.Error {
	return nil
}

func (ply *fingerGuessingPlayer) TryLeave() errcode.Error {
	return nil
}

func (ply *fingerGuessingPlayer) BeforeEnter() {
	room := roomutils.GetRoomObj(ply.Player).CustomRoom().(*fingerGuessingRoom)
	ply.SetClientValue("fingerGuessingRoom", room.GetClientInfo())
}

func (ply *fingerGuessingPlayer) AfterEnter() {
}

func (ply *fingerGuessingPlayer) BeforeLeave() {
}

func (ply *fingerGuessingPlayer) Load(data interface{}) {
	bin := data.(*pb.UserBin)
	if bin.Room.FingerGuessing != nil {
		ply.winPlay = int(bin.Room.FingerGuessing.WinPlay)
		ply.losePlay = int(bin.Room.FingerGuessing.LosePlay)
		ply.totalPlay = int(bin.Room.FingerGuessing.TotalPlay)
	}
}

func (ply *fingerGuessingPlayer) Save(data interface{}) {
	bin := data.(*pb.UserBin)
	bin.Room.FingerGuessing = &pb.FingerGuessingRoom{
		WinPlay:   int32(ply.winPlay),
		LosePlay:  int32(ply.losePlay),
		TotalPlay: int32(ply.totalPlay),
	}
}

func (ply *fingerGuessingPlayer) ChooseGesture(guesture string) errcode.Error {
	if slices.Index(fingerGuessingGuestures, guesture) < 0 {
		return errInvalidGesture
	}

	room := roomutils.GetRoomObj(ply.Player).Room()

	var isAllChoose bool
	for _, roomp := range room.GetAllPlayers() {
		p := getFingerGuessingPlayer(roomp)
		if p.gesture == "" {
			isAllChoose = false
			break
		}
	}
	if isAllChoose {
		room.GameOver()
	}

	return nil
}

func (ply *fingerGuessingPlayer) Compare(guesture string) int {
	if ply.gesture == guesture {
		return 0
	}
	index := slices.Index(fingerGuessingGuestures, ply.gesture)
	if (index+1)%3 == slices.Index(fingerGuessingGuestures, ply.gesture) {
		return 1
	}
	return -1
}

func (ply *fingerGuessingPlayer) GameOver(guesture string) (int64, int) {
	cmp := ply.Compare(guesture)
	if cmp == 0 {
		return 0, 0
	}

	roomObj := roomutils.GetRoomObj(ply.Player)
	awardStr, _ := config.String("room", roomObj.Room().SubId, "Award")
	items := gameutils.ParseNumbericItems(awardStr)
	ply.BagObj().AddSomeItems(items, "finger_guessing_award")

	var winGold int64
	for _, item := range items {
		if item.GetId() == gameutils.ItemIdGold {
			winGold += item.GetNum()
		}
	}
	return winGold * int64(cmp), cmp
}

const enterActionFingerGuessingPlayer = "fingerGuessingPlayer"

func init() {
	service.AddAction(enterActionFingerGuessingPlayer, newFingerGuessingPlayer)
	cmd.Bind("chooseGesture", funcChooseGesture, (*fingerGuessingArgs)(nil))
}

var fingerGuessingGuestures = []string{"rock", "scissor", "paple"}

type fingerGuessingArgs struct {
	Guesture string `json:"gesture"`
}

func getFingerGuessingPlayer(player *service.Player) *fingerGuessingPlayer {
	return player.GetAction(enterActionFingerGuessingPlayer).(*fingerGuessingPlayer)
}

func funcChooseGesture(ctx *cmd.Context, data any) {
	args := data.(*fingerGuessingArgs)
	ply := service.GetPlayerByContext(ctx)
	if ply == nil {
		return
	}

	e := getFingerGuessingPlayer(ply).ChooseGesture(args.Guesture)
	ply.WriteJSON("chooseGesture", e)
}
