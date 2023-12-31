package cache

import (
	"context"

	"gofishing-game/internal/dbo"
	"gofishing-game/internal/pb"

	"github.com/guogeer/quasar/util"
)

func (cc *Cache) QueryUserInfo(ctx context.Context, req *pb.QueryUserInfoReq) (*pb.QueryUserInfoResp, error) {
	db := dbo.Get()

	uid := req.Uid
	userInfo := &pb.UserInfo{
		Uid:   uid,
		Token: generateToken(int(uid)),
	}

	db.QueryRow("select account_info,create_time from user_info where uid=?", uid).Scan(dbo.JSON(userInfo), &userInfo.CreateTime)
	db.QueryRow("select open_id from user_plate where uid=?", uid).Scan(&userInfo.OpenId)
	return &pb.QueryUserInfoResp{Info: userInfo}, nil
}

func (cc *Cache) SetUserInfo(ctx context.Context, req *pb.SetUserInfoReq) (*pb.EmptyResp, error) {
	db := dbo.Get()
	db.Exec(`update user_info set sex=?,nickname=?,icon=?,email=? where uid=?`,
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
	util.DeepCopy(simpleInfo, userInfo)

	globalBin := &pb.GlobalBin{}
	db.QueryRow("select bin from user_bin where uid=? and `class`=?", simpleInfo.Uid, "global").Scan(dbo.PB(globalBin))
	simpleInfo.Level = globalBin.Level
	return &pb.QuerySimpleUserInfoResp{Info: simpleInfo}, nil
}
