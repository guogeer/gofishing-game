package fingerguessing

import (
	"gofishing-game/internal/gameutils"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"

	"github.com/guogeer/quasar/v2/cmd"
)

func init() {
	cmd.Bind("chooseGesture", funcChooseGesture, (*fingerGuessingArgs)(nil), cmd.WithServer((*fingerGuessingWorld)(nil).GetName()))
}

var fingerGuessingGuestures = []string{"rock", "scissor", "paple"}

type fingerGuessingArgs struct {
	Gesture string `json:"gesture"`
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

	e := getFingerGuessingPlayer(ply).ChooseGesture(args.Gesture)
	ply.WriteErr("chooseGesture", e, map[string]any{"gesture": args.Gesture, "uid": ply.Id})
	roomObj := roomutils.GetRoomObj(ply)
	if roomObj.GetSeatIndex() != roomutils.NoSeat {
		roomObj.Room().Broadcast("chooseGesture", gameutils.MergeError(nil, map[string]any{"uid": ply.Id, "gesture": args.Gesture, "seatIndex": roomObj.GetSeatIndex()}), ply.Id)
	}
}
