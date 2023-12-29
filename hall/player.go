package hall

import (
	"gofishing-game/internal/errcode"
	"gofishing-game/internal/gameutils"
	"gofishing-game/internal/pb"
	"gofishing-game/service"
)

type hallPlayer struct {
	*service.Player

	mailObj            *mailObj
	loginClientVersion string
}

func (ply *hallPlayer) TryEnter() errcode.Error {
	return errcode.Ok
}

func (ply *hallPlayer) TryLeave() errcode.Error {
	return errcode.Ok
}

func (ply *hallPlayer) BeforeEnter() {
	ply.mailObj.BeforeEnter()
}

func (ply *hallPlayer) AfterEnter() {

}

func (ply *hallPlayer) BeforeLeave() {
}

func (ply *hallPlayer) OnClose() {
	ply.Player.OnClose()
}

func (ply *hallPlayer) Load(pdata any) {
	bin := pdata.(*pb.UserBin)
	gameutils.InitNilFields(bin.Hall)
	ply.loginClientVersion = bin.Hall.LoginClientVersion

	ply.mailObj.Load(pdata)
}

func (ply *hallPlayer) Save(pdata any) {
	bin := pdata.(*pb.UserBin)
	bin.Hall = &pb.HallBin{
		LoginClientVersion: ply.loginClientVersion,
	}
	ply.mailObj.Save(pdata)
}
