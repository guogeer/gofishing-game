package main

import (
	"context"

	"github.com/guogeer/quasar/v2/utils"

	"gofishing-game/cache/models"
	"gofishing-game/internal/dbo"
	"gofishing-game/internal/pb"
)

// 邮件
func (cc *Cache) QuerySomeMail(ctx context.Context, req *pb.QuerySomeMailReq) (*pb.QuerySomeMailResp, error) {
	db := dbo.Get()

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

	var mails []models.Mail
	db.Find(&mails).Where(where, params...)

	var pbMails []*pb.Mail
	for _, mail := range mails {
		pbMail := &pb.Mail{}
		utils.DeepCopy(pbMail, mail)
		pbMails = append(pbMails, pbMail)
	}
	return &pb.QuerySomeMailResp{Mails: pbMails}, nil
}

func (cc *Cache) SendMail(ctx context.Context, req *pb.SendMailReq) (*pb.SendMailResp, error) {
	db := dbo.Get()

	mail := models.Mail{
		Type:    int(req.Mail.Type),
		RecvUid: int(req.Mail.RecvId),
		SendUid: int(req.Mail.SendId),
		Title:   req.Mail.Title,
		Body:    req.Mail.Body,
	}
	result := db.Create(&mail)
	return &pb.SendMailResp{Id: int64(mail.Id)}, result.Error
}

func (cc *Cache) OperateMail(ctx context.Context, req *pb.OperateMailReq) (*pb.OperateMailResp, error) {
	db := dbo.Get()

	// 邮件状态只能递增
	result := db.Model(models.Mail{}).UpdateColumn("`status`=?", req.NewStatus).Where("id=? and `status`=?", req.Id, req.CurStatus)
	return &pb.OperateMailResp{EffectRows: int32(result.RowsAffected)}, result.Error
}
