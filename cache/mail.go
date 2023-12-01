package cache

import (
	"context"

	"gofishing-game/internal/dbo"
	"gofishing-game/internal/pb"

	"github.com/guogeer/quasar/log"
	"github.com/guogeer/quasar/util"
)

type Mail struct {
	Title         string
	Body          string
	Reward        string
	SendId        int32
	ClientVersion string
	EffectTime    []string
	RegTime       []string
	LoginTime     []string
}

// 邮件
func (cc *Cache) GetMailList(ctx context.Context, req *pb.MailReq) (*pb.MailResp, error) {
	db := dbo.Get()

	mails := make([]*pb.Mail, 0, 4)
	rs, _ := db.Query("select id,`type`,recv_uid,`data`,`status`,send_time from mail where `type`=? and recv_uid=? and `status`=0 order by id desc limit ?", req.Type, req.RecvId, req.Num)
	for rs != nil && rs.Next() {
		mail := &pb.Mail{}
		simpleMail := &Mail{}
		rs.Scan(&mail.Id, &mail.Type, &mail.RecvId, dbo.JSON(simpleMail), &mail.Status, &mail.SendTime)
		util.DeepCopy(mail, simpleMail)
		mails = append(mails, mail)
	}
	return &pb.MailResp{List: mails}, nil
}

func (cc *Cache) SendMail(ctx context.Context, req *pb.MailReq) (*pb.MailResp, error) {
	db := dbo.Get()

	mail := req.Mail
	simpleMail := &Mail{}
	util.DeepCopy(simpleMail, req.Mail)
	rs, err := db.Exec("insert into mail(`type`,recv_uid,`data`) values(?,?,?)", mail.Type, mail.RecvId, dbo.JSON(simpleMail))
	if err != nil {
		log.Error("send mail", err)
	}

	insertId := int64(-1)
	if rs != nil {
		insertId, _ = rs.LastInsertId()
	}
	return &pb.MailResp{Id: insertId}, nil
}

func (cc *Cache) OperateMail(ctx context.Context, req *pb.MailReq) (*pb.MailResp, error) {
	db := dbo.Get()

	// 邮件状态只能递增
	mail := &pb.Mail{}
	simpleMail := &Mail{}
	db.Exec("update mail set `status`=? where id=? and `status`=0 and recv_uid=?", req.Status, req.Id, req.RecvId)
	db.QueryRow("select id,`type`,recv_uid,`data`,`status`,send_time from mail where id=?", req.Id).Scan(&mail.Id, &mail.Type, &mail.RecvId, dbo.JSON(simpleMail), &mail.Status, &mail.SendTime)
	util.DeepCopy(mail, simpleMail)
	return &pb.MailResp{Mail: mail}, nil
}
