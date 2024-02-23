// 各种平台的参数

package system

import (
	"gofishing-game/service"
	"slices"
)

const actionKeyLogin = "login"

type loginObj struct {
	player *service.Player

	TimeZone      float32
	LoginPlates   []string
	ClientVersion string
}

func (obj *loginObj) BeforeEnter() {
}

func (obj *loginObj) Load(data any) {
	req := service.GetEnterQueue().GetRequest(obj.player.Id)
	if req != nil {
		obj.TimeZone = float32(req.LoginParamsResp.Params.TimeZone)
		obj.LoginPlates = append([]string{}, req.AuthResp.LoginPlates...)
	}
}

func (obj *loginObj) Save(data any) {
}

var sysPlates = []string{"robot", "guest"}

func (obj *loginObj) IsBindPlate() bool {
	for _, plate := range obj.LoginPlates {
		if slices.Index(sysPlates, plate) < 0 {
			return true
		}
	}
	return false
}

func (obj *loginObj) IsRobot() bool {
	return slices.Index(obj.LoginPlates, "robot") >= 0
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
