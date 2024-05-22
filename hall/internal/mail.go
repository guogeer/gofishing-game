package internal

import (
	"context"
	"time"

	"github.com/guogeer/quasar/utils"

	"gofishing-game/internal"
	"gofishing-game/internal/errcode"
	"gofishing-game/internal/gameutils"
	"gofishing-game/internal/pb"
	"gofishing-game/internal/rpc"
	"gofishing-game/service"
	"gofishing-game/service/system"

	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
)

const (
	_              = iota
	MailTypeSystem // 系统邮件
	MailTypeMass   // 群发邮件
)

const (
	maxMailNum = 1
)

const (
	_                = iota
	MailStatusNew    // 新邮件
	MailStatusLook   // 已查看
	MailStatusDraw   // 已领取
	MailStatusDelete // 已删除
	MailStatusClean  // 已清理
)

type Mail struct {
	Id            int64    `json:"id"`
	Type          int      `json:"type"`
	SendId        int      `json:"sendId"`
	RecvId        int      `json:"recvId"`
	Title         string   `json:"title"`
	Body          string   `json:"body"`
	Reward        string   `json:"reward"`
	Status        int      `json:"status"`
	SendTime      string   `json:"sendTime"`
	ClientVersion string   `json:"clientVersion"` // 指定版本
	EffectTime    []string `json:"-"`             // 有效时间
	LoginTime     []string `json:"loginTime"`     // 上次登陆时间
	RegTime       []string `json:"-"`             // 用户注册时间
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
	if service.GetPlayer(p.Id) != nil {
		massMails := obj.checkMassMails()
		p.mailObj.OnRecv(len(massMails))
	}

	p.Notify(cmd.M{"mails": obj.newMailNum})
}

func (obj *mailObj) IsMassMailValid(mail *Mail) bool {
	p := obj.player

	sendTime, _ := config.ParseTime(mail.SendTime)
	if obj.lastMassMail > sendTime.Unix() {
		return false
	}
	if mail.ClientVersion != "" && system.GetLoginObj(p.Player).ClientVersion != mail.ClientVersion {
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
			utils.DeepCopy(copyMail, mail)
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

		resp, err := rpc.CacheClient().QuerySomeMail(context.Background(),
			&pb.QuerySomeMailReq{RecvId: int32(uid), Type: MailTypeSystem, Num: maxMailNum})
		if err != nil {
			log.Errorf("look mail %v", err)
		}

		// 无奖励的邮件查看标记为已查看
		var emptyMailId int
		if resp != nil && len(resp.Mails) > 0 && resp.Mails[0].Reward == "" {
			emptyMailId = int(resp.Mails[0].Id)
		}
		if emptyMailId > 0 {
			rpc.CacheClient().OperateMail(context.Background(),
				&pb.OperateMailReq{Id: resp.Mails[0].Id, CurStatus: MailStatusNew, NewStatus: MailStatusLook})
		}

		rpc.OnResponse(func() {
			p := GetPlayer(uid)
			if p == nil {
				return
			}
			mails := make([]*Mail, 0, 8)
			for _, pbMail := range resp.Mails {
				mail := &Mail{}
				utils.DeepCopy(mail, pbMail)
				mails = append(mails, mail)
			}
			if emptyMailId > 0 {
				p.mailObj.OnRecv(-1)
			}
			p.WriteJSON("lookMails", cmd.M{
				"list": mails,
			})
		})
	}()
}

func (obj *mailObj) Load(pdata any) {
	bin := pdata.(*pb.UserBin)
	obj.lastMassMail = bin.Hall.LastMassMail
	obj.newMailNum = 0
	if data := service.GetEnterQueue().GetRequest(obj.player.Id).EnterGameResp; data != nil {
		obj.newMailNum = int(data.NewMailNum)
	}
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
		"mails": obj.newMailNum,
	})
}

// draw mail
func (obj *mailObj) Draw(id int64) {
	uid := obj.player.Id
	go func() {
		mailResp, err := rpc.CacheClient().QuerySomeMail(context.TODO(), &pb.QuerySomeMailReq{Id: id})
		if err != nil {
			return
		}
		if len(mailResp.Mails) == 0 {
			return
		}
		pbMail := mailResp.Mails[0]
		if pbMail.RecvId != int32(uid) {
			return
		}
		if pbMail.Status >= MailStatusDraw {
			return
		}

		operateResp, err := rpc.CacheClient().OperateMail(context.Background(),
			&pb.OperateMailReq{Id: id, CurStatus: pbMail.Status, NewStatus: MailStatusDraw})
		if err != nil {
			return
		}
		if operateResp.EffectRows != 1 {
			return
		}
		rpc.OnResponse(func() {
			p := GetPlayer(uid)
			if p == nil {
				return
			}

			var e errcode.Error
			if pbMail == nil || pbMail.Id == 0 {
				e = errcode.Retry
			}
			p.WriteErr("drawMail", e)
			if e != nil {
				return
			}
			// OK
			mail := &Mail{}
			utils.DeepCopy(mail, pbMail)
			p.BagObj().AddSomeItems(gameutils.ParseNumbericItems(mail.Reward), "mail_draw")
			p.mailObj.OnRecv(-1)
		})
	}()
}

func SyncSendMail(mail *Mail) int64 {
	pbMail := &pb.Mail{}
	utils.DeepCopy(pbMail, mail)
	resp, err := rpc.CacheClient().SendMail(context.Background(),
		&pb.SendMailReq{Mail: pbMail})
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
	utils.DeepCopy(mail, newMail)

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
	Id int64 `json:"id,omitempty"`
}

func init() {
	cmd.Bind("lookMails", funcLookMails, (*mailArgs)(nil))
	cmd.Bind("drawMail", funcDrawMail, (*mailArgs)(nil))
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
