package hall

import (
	"gofishing-game/internal/errcode"
	"gofishing-game/internal/gameutil"
	"gofishing-game/internal/pb"
	"gofishing-game/service"
)

type Item = gameutil.Item

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
	ply.checkClientVersion()
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
	gameutil.InitNilFields(bin.Hall)
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

func (ply *hallPlayer) OnAddItems(itemLog *gameutil.ItemLog) {
	ply.Player.OnAddItems(itemLog)
}

func (ply *hallPlayer) checkClientVersion() {
	w := GetWorld()
	clientVersion := ply.EnterReq().Auth.ClientVersion

	if ply.loginClientVersion == "" {
		ply.loginClientVersion = clientVersion
	}

	if ply.loginClientVersion == clientVersion {
		return
	}

	version := ply.loginClientVersion
	upgradeClientVersion := w.getClientVersion(version)
	if upgradeClientVersion == nil {
		return
	}

	if len(upgradeClientVersion.Reward) > 0 {
		ply.SetClientValue("UpdateClientReward", map[string]any{
			"ChangeLog": upgradeClientVersion.ChangeLog,
			"Reward":    upgradeClientVersion.Reward,
			"Version":   upgradeClientVersion.Version,
		})
	}
}
