package main

import (
	"context"

	"github.com/guogeer/quasar/utils"

	"gofishing-game/internal/dbo"
	"gofishing-game/internal/pb"

	"github.com/guogeer/quasar/log"
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
func (cc *Cache) QuerySomeMail(ctx context.Context, req *pb.QuerySomeMailReq) (*pb.QuerySomeMailResp, error) {
	db := dbo.Get()

	var mails []*pb.Mail
	var params []any
	where := " where 1=1"
	if req.Id > 0 {
		where += " and `id`=?"
		params = append(params, req.Id)
	}
	if req.Type > 0 {
		where += " and `type`=?"
		params = append(params, req.Type)
	}
	if req.RecvId > 0 {
		where += " and `recv_id`=?"
		params = append(params, req.RecvId)
	}
	if req.Status > 0 {
		where += " and `status`=?"
		params = append(params, req.Status)
	}
	where += " order by id desc"
	if req.Num > 0 {
		where += " limit ?"
		params = append(params, req.Num)
	}

	rs, _ := db.Query("select id,`type`,recv_uid,`data`,`status`,send_time from mail"+where, params)
	for rs != nil && rs.Next() {
		mail := &pb.Mail{}
		simpleMail := &Mail{}
		rs.Scan(&mail.Id, &mail.Type, &mail.RecvId, dbo.JSON(simpleMail), &mail.Status, &mail.SendTime)
		utils.DeepCopy(mail, simpleMail)
		mails = append(mails, mail)
	}
	return &pb.QuerySomeMailResp{Mails: mails}, nil
}

func (cc *Cache) SendMail(ctx context.Context, req *pb.SendMailReq) (*pb.SendMailResp, error) {
	db := dbo.Get()

	mail := req.Mail
	simpleMail := &Mail{}
	utils.DeepCopy(simpleMail, req.Mail)
	rs, err := db.Exec("insert into mail(`type`,recv_uid,`data`) values(?,?,?)", mail.Type, mail.RecvId, dbo.JSON(simpleMail))
	if err != nil {
		log.Error("send mail", err)
	}

	insertId := int64(-1)
	if rs != nil {
		insertId, _ = rs.LastInsertId()
	}
	return &pb.SendMailResp{Id: insertId}, nil
}

func (cc *Cache) OperateMail(ctx context.Context, req *pb.OperateMailReq) (*pb.OperateMailResp, error) {
	db := dbo.Get()

	// 邮件状态只能递增
	result, err := db.Exec("update mail set `status`=? where id=? and `status`=?", req.NewStatus, req.Id, req.CurStatus)
	if err != nil {
		return nil, err
	}
	num, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	return &pb.OperateMailResp{EffectRows: int32(num)}, nil
}
