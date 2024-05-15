package internal

import (
	mjutils "gofishing-game/migrate/mahjong/utils"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
	"math/rand"
	"quasar/utils"
	"strings"
	"time"

	"github.com/guogeer/quasar/config"
)

type HunanMahjong struct {
	room          *MahjongRoom
	boomPlayer    *MahjongPlayer
	buyHorseTimer *utils.Timer

	ghostCard int
}

func (h *HunanMahjong) OnCreateRoom() {
	room := h.room
	// 默认没有字牌
	zipai := []int{60, 70, 80, 90, 100, 110, 120}
	room.CardSet().Remove(zipai...)

	room.CardSet().Recover(100)
	if room.CanPlay(OptFanGui1) || room.CanPlay(OptFanGui2) {
		room.CardSet().Remove(100)
	}
}

func (h *HunanMahjong) OnEnter(comer *MahjongPlayer) {
	room := h.room

	if room.Status == roomStatusBuyHorse {
		uid := 0
		for _, p := range room.winPlayers {
			uid = p.Id
		}
		comer.WriteJSON("startBuyHorse", map[string]any{"ts": room.deadline.Unix(), "num": 4, "uid": uid})
	}
	data := map[string]any{
		"card":  h.ghostCard,
		"ghost": h.getAnyCards(),
	}
	comer.SetClientValue("localMahjong", data)
}

func (h *HunanMahjong) getAnyCards() []int {
	room := h.room
	m := make(map[int]bool)

	ghostNum := 0
	if room.CanPlay(OptFanGui1) {
		// 翻鬼
		ghostNum = 1
	} else if room.CanPlay(OptFanGui2) {
		// 翻双鬼
		ghostNum = 2
	} else {
		// 默认红中做鬼
		m[100] = true
	}
	for _, c := range GetNextCards(h.ghostCard, ghostNum) {
		m[c] = true
	}
	var a []int
	for c := range m {
		a = append(a, c)
	}
	return a
}

func (h *HunanMahjong) OnReady() {
	room := h.room

	h.ghostCard = -1
	// 开始选鬼牌
	if room.CanPlay(OptFanGui1) || room.CanPlay(OptFanGui2) {
		h.ghostCard = room.CardSet().Deal()
	}
	room.Broadcast("ChooseGhostCard", map[string]any{
		"card":  h.ghostCard,
		"Ghost": h.getAnyCards(),
	})

	//重新洗牌
	room.CardSet().Shuffle()
	room.StartDealCard()
	room.dealer.OnDraw()
}

func (h *HunanMahjong) OnWin() {
	room := h.room
	h.boomPlayer = room.kongPlayer
	// 开始中马
	d := 6 * time.Second
	room.Status = roomStatusBuyHorse
	room.deadline = time.Now().Add(d)
	uid := 0
	for _, p := range room.winPlayers {
		uid = p.Id
	}
	roomName, _ := config.String("Room", room.SubId, "RoomName")
	if strings.Contains(roomName, "New") {
		c := room.lastCard
		if room.CardSet().Count() > 0 {
			c = room.CardSet().Deal()
		}
		room.buyHorse = c
		room.Broadcast("BuyHorse", map[string]any{"uid": uid, "NextCard": c})
		room.Award()
	} else {
		room.Broadcast("StartBuyHorse", map[string]any{"ts": room.deadline.Unix(), "Num": 4, "uid": uid})
		h.buyHorseTimer = utils.NewTimer(func() {
			index := rand.Intn(4)
			h.OnBuyHorse(index)
		}, d)
	}
}

func (h *HunanMahjong) OnBuyHorse(index int) {
	room := h.room
	utils.StopTimer(h.buyHorseTimer)

	var horses [4]int
	for i := range horses {
		c := room.lastCard
		if room.CardSet().Count() > 0 {
			c = room.CardSet().Deal()
		}
		horses[i] = c
	}
	room.buyHorse = horses[0]
	room.Broadcast("BuyHorse", map[string]any{"Horse": room.buyHorse, "Index": index, "Horses": horses})
	room.Award()
}

func (h *HunanMahjong) Award() {
	room := h.room
	unit, _ := config.Int("Room", room.SubId, "Unit")

	horseValue := room.buyHorse % 10
	if room.IsAnyCard(room.buyHorse) {
		horseValue = 10
	}
	winGold := unit * int64(horseValue+1)

	for _, p := range room.winPlayers {
		bills := make([]Bill, room.NumSeat())

		addition2 := map[string]int{}
		detail := ChipDetail{Seats: 1 << uint(p.GetSeatIndex()), Operate: mjutils.OperateWin}
		if cards := h.getAnyCards(); len(cards) > 0 && room.CanPlay(OptWuGuiJiaBei) && CountSomeCards(p.handCards, nil, cards...) == 0 {
			addition2["WGJB"] = 2
			winGold *= 2
			p.totalTimes["WGJB"]++
		}
		detail.Chip = -winGold
		detail.Addition2 = addition2

		// 有人放炮
		if p.drawCard == -1 {
			other := h.boomPlayer
			bill := &bills[other.GetSeatIndex()]
			bill.Details = append(bill.Details, detail)
		} else {
			for k := 0; k < room.NumSeat(); k++ {
				if other := room.GetPlayer(k); other != p {
					bill := &bills[other.GetSeatIndex()]
					bill.Details = append(bill.Details, detail)
				}
			}
		}
		room.Billing(bills)
	}
}

func (h *HunanMahjong) GameOver() {
	h.ghostCard = -1
}

type HunanMahjongWorld struct{}

func (w *HunanMahjongWorld) NewRoom(subId int) *roomutils.Room {
	r := NewMahjongRoom(id, subId)
	r.localMahjong = &HunanMahjong{
		room:      r,
		ghostCard: -1,
	}
	r.SetNoPlay(OptAbleRobKong)
	r.SetNoPlay(OptSevenPairs)
	r.SetPlay(OptCostAfterKong)

	// 2017-6-2 增加翻鬼、翻双鬼、无鬼加倍
	r.SetNoPlay(OptFanGui1)
	r.SetNoPlay(OptFanGui2)
	r.SetNoPlay(OptWuGuiJiaBei)

	// 增加2、3、4人
	r.SetNoPlay("seat_2")
	r.SetNoPlay("seat_3")
	r.SetNoPlay("seat_4")

	return r.Room
}

func (w *HunanMahjongWorld) GetName() string {
	return "hnmj"
}

func (w *HunanMahjongWorld) NewPlayer() *service.Player {
	p := NewMahjongPlayer()
	p.Player = service.NewPlayer(p)
	// p.Player.ItemAdder = p
	// p.Player.SessionCloser = p

	p.localObj = &HunanObj{MahjongPlayer: p}
	return p.Player
}

// HunanObj 湖南麻将玩家逻辑
type HunanObj struct {
	*MahjongPlayer
}

// BuyHorse 湖南麻将买马
func (ply *HunanObj) BuyHorse(index int) {
	room := ply.Room()
	if room.Status != roomStatusBuyHorse {
		return
	}
	if !ply.isWin {
		return
	}

	if local, ok := room.localMahjong.(*HunanMahjong); ok {
		local.OnBuyHorse(index)
	}
}

func (ply *HunanObj) IsAbleWin() bool {
	// 没出牌之前，手牌中有4张癞子牌
	room := ply.Room()
	if !room.CanPlay(OptFanGui1) && !room.CanPlay(OptFanGui2) && ply.discardNum == 0 && ply.drawCard != -1 && CountSomeCards(ply.handCards, nil, 100) == 4 {
		return true
	}
	return ply.MahjongPlayer.IsAbleWin()
}
