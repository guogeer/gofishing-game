package main

import (
	"context"

	"github.com/guogeer/quasar/v2/utils"

	"gofishing-game/cache/models"
	"gofishing-game/internal"
	"gofishing-game/internal/dbo"
	"gofishing-game/internal/pb"
)

func (cc *Cache) QueryUserInfo(ctx context.Context, req *pb.QueryUserInfoReq) (*pb.QueryUserInfoResp, error) {
	db := dbo.Get()

	var userInfo models.UserInfo
	db.Take(&userInfo, req.Uid)
	var userPlate models.UserPlate
	db.First(&userPlate, req.Uid)
	return &pb.QueryUserInfoResp{Info: &pb.UserInfo{
		Uid:            int32(req.Uid),
		ServerLocation: userInfo.ServerLocation,
		CreateTime:     userInfo.CreateTime.Format(internal.LongDateFmt),
		ChanId:         userInfo.ChanId,
		Sex:            int32(userInfo.Sex),
		Icon:           userInfo.Icon,
		PlateIcon:      userInfo.PlateIcon,
		Nickname:       userInfo.Nickname,
		OpenId:         userPlate.OpenId,
	}}, nil
}

func (cc *Cache) SetUserInfo(ctx context.Context, req *pb.SetUserInfoReq) (*pb.EmptyResp, error) {
	db := dbo.Get()
	db.Model(models.UserInfo{}).Updates(map[string]any{
		"sex":      req.Sex,
		"nickname": req.Nickname,
		"icon":     req.Icon,
		"email":    req.Email,
	})
	return &pb.EmptyResp{}, nil
}

func (cc *Cache) QuerySimpleUserInfo(ctx context.Context, req *pb.QuerySimpleUserInfoReq) (*pb.QuerySimpleUserInfoResp, error) {
	db := dbo.Get()
	simpleInfo := &pb.SimpleUserInfo{Uid: req.Uid}
	if req.OpenId != "" {
		var userPlate models.UserPlate
		db.Take(&userPlate, req.OpenId)
		simpleInfo.Uid = int32(userPlate.Uid)
	}
	userInfo, _ := cc.QueryUserInfo(ctx, &pb.QueryUserInfoReq{Uid: simpleInfo.Uid})
	utils.DeepCopy(simpleInfo, userInfo)

	var userBin models.UserBin
	db.Where("uid=? and `class`=?", simpleInfo.Uid, "global").Take(&userBin)
	var globalBin pb.GlobalBin
	dbo.PB(&globalBin).Scan(userBin.Bin)

	simpleInfo.Level = globalBin.Level
	return &pb.QuerySimpleUserInfoResp{Info: simpleInfo}, nil
}
