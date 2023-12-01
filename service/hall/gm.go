package hall

import (
	"gofishing-game/internal/pb"

	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/log"
	"github.com/guogeer/quasar/util"
)

type gmArgs struct {
	Content string

	Users []int
	Mail  *pb.Mail
}

func init() {
	cmd.BindWithName("FUNC_SendMail", funcSendMail, (*gmArgs)(nil))
}

func funcSendMail(ctx *cmd.Context, data any) {
	args := data.(*gmArgs)

	mail := &Mail{}
	util.DeepCopy(mail, args.Mail)
	log.Info("send mail", args.Users, mail.EffectTime, mail.RegTime)
	if mail.Type == MailTypeMass {
		args.Users = []int{0}
	}

	for _, uid := range args.Users {
		mail.RecvId = uid
		SendMail(mail)
	}
}
