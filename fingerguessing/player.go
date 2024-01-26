package fingerguessing

import (
	"gofishing-game/internal/errcode"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
	"slices"

	"github.com/guogeer/quasar/cmd"
)

var errInvalidGesture = errcode.New("invalid_shape", "错误的手势")

type fingerGuessingPlayer struct {
	player *service.Player

	gesture   string
	winPlay   int
	losePlay  int
	totalPlay int
}

var _ service.EnterAction = (*fingerGuessingPlayer)(nil)

func newFingerGuessingPlayer(player *service.Player) service.EnterAction {
	return &fingerGuessingPlayer{
		player: player,
	}
}

func (obj *fingerGuessingPlayer) BeforeEnter() {
}

func (obj *fingerGuessingPlayer) Load(data interface{}) {
	// bin := data.(*bin.UserBin)
	// obj.winPlay = bin.
}

func (obj *fingerGuessingPlayer) Save(data interface{}) {
}

func (obj *fingerGuessingPlayer) ChooseGesture(guesture string) errcode.Error {
	if slices.Index(fingerGuessingGuestures, guesture) < 0 {
		return errInvalidGesture
	}

	room := roomutils.GetRoomObj(obj.player).Room()

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
