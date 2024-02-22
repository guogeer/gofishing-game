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

var _ service.GameAction = (*fingerGuessingPlayer)(nil)

func (ply *fingerGuessingPlayer) TryEnter() errcode.Error {
	return nil
}

func (ply *fingerGuessingPlayer) TryLeave() errcode.Error {
	return nil
}

func (ply *fingerGuessingPlayer) BeforeEnter() {
	room := roomutils.GetRoomObj(ply.Player).CustomRoom().(*fingerGuessingRoom)
	ply.SetClientValue("roomInfo", room.GetClientInfo())
	ply.SetClientValue("gesture", ply.gesture)
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
	if bin.Room == nil {
		bin.Room = &pb.RoomBin{}
	}
	bin.Room.FingerGuessing = &pb.FingerGuessingRoom{
		WinPlay:   int32(ply.winPlay),
		LosePlay:  int32(ply.losePlay),
		TotalPlay: int32(ply.totalPlay),
	}
}

func (ply *fingerGuessingPlayer) ChooseGesture(gesture string) errcode.Error {
	if slices.Index(fingerGuessingGuestures, gesture) < 0 {
		return errInvalidGesture
	}
	if ply.gesture != "" {
		return errcode.New("choose_gesture_already", "choose gesture already")
	}
	ply.gesture = gesture

	room := roomutils.GetRoomObj(ply.Player).Room()
	if room.Status == 0 {
		return errcode.New("room_not_playing", "room not playing")
	}
	cost, _ := config.String("room", room.SubId, "cost")

	costItems := gameutils.ParseNumbericItems(cost)
	for _, item := range costItems {
		if !ply.BagObj().IsEnough(item.GetId(), item.GetNum()) {
			return errcode.MoreItem(item.GetId())
		}
	}

	ply.BagObj().CostSomeItems(costItems, "choose_guesture")

	var isAllChoose bool
	for _, roomp := range room.GetAllPlayers() {
		other := getFingerGuessingPlayer(roomp)
		if other.gesture == "" {
			isAllChoose = false
			break
		}
	}
	if isAllChoose {
		room.GameOver()
	}

	return nil
}

func (ply *fingerGuessingPlayer) Compare(gesture string) int {
	if ply.gesture == gesture {
		return 0
	}
	index := slices.Index(fingerGuessingGuestures, gesture)
	if (index+1)%3 == slices.Index(fingerGuessingGuestures, ply.gesture) {
		return 1
	}
	return -1
}

func (ply *fingerGuessingPlayer) GameOver(guesture string) (int64, int) {
	cmp := ply.Compare(guesture)
	room := roomutils.GetRoomObj(ply.Player).Room()

	cost, _ := config.String("room", room.SubId, "cost")
	costItems := gameutils.ParseNumbericItems(cost)

	if cmp == 0 {
		ply.BagObj().AddSomeItems(costItems, "finger_guessing_back")
	}
	if cmp <= 0 {
		return gameutils.CountItems(costItems, gameutils.ItemIdGold), cmp
	}

	awardStr, _ := config.String("room", room.SubId, "award")
	awardItems := gameutils.ParseNumbericItems(awardStr)
	ply.BagObj().AddSomeItems(awardItems, "finger_guessing_award")

	return gameutils.CountItems(awardItems, gameutils.ItemIdGold) * int64(cmp), cmp

}

func init() {
	cmd.Bind("chooseGesture", funcChooseGesture, (*fingerGuessingArgs)(nil))
}

var fingerGuessingGuestures = []string{"rock", "scissor", "paple"}

type fingerGuessingArgs struct {
	Guesture string `json:"gesture"`
}

func getFingerGuessingPlayer(player *service.Player) *fingerGuessingPlayer {
	return player.GameAction.(*fingerGuessingPlayer)
}

func funcChooseGesture(ctx *cmd.Context, data any) {
	args := data.(*fingerGuessingArgs)
	ply := service.GetPlayerByContext(ctx)
	if ply == nil {
		return
	}

	e := getFingerGuessingPlayer(ply).ChooseGesture(args.Guesture)
	ply.WriteErr("chooseGesture", e, "gesture", args.Guesture, "uid", ply.Id)
}
