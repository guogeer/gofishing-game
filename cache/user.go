package cache

import (
	"context"

	"gofishing-game/internal/dbo"
	"gofishing-game/internal/pb"

	"github.com/guogeer/quasar/util"
)

func (cc *Cache) queryUserInfo(ctx context.Context, uid int32) (*pb.UserInfo, error) {
	db := dbo.Get()
	userInfo := &pb.UserInfo{
		UId:   uid,
		Token: generateToken(int(uid)),
	}

	db.QueryRow("select account_info,create_time from user_info where uid=?", uid).Scan(dbo.JSON(userInfo), &userInfo.CreateTime)
	db.QueryRow("select open_id from user_plate where uid=?", uid).Scan(&userInfo.OpenId)
	return userInfo, nil
}

func (cc *Cache) GetUserInfo(ctx context.Context, req *pb.Request) (*pb.UserInfo, error) {
	uid := req.UId
	return cc.queryUserInfo(ctx, uid)
}

func (cc *Cache) SetUserInfo(ctx context.Context, req *pb.EditableUserInfo) (*pb.Response, error) {
	db := dbo.Get()
	db.Exec(`update user_info set account_info=json_set(account_info,
		'$.Sex',?,
		'$.Nickname',?,
		'$.Icon',?,
		'$.Email',?
		) where uid=?`,
		req.Sex, req.Nickname, req.Icon, req.Email, req.UId)
	return &pb.Response{}, nil
}

func (cc *Cache) GetSimpleUserInfo(ctx context.Context, req *pb.Request) (*pb.SimpleUserInfo, error) {
	db := dbo.Get()
	simpleInfo := &pb.SimpleUserInfo{UId: req.UId}
	if req.OpenId != "" {
		db.QueryRow("select uid from user_plate where open_id=?", req.OpenId).Scan(&simpleInfo.UId)
	}
	userInfo, _ := cc.queryUserInfo(ctx, simpleInfo.UId)
	util.DeepCopy(simpleInfo, userInfo)

	globalBin := &pb.GlobalBin{}
	db.QueryRow("select bin from user_bin where uid=? and `class`=?", simpleInfo.UId, "global").Scan(dbo.PB(globalBin))
	simpleInfo.Level = globalBin.Level
	return simpleInfo, nil
}
