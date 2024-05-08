package fruit

import (
	"gofishing-game/internal/errcode"
	"gofishing-game/internal/gameutils"
	"gofishing-game/service/roomutils"

	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/util"
)

type FruitObj struct {
	BetArea [MaxBetArea]int64
	AllBet  int64

	player *FruitPlayer
}

func NewFruitObj(ply *FruitPlayer) *FruitObj {
	return &FruitObj{player: ply}
}

func (fruitObj *FruitObj) OnEnter() {
	data := map[string]any{}
	ply := fruitObj.player
	room := ply.Room()
	if room.Status == RoomStatusPlaying {
		data["BetArea"] = fruitObj.BetArea
	}
	ply.WriteJSON("GetFruitInfo", data)
}

func (fruitObj *FruitObj) Bet(area int, gold int64) {
	code := Ok
	ply := fruitObj.player
	room := ply.Room()
	if room == nil {
		code = errcode.Retry
		return
	}

	if room.Status != RoomStatusPlaying {
		code = errcode.Retry
	}
	if area < 0 || area >= len(fruitObj.BetArea) || gold <= 0 {
		code = errcode.Retry
	}
	if ply.BagObj().NumItem(gameutils.ItemIdGold) < gold {
		code = MoreGold
	}

	s, _ := config.String("config", "FruitGoldLimit", "Value")
	limit := utils.ParseIntSlice(s)
	if len(limit) > 0 && ply.BagObj().NumItem(gameutils.ItemIdGold) < limit[0] {
		code = MoreGold
	}
	if len(limit) > 1 && fruitObj.AllBet+gold > limit[1] {
		code = TooMuchBet
	}
	data := map[string]any{
		"Code": code,
		"Msg":  code.String(),
		"UId":  ply.Id,
		"Area": area,
		"Gold": gold,
	}
	ply.WriteJSON("Bet", data)
	if code != Ok {
		return
	}
	room.Broadcast("Bet", data, ply.Id)

	fruitObj.AllBet += gold
	fruitObj.BetArea[area] += gold
	ply.AddGold(-gold, util.GUID(), "sum.fruits_bet")
	roomutils.GetRoomObj(ply.Player).WinGold += gold
	room.OnBet(ply, area, gold)
}

func (fruitObj *FruitObj) GameOver() {
	// game over
	fruitObj.AllBet = 0
	for k, _ := range fruitObj.BetArea {
		fruitObj.BetArea[k] = 0
	}
}

func (fruitObj *FruitObj) GetHistory(n int) {
	ply := fruitObj.player
	room := ply.Room()
	if room == nil {
		return
	}
	ply.WriteJSON("GetHistory", map[string]any{"Last": room.GetLast(n)})
}
