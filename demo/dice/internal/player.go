package dice

import (
	"gofishing-game/internal/errcode"
	"gofishing-game/internal/gameutils"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"

	"github.com/guogeer/quasar/config"
)

var (
	errAlreadyBet = errcode.New("bet_already", "bet already")
)

type DicePlayer struct {
	*service.Player

	areas [MaxBetArea]int64
}

func (ply *DicePlayer) TryEnter() errcode.Error {
	return nil
}

func (ply *DicePlayer) BeforeEnter() {
	ply.SetClientValue("areas", ply.areas)
}

func (ply *DicePlayer) BeforeLeave() {

}

func (ply *DicePlayer) AfterEnter() {

}

func (ply *DicePlayer) TryLeave() errcode.Error {
	if ply.countAllBet() > 0 {
		return errAlreadyBet
	}
	return nil
}

func (ply *DicePlayer) Room() *DiceRoom {
	if room := roomutils.GetRoomObj(ply.Player).CustomRoom(); room != nil {
		return room.(*DiceRoom)
	}
	return nil
}

func (ply *DicePlayer) countAllBet() int64 {
	var sum int64
	for _, n := range ply.areas {
		sum += n
	}
	return sum
}

func (ply *DicePlayer) GameOver() {
	for k := range ply.areas {
		ply.areas[k] = 0
	}
}

func (ply *DicePlayer) OnLeave() {
}

func (ply *DicePlayer) Bet(area int, gold int64) errcode.Error {
	room := ply.Room()
	if room == nil {
		return errcode.Retry
	}
	if room.Status == 0 {
		return errcode.Retry
	}
	if area < 0 || area >= len(ply.areas) || gold <= 0 {
		return errcode.Retry
	}
	n, _ := config.Int("config", "diceMinGold", "value")
	if !ply.BagObj().IsEnough(gameutils.ItemIdGold, max(n, gold)) {
		return errcode.MoreItem(gameutils.ItemIdGold)
	}
	n, _ = config.Int("config", "diceMaxGold", "value")
	if ply.countAllBet()+gold > n {
		return errcode.New("too_much_bet", "too much bet")
	}
	ply.areas[area] += gold
	ply.BagObj().Add(gameutils.ItemIdGold, -gold, "dice_bet")
	room.OnBet(ply, area, gold)
	return nil
}
