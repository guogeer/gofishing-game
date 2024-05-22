package main

import (
	"context"

	"github.com/guogeer/quasar/utils"

	"gofishing-game/internal/dbo"
	"gofishing-game/internal/pb"
)

func (cc *Cache) QueryUserInfo(ctx context.Context, req *pb.QueryUserInfoReq) (*pb.QueryUserInfoResp, error) {
	db := dbo.Get()

	uid := req.Uid
	userInfo := &pb.UserInfo{
		Uid: uid,
	}

	db.QueryRow("select chan_id,server_location,nickname,sex,icon,plate_icon,create_time from user_info where id=?", uid).Scan(
		&userInfo.ChanId, &userInfo.ServerLocation, &userInfo.Nickname, &userInfo.Sex, &userInfo.Icon, &userInfo.PlateIcon,
		&userInfo.CreateTime)
	db.QueryRow("select open_id from user_plate where uid=?", uid).Scan(&userInfo.OpenId)
	return &pb.QueryUserInfoResp{Info: userInfo}, nil
}

func (cc *Cache) SetUserInfo(ctx context.Context, req *pb.SetUserInfoReq) (*pb.EmptyResp, error) {
	db := dbo.Get()
	db.Exec(`update user_info set sex=?,nickname=?,icon=?,email=? where id=?`,
		req.Sex, req.Nickname, req.Icon, req.Email, req.Uid)
	return &pb.EmptyResp{}, nil
}

func (cc *Cache) QuerySimpleUserInfo(ctx context.Context, req *pb.QuerySimpleUserInfoReq) (*pb.QuerySimpleUserInfoResp, error) {
	db := dbo.Get()
	simpleInfo := &pb.SimpleUserInfo{Uid: req.Uid}
	if req.OpenId != "" {
		db.QueryRow("select uid from user_plate where open_id=?", req.OpenId).Scan(&simpleInfo.Uid)
	}
	userInfo, _ := cc.QueryUserInfo(ctx, &pb.QueryUserInfoReq{Uid: simpleInfo.Uid})
	utils.DeepCopy(simpleInfo, userInfo)

	globalBin := &pb.GlobalBin{}
	db.QueryRow("select bin from user_bin where uid=? and `class`=?", simpleInfo.Uid, "global").Scan(dbo.PB(globalBin))
	simpleInfo.Level = globalBin.Level
	return &pb.QuerySimpleUserInfoResp{Info: simpleInfo}, nil
}
