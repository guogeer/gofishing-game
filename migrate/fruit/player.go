package fruit

import (
	"gofishing-game/internal/gameutils"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"

	"github.com/guogeer/quasar/util"
)

// 座位上的玩家
type SeatPlayerInfo struct {
	service.SimpleUserInfo
	SeatId int

	winGold int64
}

type FruitPlayer struct {
	// winGold int64
	*service.Player
	fruitObj *FruitObj

	winGold int64
}

func (ply *FruitPlayer) AfterEnter() {
	ply.fruitObj.OnEnter()
}

func (ply *FruitPlayer) SitDown(seatId int) {
	code := Ok
	room := ply.Room()
	if room == nil {
		return
	}
	defer func() {
		info := SeatPlayerInfo{SeatId: seatId}
		util.DeepCopy(&info.SimpleUserInfo, &ply.UserInfo)
		room.Broadcast("SitDown", map[string]any{"Code": code, "Msg": code.String(), "Info": info})
	}()
	if ply.BagObj().NumItem(gameutils.ItemIdGold) < 100000 {
		code = FruitSitDownMoreGold
		return
	}
	if code = ply.RoomObj.TrySitDown(seatId); code != Ok {
		return
	}
}

func (ply *FruitPlayer) TryLeave() ErrCode {
	if ply.fruitObj.AllBet > 0 {
		return AlreadyBet
	}
	return Ok
}

func (ply *FruitPlayer) BeforeLeave() {
	if ply.SeatId != roomutils.NoSeat {
		ply.RoomObj.SitUp()
	}
}

/*
	func (ply *FruitPlayer) OnAddItem(itemId int, itemNum int64, guid, way string) {
		ply.RoomObj.OnAddItem(itemId, itemNum, guid, way)
	}
*/
func (ply *FruitPlayer) GameOver() {
	ply.fruitObj.GameOver()
}

func (ply *FruitPlayer) Room() *FruitRoom {
	if room := ply.RoomObj.CardRoom(); room != nil {
		return room.(*FruitRoom)
	}
	return nil
}
