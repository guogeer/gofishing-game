package hall

import (
	"context"
	"time"

	"gofishing-game/internal"
	"gofishing-game/internal/errcode"
	"gofishing-game/internal/gameutil"
	"gofishing-game/internal/pb"
	"gofishing-game/internal/rpc"
	"gofishing-game/service"

	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
	"github.com/guogeer/quasar/util"
)

const (
	MailTypeSystem = iota // 系统邮件
	MailTypeMass          // 群发邮件
)

const (
	maxMailNum = 1
)

const (
	MailStatusNew    = iota // 新邮件
	MailStatusLook          // 已查看
	MailStatusDraw          // 已领取
	MailStatusDelete        // 已删除
	MailStatusClean         // 已清理
)

type Mail struct {
	Id            int64
	Type          int
	SendId        int
	RecvId        int
	Title         string
	Body          string
	Reward        string
	Status        int
	SendTime      string
	ClientVersion string   // 指定版本
	EffectTime    []string `json:"-"` // 有效时间
	LoginTime     []string // 上次登陆时间
	RegTime       []string `json:"-"` // 用户注册时间
}

type mailObj struct {
	player *hallPlayer

	lastMassMail  int64
	newMailNum    int
	lastEnterTime string
}

func newMailObj(p *hallPlayer) *mailObj {
	return &mailObj{player: p}
}

func (obj *mailObj) BeforeEnter() {
	p := obj.player
	if p.EnterReq().IsFirst() {
		massMails := obj.checkMassMails()
		p.mailObj.OnRecv(len(massMails))
	}

	p.Notify(cmd.M{"Mails": obj.newMailNum})
}

func (obj *mailObj) IsMassMailValid(mail *Mail) bool {
	p := obj.player

	sendTime, _ := config.ParseTime(mail.SendTime)
	if obj.lastMassMail > sendTime.Unix() {
		return false
	}
	if mail.ClientVersion != "" && p.EnterReq().Auth.ClientVersion != mail.ClientVersion {
		return false
	}
	if len(mail.LoginTime) > 1 &&
		!(obj.lastEnterTime > (mail.LoginTime[0]) &&
			obj.lastEnterTime < mail.LoginTime[0]) {
		return false
	}

	curDate := time.Now().Format(internal.ShortDateFmt)
	if len(mail.EffectTime) > 1 &&
		!(mail.EffectTime[0] > curDate && mail.EffectTime[1] < curDate) {
		return false
	}
	if len(mail.RegTime) > 1 &&
		!(p.CreateTime > mail.RegTime[0] && p.CreateTime < mail.RegTime[1]) {
		return false
	}
	return true
}

func (obj *mailObj) checkMassMails() []*Mail {
	p := obj.player
	massMails := make([]*Mail, 0, 4)
	for _, mail := range GetWorld().massMails {
		if obj.IsMassMailValid(mail) {
			copyMail := &Mail{}
			util.DeepCopy(copyMail, mail)
			copyMail.RecvId = p.Id
			copyMail.Type = MailTypeSystem
			massMails = append(massMails, copyMail)
		}
	}
	return massMails
}

// mail list
func (obj *mailObj) Look() {
	p := obj.player
	uid := p.Id
	massMails := obj.checkMassMails()
	if len(massMails) > 0 {
		obj.lastMassMail = time.Now().Unix()
	}

	go func() {
		for _, mail := range massMails {
			SyncSendMail(mail)
		}

		resp, err := rpc.CacheClient().GetMailList(context.Background(),
			&pb.MailReq{RecvId: int32(uid), Type: MailTypeSystem, Num: maxMailNum})
		if err != nil {
			log.Errorf("look mail %v", err)
		}

		// 无奖励的邮件查看标记为已查看
		var emptyMailId int
		if resp != nil && len(resp.List) > 0 && resp.List[0].Reward == "" {
			emptyMailId = int(resp.List[0].Id)
		}
		if emptyMailId > 0 {
			syncOperateMail(uid, resp.List[0].Id, MailStatusLook)
		}

		rpc.OnResponse(func() {
			p := GetPlayer(uid)
			if p == nil {
				return
			}
			mails := make([]*Mail, 0, 8)
			for _, pbMail := range resp.List {
				mail := &Mail{}
				util.DeepCopy(mail, pbMail)
				mails = append(mails, mail)
			}
			if emptyMailId > 0 {
				p.mailObj.OnRecv(-1)
			}
			p.WriteJSON("LookMails", cmd.M{
				"List": mails,
			})
		})
	}()
}

func (obj *mailObj) Load(pdata any) {
	bin := pdata.(*pb.UserBin)
	obj.lastMassMail = bin.Hall.LastMassMail
	obj.newMailNum = 0
	if data := obj.player.EnterReq().Data; data != nil {
		obj.newMailNum = int(data.NewMailNum)
	}
	obj.lastEnterTime = bin.Stat.LastEnterTime
}

func (obj *mailObj) Save(pdata any) {
	bin := pdata.(*pb.UserBin)
	bin.Hall.LastMassMail = obj.lastMassMail
}

func (obj *mailObj) OnRecv(n int) {
	if n == 0 {
		return
	}
	obj.newMailNum += n
	obj.player.Notify(cmd.M{
		"Mails": obj.newMailNum,
	})
}

// draw mail
func (obj *mailObj) Draw(id int64) {
	uid := obj.player.Id
	go func() {
		pbMail := syncOperateMail(uid, id, MailStatusDraw)
		rpc.OnResponse(func() {
			p := GetPlayer(uid)
			if p == nil {
				return
			}

			e := errcode.Ok
			if pbMail == nil || pbMail.Id == 0 {
				e = errcode.Retry
			}
			p.WriteJSON("DrawMail", e)
			if e != errcode.Ok {
				return
			}
			// OK
			mail := &Mail{}
			util.DeepCopy(mail, pbMail)
			p.ItemObj().AddSome(gameutil.ParseItems(mail.Reward), "mail_draw")
			p.mailObj.OnRecv(-1)
		})
	}()
}

func syncOperateMail(uid int, mailId int64, status int) *pb.Mail {
	resp, err := rpc.CacheClient().OperateMail(context.Background(),
		&pb.MailReq{RecvId: int32(uid), Id: mailId, Status: int32(status)})
	if err != nil {
		log.Errorf("player %d operate mail %d %d error %v", uid, mailId, status, err)
	}
	if resp != nil {
		return resp.Mail
	}
	return nil
}

func SyncSendMail(mail *Mail) int64 {
	pbMail := &pb.Mail{}
	util.DeepCopy(pbMail, mail)
	resp, err := rpc.CacheClient().SendMail(context.Background(),
		&pb.MailReq{Mail: pbMail})
	if err != nil {
		log.Errorf("send mail error %v %v", mail, err)
	}
	if resp == nil {
		return -1
	}
	log.Debug("sync send mail", resp.Id, mail.Title, mail.Body)
	return int64(resp.Id)
}

// 增加标签支持
// {wx} 客服微信
// {items} 物品列表
func SendMail(newMail *Mail) {
	mail := &Mail{}
	util.DeepCopy(mail, newMail)

	go func() {
		id := SyncSendMail(mail)
		rpc.OnResponse(func() {
			if p := GetPlayer(mail.RecvId); p != nil {
				p.mailObj.OnRecv(1)
			}
			w := GetWorld()
			if id >= 0 && mail.Type == MailTypeMass {
				mail.Id = id
				w.massMails = append(w.massMails, mail)
				for _, player := range service.GetAllPlayers() {
					p := player.GameAction.(*hallPlayer)
					if p.mailObj.IsMassMailValid(mail) {
						p.mailObj.OnRecv(1)
					}
				}
			}
		})
	}()
}

type mailArgs struct {
	Id int64
}

func init() {
	cmd.BindWithName("LookMails", funcLookMails, (*mailArgs)(nil))
	cmd.BindWithName("DrawMail", funcDrawMail, (*mailArgs)(nil))
}

func funcLookMails(ctx *cmd.Context, in any) {
	// args := in.(*mailArgs)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}

	ply.mailObj.Look()
}

func funcDrawMail(ctx *cmd.Context, in any) {
	args := in.(*mailArgs)
	ply := GetPlayerByContext(ctx)
	if ply == nil {
		return
	}

	ply.mailObj.Draw(args.Id)
}
