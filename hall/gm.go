package main

import (
	"github.com/guogeer/quasar/utils"

	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/log"
)

type gmArgs struct {
	Content string `json:"content,omitempty"`

	Users []int `json:"users,omitempty"`
	Mail  *Mail `json:"mail,omitempty"`
}

func init() {
	cmd.Bind("sendMail", funcSendMail, (*gmArgs)(nil))
}

func funcSendMail(ctx *cmd.Context, data any) {
	args := data.(*gmArgs)

	mail := &Mail{}
	utils.DeepCopy(mail, args.Mail)
	log.Info("send mail", args.Users, mail.EffectTime, mail.RegTime)
	if mail.Type == MailTypeMass {
		args.Users = []int{0}
	}

	for _, uid := range args.Users {
		mail.RecvId = uid
		SendMail(mail)
	}
}
