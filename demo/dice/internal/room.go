package dice

import (
	"fmt"
	"gofishing-game/internal/gameutils"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
	"math/rand"
	"time"

	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
	"github.com/guogeer/quasar/utils"
)

const (
	MaxBetArea = 18
	syncTime   = 1000 * time.Millisecond
)

type DiceRoom struct {
	*roomutils.Room

	betArea [MaxBetArea]int64
	chips   []int64
	last    [64]int // 历史记录
	lasti   int     // 历史记录索引
}

func (room *DiceRoom) OnEnter(player *service.Player) {
	ply := player.GameAction.(*DicePlayer)
	log.Infof("player %d enter room %d", ply.Id, room.Id)

	ply.SetClientValue("roomInfo", map[string]any{
		"status":    room.Status,
		"subId":     room.Room.SubId,
		"chips":     room.chips,
		"countdown": room.Countdown(),
		"betArea":   room.betArea,
	})
}

func (room *DiceRoom) OnBet(ply *DicePlayer, area int, gold int64) {
	room.betArea[area] += gold
}

func (room *DiceRoom) StartGame() {
	room.Room.StartGame()
	for k := range room.betArea {
		room.betArea[k] = 0
	}
}

func (room *DiceRoom) Award() {
	// 骰子
	var winArea [MaxBetArea]int64
	dice1, dice2 := rand.Intn(6)+1, rand.Intn(6)+1
	// dice1, dice2 = 2, 2
	sum := dice1 + dice2
	if dice1 == dice2 {
		a := []int{0, 11, 12, 13, 15, 16, 17}
		k := a[dice1]
		winArea[k] = 1
		winArea[14] = 1
	} else {
		if sum > 2 && sum < 7 {
			winArea[0] = 1
		}
		if sum > 6 && sum < 12 {
			winArea[10] = 1
		}
	}
	if sum > 2 && sum < 12 {
		winArea[sum-2] = 1
	}
	room.Sync()
	room.last[room.lasti] = dice1*10 + dice2
	room.lasti = (room.lasti + 1) % len(room.last)

	bigWinner, _ := config.Int("config", "diceBigWinner", "value", 50000)
	for _, one := range room.GetAllPlayers() {
		p := one.GameAction.(*DicePlayer)

		var gold int64
		for k, v := range p.areas {
			if winArea[k] == 0 {
				continue
			}
			gold += v * winArea[k]
		}

		if gold >= bigWinner {
			msg := fmt.Sprintf("恭喜%v在骰子场赢得%d万金币", p.Nickname, gold/10000)
			service.BroadcastNotification(msg)
		}

		p.BagObj().Add(gameutils.ItemIdGold, gold, "dice_award")
		p.WriteJSON("award", map[string]any{"winGold": gold, "countdown": room.Countdown(), "dice2": []int{dice1, dice2}})
	}
	room.GameOver()
}

func (room *DiceRoom) OnTime() {
	room.Sync()
	utils.NewTimer(room.OnTime, syncTime)
}

func (room *DiceRoom) Sync() {
	room.Broadcast("sync", map[string]any{
		"onlineNum": len(room.GetAllPlayers()),
		"betArea":   room.betArea,
	})
}

func (room *DiceRoom) GetLastHistory(n int) []int {
	var last []int
	N := len(room.last)
	if N == 0 {
		return last
	}
	for i := (N - n + room.lasti) % N; i != room.lasti; i = (i + 1) % N {
		d := room.last[i]
		if d > 0 {
			last = append(last, d)
		}
	}
	return last
}
