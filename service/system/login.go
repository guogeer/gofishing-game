// 各种平台的参数

package system

import (
	"gofishing-game/internal/gameutil"
	"gofishing-game/internal/pb"
	"gofishing-game/service"
)

const actionKeyLogin = "login"

type loginObj struct {
	player *service.Player

	params *pb.LoginParams
}

func (obj *loginObj) BeforeEnter() {
}

func (obj *loginObj) Load(data any) {
	obj.params = obj.player.EnterReq().Data.LoginParams
	if obj.params == nil {
		obj.params = &pb.LoginParams{}
	}
	gameutil.InitNilFields(obj.params)
}

func (obj *loginObj) Save(data any) {
}

func (obj *loginObj) Params() *pb.LoginParams {
	return obj.params
}

func (obj *loginObj) IsBindPlate() bool {
	for _, plate := range obj.player.EnterReq().Auth.LoginPlates {
		if plate == "google" || plate == "facebook" {
			return true
		}
	}
	return false
}

func GetLoginObj(player *service.Player) *loginObj {
	return player.GetAction(actionKeyLogin).(*loginObj)
}

func newloginObj(player *service.Player) service.EnterAction {
	obj := &loginObj{player: player}
	player.DataObj().Push(obj)
	return obj
}

func init() {
	service.AddAction(actionKeyLogin, newloginObj)
}
