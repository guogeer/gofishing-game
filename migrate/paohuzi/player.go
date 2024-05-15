package paohuzi

import (
	"gofishing-game/internal/errcode"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
	"sort"
	"third/cardutil"

	"github.com/guogeer/quasar/log"
)

type OperateTip struct {
	Type  int
	Melds []cardutil.PaohuziMeld `json:",omitempty"`
}

// 玩家信息
type PaohuziUserInfo struct {
	service.UserInfo

	SeatId  int
	CardNum int

	Cards          []int `json:",omitempty"`
	DiscardHistory []int `json:",omitempty"`

	IsReady, IsReadyHand, IsGiveUp bool `json:",omitempty"`

	Melds []cardutil.PaohuziMeld

	DrawCard, DiscardCard int `json:",omitempty"`
}

// score 胡子
// quality 番数

type PaohuziPlayer struct {
	*service.Player

	cards      []int // 手牌
	isAutoPlay bool
	discardNum int

	drawCard           int
	chowCards          [][3]int
	pongCard, kongCard int
	discardHistory     []int

	isGiveUp        bool // 放弃了
	isReadyHand     bool
	unableChowCards map[int]bool
	unablePongCards map[int]bool
	unableWinCards  map[int]bool

	melds       []cardutil.PaohuziMeld
	operateTips []OperateTip

	winTimes   int   // 胡牌次数
	maxScore   int   // 最大胡子
	maxQuality int   // 最大番型
	maxGold    int64 // 最多金币
}

func (ply *PaohuziPlayer) TryLeave() errcode.Error {
	room := ply.Room()
	if room.Status != service.RoomStatusFree {
		return Retry
	}
	return Ok
}

func (ply *PaohuziPlayer) initGame() {
	for i := 0; i < len(ply.cards); i++ {
		ply.cards[i] = 0
	}
	ply.discardNum = 0
	ply.isAutoPlay = false

	ply.pongCard = -1
	ply.kongCard = -1
	ply.chowCards = nil

	ply.melds = nil
	ply.operateTips = nil
	ply.discardHistory = nil
	ply.unableChowCards = make(map[int]bool)
	ply.unablePongCards = make(map[int]bool)
	ply.unableWinCards = make(map[int]bool)

	ply.isReadyHand = false
	ply.isGiveUp = false
}

func (ply *PaohuziPlayer) GameOver() {
	ply.initGame()
}

func (ply *PaohuziPlayer) GetUserInfo(self bool) *PaohuziUserInfo {
	info := &PaohuziUserInfo{}
	info.UserInfo = ply.UserInfo
	// info.UId = ply.GetCharObj().Id
	info.SeatId = ply.GetSeatIndex()
	info.IsReady = roomutils.GetRoomObj(ply.Player).IsReady()
	info.DrawCard = ply.drawCard

	info.Cards = ply.GetSortedCards()
	info.Melds = ply.melds

	room := ply.Room()
	if room.discardPlayer == ply {
		info.DiscardCard = room.lastCard
	}

	info.IsGiveUp = ply.isGiveUp
	info.IsReadyHand = ply.isReadyHand
	return info
}

func (ply *PaohuziPlayer) GetSortedCards() []int {
	cards := make([]int, 0, 32)
	for c, n := range ply.cards {
		for k := 0; k < n; k++ {
			cards = append(cards, c)
		}
	}
	return cards
}

func (ply *PaohuziPlayer) Room() *PaohuziRoom {
	if room := roomutils.GetRoomObj(ply.Player).CustomRoom(); room != nil {
		return room.(*PaohuziRoom)
	}
	return nil
}

func (ply *PaohuziPlayer) Draw() {
	room := ply.Room()
	if other := room.discardPlayer; other != nil && room.lastCard != -1 {
		other.discardHistory = append(other.discardHistory, room.lastCard)
		log.Debugf("player %d draw, other %d history %v", ply.Id, other.Id, other.discardHistory)
	}

	drawCard := room.CardSet().Deal()
	log.Debugf("player %d draw card %d", ply.Id, drawCard)
	if drawCard == -1 {
		room.Award()
		return
	}
	room.Broadcast("Draw", map[string]any{"UId": ply.Id, "Card": drawCard})

	ply.drawCard = drawCard
	room.lastCard = drawCard
	room.discardPlayer = ply

	// 暗刻
	if ply.cards[drawCard] == 2 {
		room.expectPongPlayer = ply
		ply.Pong()
		return
	}
	// 暗杠
	if t := ply.GetKongType(drawCard); t == cardutil.PaohuziInvisibleKong {
		room.expectKongPlayer = ply
		ply.Kong()
		return
	}

	isPass := true
	meldCards := []int{drawCard, drawCard, drawCard, drawCard}
	for i := 0; i < room.NumSeat(); i++ {
		other := room.GetPlayer(i)
		tips := make([]OperateTip, 0, 4)
		// 吃
		if other.IsAbleChow() {
			samples := room.helper.TryChow(other.GetSortedCards(), drawCard)
			for _, sample := range samples {
				tips = append(tips, OperateTip{Type: cardutil.PaohuziChow, Melds: sample})
			}
			room.expectChowPlayers[other.Id] = other
			other.Timeout(func() { other.Pass() })
		}
		// 碰
		if other.IsAblePong() {
			meld := room.helper.NewMeld(meldCards[:3])
			tips = append(tips, OperateTip{Type: cardutil.PaohuziPong, Melds: []cardutil.PaohuziMeld{meld}})
			room.expectPongPlayer = other
			other.Timeout(func() { other.Pass() })
		}
		// 胡息
		if other.TryWin() != nil {
			room.expectWinPlayers[other.Id] = other
			tips = append(tips, OperateTip{Type: cardutil.PaohuziWin})
			other.Timeout(func() { other.Win() })
		}

		other.operateTips = tips
		log.Infof("discard tips %v", tips)
		if len(tips) > 0 {
			isPass = false
		}
		other.Prompt()
	}
	for i := 0; i < room.NumSeat(); i++ {
		other := room.GetPlayer(i)
		if t := other.GetKongType(drawCard); t != -1 {
			isPass = false

			room.expectKongPlayer = other
			other.Kong()
		}
	}

	room.Timing()
	if isPass == true {
		room.Turn()
	}
}

func (ply *PaohuziPlayer) GetKongType(c int) int {
	room := ply.Room()

	if cardutil.IsCardValid(c) == false {
		return -1
	}
	// 提龙
	for _, m := range ply.melds {
		if c == m.Cards[0] && m.Type == cardutil.PaohuziInvisibleTriplet {
			if ply.drawCard == -1 {
				return cardutil.PaohuziVisibleKong
			}
			return cardutil.PaohuziInvisibleKong
		}
		if c == m.Cards[0] && m.Type == cardutil.PaohuziVisibleTriplet {
			for i := 0; i < room.NumSeat(); i++ {
				if other := room.GetPlayer(i); other != nil && other.drawCard != -1 {
					return cardutil.PaohuziVisibleKong
				}
			}
		}
	}
	if ply.cards[c] == 3 {
		if ply.drawCard == -1 {
			return cardutil.PaohuziVisibleKong
		}
		return cardutil.PaohuziInvisibleKong
	}
	if ply.cards[c] == 4 {
		return cardutil.PaohuziInvisibleKong
	}

	return -1
}

// 杠
func (ply *PaohuziPlayer) Kong() {
	room := ply.Room()

	kongCard := room.lastCard
	typ := ply.GetKongType(kongCard)
	// 无效的杠
	if typ == -1 {
		return
	}
	if ply != room.expectKongPlayer {
		return
	}

	ply.kongCard = kongCard
	// 有玩家已胡牌
	if len(room.winPlayers) > 0 {
		return
	}
	// 有玩家胡牌
	if len(room.expectWinPlayers) > 0 {
		return
	}

	// OK
	meldCards := []int{kongCard, kongCard, kongCard, kongCard}
	if typ == cardutil.PaohuziInvisibleTriplet {
		meldCards = []int{kongCard, 0, 0, 0}
	}
	meld := room.helper.NewMeld(meldCards)

	ply.kongCard = -1
	ply.cards[kongCard] = 0
	if ply.cards[kongCard] > 0 {
		ply.melds = append(ply.melds, meld)
	} else {
		for k, m := range ply.melds {
			if m.Cards[0] == kongCard {
				ply.melds[k] = meld
				break
			}
		}
	}
	room.Broadcast("Kong", map[string]any{
		"Code": Ok,
		"Meld": meld,
		"UId":  ply.Id,
	})

	room.lastCard = -1
	if ply.TryWin() != nil && typ == cardutil.PaohuziInvisibleKong {
		ply.operateTips = []OperateTip{OperateTip{Type: cardutil.PaohuziWin}}

		room.kongPlayer = ply
		room.expectWinPlayers[ply.Id] = ply

		room.Timing()
		ply.Prompt()
		return
	}

	ply.AfterKong()
}

func (ply *PaohuziPlayer) AfterKong() {
	room := ply.Room()
	if other := room.discardPlayer; other != nil {
		other.drawCard = -1
	}
	room.kongPlayer = nil
	room.discardPlayer = nil

	var counter int
	for _, m := range ply.melds {
		switch m.Type {
		case cardutil.PaohuziVisibleKong, cardutil.PaohuziInvisibleKong:
			counter++
		}
	}

	// 第一条龙以后不用出牌
	if counter > 1 {
		room.discardPlayer = ply
		room.Turn()
	} else {
		room.expectDiscardPlayer = ply
		ply.tryDiscard()
	}
}

// 吃，最多带两个下去
func (ply *PaohuziPlayer) Chow(chowCards [][3]int) {
	log.Infof("player %d chow", ply.Id)
	room := ply.Room()
	if len(chowCards) == 0 {
		return
	}
	if _, ok := room.expectChowPlayers[ply.Id]; !ok {
		return
	}
	if ply.IsAbleChow() == false {
		return
	}
	chowCard := room.lastCard
	if room.helper.IsAbleChow(ply.GetSortedCards(), chowCards, chowCard) {
		return
	}

	// 把吃的牌放到第一个
	for i := 0; i < len(chowCards); i++ {
		tri := chowCards[i][:]
		sort.Ints(tri)
		for k := 0; k < len(tri); k++ {
			if tri[k] == chowCard {
				tri[0], tri[k] = tri[k], tri[0]
			}
		}
	}

	ply.operateTips = nil
	ply.chowCards = chowCards
	ply.StopTimer(service.TimerEventOperate)
	delete(room.expectWinPlayers, ply.Id)
	if room.expectPongPlayer == ply {
		room.expectPongPlayer = nil
	}
	if len(room.winPlayers) > 0 {
		ply.Pass()
		return
	}

	// 有玩家胡牌
	if len(room.expectWinPlayers) > 0 {
		return
	}
	// 有其他玩家碰牌
	if other := room.expectPongPlayer; other != nil {
		return
	}
	// 有其他玩家杠牌
	if other := room.expectKongPlayer; other != nil {
		return
	}

	// OK
	ply.chowCards = nil
	delete(room.expectChowPlayers, ply.Id)

	if ply.drawCard != -1 {
		// 下家
		nextId := (ply.GetSeatIndex() + 1) % room.NumSeat()
		next := room.GetPlayer(nextId)
		if _, ok := room.expectWinPlayers[next.Id]; !ok &&
			room.expectPongPlayer != next &&
			room.expectKongPlayer != next &&
			ply != next {
			next.Pass()
		}

		// 上家
		lastId := (ply.GetSeatIndex() - 1 + room.NumSeat()) % room.NumSeat()
		last := room.GetPlayer(lastId)
		if ply != last {
			last.unableChowCards[chowCard] = true
			last.unablePongCards[chowCard] = true
		}
	}

	log.Infof("player %d chow card %d", ply.Id, room.lastCard)

	melds := make([]cardutil.PaohuziMeld, 0, 1)
	for _, tri := range chowCards {
		melds = append(melds, room.helper.NewMeld(tri[:]))
	}
	data := map[string]any{
		"Code":  Ok,
		"Melds": melds,
		"UId":   ply.Id,
	}
	room.Broadcast("Chow", data)

	ply.drawCard = -1
	ply.operateTips = nil
	ply.StopTimer(service.TimerEventOperate)

	if other := room.discardPlayer; other != nil {
		other.drawCard = -1
	}
	room.discardPlayer = nil
	room.expectDiscardPlayer = ply

	room.Timing()
	ply.tryDiscard()
}

// 碰
func (ply *PaohuziPlayer) Pong() {
	log.Infof("player %d pong", ply.Id)
	room := ply.Room()
	pongCard := room.lastCard
	if room.expectPongPlayer != ply {
		return
	}
	if ply.IsAblePong() == false {
		return
	}

	ply.pongCard = pongCard
	ply.operateTips = nil
	ply.StopTimer(service.TimerEventOperate)
	delete(room.expectWinPlayers, ply.Id)
	delete(room.expectChowPlayers, ply.Id)
	if len(room.winPlayers) > 0 {
		ply.Pass()
		return
	}

	if len(room.expectWinPlayers) > 0 {
		return
	}

	// OK
	ply.pongCard = -1
	room.expectPongPlayer = nil
	for _, other := range room.expectChowPlayers {
		if _, ok := room.expectWinPlayers[other.Id]; !ok {
			other.Pass()
		}
	}

	// 增加一个刻子
	pongCards := []int{pongCard, pongCard, pongCard}
	if ply.drawCard != -1 {
		pongCards = []int{pongCard, 0, 0}
	}

	meld := room.helper.NewMeld(pongCards)
	ply.melds = append(ply.melds, meld)
	log.Infof("player %d pong card %d", ply.Id, pongCard)

	ply.cards[pongCard] -= 2
	room.Broadcast("Pong", map[string]any{"Code": Ok, "Meld": meld, "UId": ply.Id})

	room.lastCard = -1
	if ply.drawCard > 0 && ply.TryWin() != nil {
		ply.operateTips = []OperateTip{OperateTip{Type: cardutil.PaohuziWin}}

		room.pongPlayer = ply
		room.expectWinPlayers[ply.Id] = ply

		room.Timing()
		ply.Prompt()
		return
	}
	ply.AfterPong()
}

func (ply *PaohuziPlayer) AfterPong() {
	room := ply.Room()
	ply.operateTips = nil
	if other := room.discardPlayer; other != nil {
		other.drawCard = -1
	}

	room.pongPlayer = nil
	room.discardPlayer = nil
	room.expectDiscardPlayer = ply

	room.Timing()
	ply.tryDiscard()
}

// 胡牌
func (ply *PaohuziPlayer) Win() {
	room := ply.Room()
	if _, ok := room.expectWinPlayers[ply.Id]; !ok {
		return
	}

	delete(room.expectChowPlayers, ply.Id)
	if room.expectPongPlayer == ply {
		room.expectPongPlayer = nil
	}
	if other := room.expectKongPlayer; other != nil {
		other.kongCard = -1
		room.expectKongPlayer = nil
	}

	if other := room.expectPongPlayer; other != nil {
		if _, ok := room.expectWinPlayers[other.Id]; !ok {
			other.Pass()
		}
	}
	for _, other := range room.expectChowPlayers {
		if _, ok := room.expectWinPlayers[other.Id]; !ok {
			other.Pass()
		}
	}

	ply.operateTips = nil
	ply.WriteJSON("Win", map[string]any{"Code": Ok})

	// OK
	ply.StopTimer(service.TimerEventOperate)
	delete(room.expectWinPlayers, ply.Id)
	room.winPlayers = append(room.winPlayers, ply)
	room.OnWin()

}

func (ply *PaohuziPlayer) Discard(discardCard int) {
	log.Infof("player %d discard card %d", ply.Id, discardCard)
	PrintCards(ply.cards)

	room := ply.Room()
	if room.CardSet().IsCardValid(discardCard) == false {
		return
	}
	if cardNum := ply.cards[discardCard]; cardNum < 1 || cardNum > 2 {
		return
	}
	if room.expectDiscardPlayer != ply {
		return
	}
	// OK
	log.Infof("player %d discard card %d ok", ply.Id, discardCard)

	ply.cards[discardCard]--
	room.discardPlayer = ply
	room.lastCard = discardCard
	room.expectPongPlayer = nil
	room.expectDiscardPlayer = nil
	delete(room.expectChowPlayers, ply.Id)
	delete(room.expectWinPlayers, ply.Id)

	ply.drawCard = -1
	ply.discardNum++
	ply.operateTips = nil
	ply.unableChowCards[discardCard] = true
	ply.unablePongCards[discardCard] = true
	ply.unableWinCards[discardCard] = true
	ply.StopTimer(service.TimerEventOperate)
	room.Broadcast("Discard", map[string]any{"Code": Ok, "UId": ply.Id, "Card": discardCard})

	isPass := true
	meldCards := []int{discardCard, discardCard, discardCard, discardCard}
	for i := 0; i < room.NumSeat(); i++ {
		tips := make([]OperateTip, 0, 4)
		if other := room.GetPlayer(i); other != ply {
			// 吃
			if other.IsAbleChow() {
				samples := room.helper.TryChow(other.GetSortedCards(), discardCard)
				for _, sample := range samples {
					tips = append(tips, OperateTip{Type: cardutil.PaohuziChow, Melds: sample})
				}
				room.expectChowPlayers[other.Id] = other
				other.Timeout(func() { other.Pass() })
			}
			// 碰
			if other.IsAblePong() {
				meld := room.helper.NewMeld(meldCards[:3])
				tips = append(tips, OperateTip{Type: cardutil.PaohuziPong, Melds: []cardutil.PaohuziMeld{meld}})
				room.expectPongPlayer = other
				other.Timeout(func() { other.Pass() })
			}
			// 胡
			if other.TryWin() != nil {
				room.expectWinPlayers[other.Id] = other
				tips = append(tips, OperateTip{Type: cardutil.PaohuziWin})
				other.Timeout(func() { other.Win() })
			}

			other.operateTips = tips
			log.Infof("discard tips %v", tips)
			if len(tips) > 0 {
				isPass = false
			}
			other.Prompt()
		}
	}
	for i := 0; i < room.NumSeat(); i++ {
		if other := room.GetPlayer(i); other != ply {
			if t := other.GetKongType(discardCard); t != -1 {
				isPass = false

				room.expectKongPlayer = other
				other.Timeout(func() { other.Kong() })
			}
		}
	}

	room.Timing()
	if isPass == true {
		room.Turn()
	}
}

func (ply *PaohuziPlayer) Pass() {
	room := ply.Room()

	log.Debugf("player %d pass", ply.Id)
	// 玩家不可吃、碰、胡
	_, expectWin := room.expectWinPlayers[ply.Id]
	_, expectChow := room.expectChowPlayers[ply.Id]
	if !(expectWin ||
		expectChow ||
		room.expectPongPlayer == ply) {
		return
	}
	// OK
	ply.WriteJSON("Pass", map[string]any{"Code": Ok})

	ply.chowCards = nil
	ply.pongCard = -1
	ply.kongCard = -1

	ply.operateTips = nil
	delete(room.expectChowPlayers, ply.Id)
	delete(room.expectWinPlayers, ply.Id)
	if expectWin == true {
		ply.unableWinCards[room.lastCard] = true
	}

	if room.expectPongPlayer == ply {
		room.expectPongPlayer = nil
	}
	// 没人胡
	if len(room.expectWinPlayers) == 0 && len(room.winPlayers) == 0 {
		// 没人吃、碰、杠
		if other := room.expectKongPlayer; other != nil && other.kongCard != -1 {
			other.Kong()
		} else if other := room.expectPongPlayer; other != nil && other.pongCard != -1 {
			other.Pong()
		} else if other := room.pongPlayer; other != nil {
			other.AfterPong()
		} else if other := room.kongPlayer; other != nil {
			other.AfterKong()
		} else if len(room.expectChowPlayers) > 0 {
			for _, other := range room.expectChowPlayers {
				if other.chowCards != nil {
					other.Chow(other.chowCards)
				}
			}
		} else {
			if room.expectDiscardPlayer == ply {
				ply.tryDiscard()
			} else {
				room.Turn()
			}
		}
	} else {
		room.OnWin()
	}
}

// start 吃后顺子开始的牌
func (ply *PaohuziPlayer) IsAbleChow() bool {
	room := ply.Room()
	chowCard := room.lastCard
	if ply.isGiveUp || ply.isReadyHand {
		return false
	}
	if cardutil.IsCardValid(chowCard) == false {
		return false
	}
	// 上家出牌
	lastId := (ply.GetSeatIndex() + room.NumSeat() - 1) % room.NumSeat()
	if other := room.discardPlayer; other != nil && lastId != other.SeatId {
		return false
	}
	other := room.GetPlayer(lastId)
	for _, c := range other.discardHistory {
		if c == chowCard {
			return false
		}
	}
	for _, c := range ply.discardHistory {
		if c == chowCard {
			return false
		}
	}
	if _, ok := ply.unableChowCards[chowCard]; ok {
		return false
	}

	// 摸息
	samples := room.helper.TryChow(ply.GetSortedCards(), chowCard)
	log.Debug("check chow ", ply.GetSortedCards(), chowCard, samples)
	return len(samples) > 0
}

func (ply *PaohuziPlayer) IsAblePong() bool {
	room := ply.Room()

	pongCard := room.lastCard
	if ply.isGiveUp && ply.drawCard == -1 {
		return false
	}
	if ply.isReadyHand && ply.drawCard == -1 {
		return false
	}
	if cardutil.IsCardValid(pongCard) == false {
		return false
	}
	if ply.cards[pongCard] != 2 {
		return false
	}
	if ply.drawCard == -1 {
		for i := 0; i < room.NumSeat(); i++ {
			other := room.GetPlayer(i)
			for _, c := range other.discardHistory {
				if c == pongCard {
					return false
				}
			}
		}
		if _, ok := ply.unablePongCards[pongCard]; ok {
			return false
		}
	}

	return true
}

type WinOption struct {
	Melds []cardutil.PaohuziMeld
	Split []cardutil.PaohuziMeld
	Pair  int
}

func (ply *PaohuziPlayer) TryWin() (winOpt *WinOption) {
	room := ply.Room()
	winCard := room.lastCard
	if _, ok := ply.unableWinCards[winCard]; ok && ply.drawCard == -1 {
		return nil
	}

	maxScore := -1 // 考虑到无胡
	check := func(melds []cardutil.PaohuziMeld, cards []int) {
		splitOpt := room.helper.TryWin(cards)
		if splitOpt == nil {
			return
		}

		score := room.helper.Sum(melds) + room.helper.Sum(splitOpt.Melds)
		if maxScore < score {
			maxScore = score
			winOpt = &WinOption{
				Melds: melds,
				Split: splitOpt.Melds,
				Pair:  splitOpt.Pair,
			}
		}
	}

	var melds []cardutil.PaohuziMeld
	melds = append(melds, ply.melds...)
	// 22 (300) + 3 => 22 (3000) or 22 333 + 3 => 22 (3000)
	if room.pongPlayer == ply || room.kongPlayer == ply {
		cards := ply.GetSortedCards()
		check(melds, cards)
	} else {
		// 22 34 + 5 => 22 345
		ply.cards[winCard]++
		cards := ply.GetSortedCards()
		ply.cards[winCard]--
		check(melds, cards)

		if t := ply.GetKongType(winCard); t != -1 && ply.cards[winCard] == 0 {
			// 22 (333) + 3 => 22 (3333)
			// TODO
			for k, m := range melds {
				if m.Cards[0] == winCard {
					melds[k] = room.helper.NewMeld([]int{winCard, winCard, winCard, winCard})
				}
			}
			cards := ply.GetSortedCards()
			check(melds, cards)
			copy(melds, ply.melds)
		}
	}
	if maxScore == 0 {
		return
	}

	if maxScore < 10 {
		winOpt = nil
	}
	return
}

func (ply *PaohuziPlayer) Timeout(f func()) {
	ply.StopTimer(service.TimerEventOperate)

	room := ply.Room()
	roomType := room.GetRoomType()
	if roomType == service.RoomTypeScore {
		return
	}

	ply.AddTimer(service.TimerEventOperate, f, maxOperateTime)
}

func (ply *PaohuziPlayer) Prompt() {
	if len(ply.operateTips) == 0 {
		return
	}
	ply.WriteJSON("OperateTip", map[string]any{"Operate": ply.operateTips})
}

func (ply *PaohuziPlayer) tryDiscard() {
	ply.unableWinCards = make(map[int]bool)

	discardCard := -1
	for _, c := range cardutil.GetAllCards() {
		if cardNum := ply.cards[c]; cardNum < 3 && cardNum > 0 {
			discardCard = c
			break
		}
	}
	room := ply.Room()
	// 没牌可出，下家直接摸牌
	if discardCard == -1 || ply.isGiveUp {
		ply.isGiveUp = true
		room.lastCard = -1
		room.expectDiscardPlayer = nil
		room.discardPlayer = ply
		room.Turn()
	} else {
		ply.Timeout(func() { ply.Discard(discardCard) })
	}
}
