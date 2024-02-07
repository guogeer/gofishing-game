package roomutils

// 2018-01-10
// 房间内玩家

import (
	"encoding/json"
	"gofishing-game/internal/errcode"
	"gofishing-game/internal/gameutils"
	"gofishing-game/service"
	"strconv"
	"strings"

	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
)

const actionRoom = "Room"

type roomError struct {
	*errcode.BaseError
	SubId int `json:"subId,omitempty"`
}

var errTooMuchRoom = errcode.New("too_much_room", "too much room")
var errEnterOtherRoom = errcode.New("enter_other_room", "enter other room")

type roomEnterArgs struct {
	SubId int `json:"subId,omitempty"`
}

type RoomAction interface {
	// ChangeRoom()
	ChooseRoom() (*Room, errcode.Error)
}

func init() {
	service.AddAction(actionRoom, newRoomObj)
}

type RoomObj struct {
	seatIndex int

	player *service.Player
	room   *Room
}

func GetRoomObj(player *service.Player) *RoomObj {
	return player.GetAction(actionRoom).(*RoomObj)
}

func newRoomObj(player *service.Player) service.EnterAction {
	obj := &RoomObj{seatIndex: NoSeat, player: player}
	return obj
}

func (obj *RoomObj) GetSeat() int {
	return obj.seatIndex
}

func (obj *RoomObj) TryEnter() errcode.Error {
	if obj.room != nil {
		return nil
	}

	args := &roomEnterArgs{}
	enterReq := service.GetEnterQueue().GetRequest(obj.player.Id)
	json.Unmarshal(enterReq.RawData, args)

	serverLocation := enterReq.EnterGameResp.UserInfo.ServerLocation
	values := strings.Split(serverLocation+":", ":")
	curSubId, _ := strconv.Atoi(values[1])
	curServerId := values[0]
	if enterReq.IsOnline() {
		old := service.GetPlayer(obj.player.Id)
		if GetRoomObj(old).room.SubId != args.SubId {
			return &roomError{BaseError: errEnterOtherRoom, SubId: curSubId}
		}
	} else {
		if curServerId != "" && !(curSubId == args.SubId && service.GetServerId() == curServerId) {
			return &roomError{BaseError: errEnterOtherRoom, SubId: curSubId}
		}
	}

	// 比赛场先全部进入一个房间，比赛开始后再分配座位
	if roomAction, ok := obj.player.GameAction.(RoomAction); ok {
		room, e := roomAction.ChooseRoom()
		if e == nil {
			return e
		}
		obj.room = room
	}

	return nil
}

func (obj *RoomObj) BeforeEnter() {
	if obj.room == nil {
		return
	}

	p := obj.player
	room := obj.room
	if _, ok := room.allPlayers[p.Id]; !ok {
		room.allPlayers[p.Id] = p
	}
	if seat := room.GetEmptySeat(); seat != NoSeat {
		obj.SitDown(seat)
	}
}

func (obj *RoomObj) OnLeave() {
	if obj.room == nil {
		return
	}

	room := obj.room
	log.Infof("player %d leave room", obj.player.Id)
	delete(room.allPlayers, obj.player.Id)
	room.Broadcast("Leave", nil)
	if obj.seatIndex != NoSeat {
		obj.SitUp()
	}
	obj.room = nil
}

func (obj *RoomObj) Room() *Room {
	return obj.room
}

func (obj *RoomObj) CustomRoom() CustomRoom {
	if obj.room != nil {
		return obj.room.customRoom
	}
	return nil
}

func (obj *RoomObj) ChangeRoom() errcode.Error {
	player := obj.player
	subId, mySubId := -1, obj.room.SubId
	if room := obj.room; room != nil {
		mySubId = room.SubId
	}
	// 当前场次可换桌
	if _, ok := gSubGames[mySubId]; !ok {
		return errcode.Retry
	}
	for id := range gSubGames {
		// 机器人必须进入指定的场次
		if player.IsRobot && id != mySubId {
			continue
		}

		if (id-mySubId)*(id-mySubId) < (subId-mySubId)*(subId-mySubId) {
			subId = id
		}
	}
	if subId == -1 {
		return errcode.Retry
	}

	// 进入房间
	roomAction, ok := obj.player.GameAction.(RoomAction)
	if !ok {
		return errcode.Retry
	}
	room, e := roomAction.ChooseRoom()
	if e != nil {
		return e
	}

	// OK
	obj.OnLeave()
	obj.room = room
	return nil
}
func (obj *RoomObj) Choose(subId int) (*Room, errcode.Error) {
	// 优先分配座位未满的房间，最后分配座位坐满可观战的房间
	var maxPlayerNum, robotNum, maxRoomNum int
	var needItemStr, limitItemStr string
	config.Scan("room", subId, "maxPlayerNum,robotNumPerRoom,roomNum,needItems,limitItems",
		&maxPlayerNum, &robotNum, &maxRoomNum, &needItemStr, &limitItemStr,
	)

	sub, ok := gSubGames[subId]
	if !ok {
		return nil, errcode.Retry
	}
	if maxRoomNum > 0 && len(sub.rooms) >= maxRoomNum {
		log.Errorf("server %s sub_id %d room num %d limit %d error: %v", service.GetServerName(), subId, len(sub.rooms), maxRoomNum, errTooMuchRoom)
		return nil, errTooMuchRoom
	}
	bagObj := obj.player.BagObj()

	var needItemId int
	for _, item := range gameutils.ParseNumbericItems(needItemStr) {
		if item.GetNum() > bagObj.NumItem(item.GetId()) {
			needItemId = item.GetId()
		}
	}
	if needItemId > 0 {
		return nil, errcode.MoreItem(needItemId)
	}

	var tooMuchItemId int
	for _, item := range gameutils.ParseNumbericItems(needItemStr) {
		if item.GetNum() > bagObj.NumItem(item.GetId()) {
			needItemId = item.GetId()
		}
	}
	if tooMuchItemId > 0 {
		return nil, errcode.TooMuchItem(needItemId)
	}

	// 重复利用已有的房间
	var freeRoom *Room
	for _, room := range sub.rooms {
		if len(room.GetAllPlayers()) == 0 {
			freeRoom = room
			break
		}
	}

	if freeRoom == nil {
		freeRoom = service.GetWorld().(RoomWorld).NewRoom(subId)
		sub.rooms = append(sub.rooms, freeRoom)
	}
	return freeRoom, nil
}

func (obj *RoomObj) SitDown(seatIndex int) errcode.Error {
	room := obj.room
	if obj.seatIndex != NoSeat {
		return errcode.New("sit_down_already", "sit down already")
	}
	if seatIndex < 0 || seatIndex >= len(room.seatPlayers) {
		return errcode.Retry
	}
	if room.seatPlayers[seatIndex] != nil {
		return errcode.New("seat_had_player", "seat had player already")
	}
	obj.seatIndex = seatIndex
	room.seatPlayers[seatIndex] = obj.player
	return nil
}

func (obj *RoomObj) SitUp() errcode.Error {
	room := obj.room
	if obj.seatIndex == NoSeat {
		return errcode.Retry
	}
	if sp := room.seatPlayers[obj.seatIndex]; !(sp != nil && sp == obj.player) {
		return errcode.Retry
	}

	obj.seatIndex = NoSeat
	room.seatPlayers[obj.seatIndex] = nil
	return nil
}
