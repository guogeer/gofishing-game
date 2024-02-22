package internal

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"time"

	"gofishing-game/internal"
	"gofishing-game/internal/dbo"
	"gofishing-game/internal/gameutils"
	"gofishing-game/internal/pb"

	"github.com/guogeer/quasar/log"
	"github.com/guogeer/quasar/util"
	"google.golang.org/protobuf/proto"
)

var matchIPs = regexp.MustCompile(`[0-9.]+`)

type Cache struct {
	pb.UnimplementedCacheServer
}

func (cc *Cache) EnterGame(ctx context.Context, req *pb.EnterGameReq) (*pb.EnterGameResp, error) {
	db := dbo.Get()
	uid := req.Uid
	resp := &pb.EnterGameResp{}

	infoResp, err := cc.QueryUserInfo(ctx, &pb.QueryUserInfoReq{Uid: uid})
	if err != nil {
		return nil, err
	}

	binResp, err := cc.LoadBin(ctx, &pb.LoadBinReq{Uid: int32(uid)})
	if err != nil {
		log.Errorf("load player %d bin %v", uid, err)
	}
	resp.Bin = binResp.Bin
	resp.UserInfo = infoResp.Info

	db.QueryRow("select count(*) from `mail` where recv_uid=? and `status`=0", uid).Scan(&resp.NewMailNum)         // 邮件
	db.QueryRow("select expire_millis from charge_subscription where uid=?", uid).Scan(&resp.SubscriptionExpireTs) // 订阅
	return resp, nil
}
func (cc *Cache) QueryLoginParams(ctx context.Context, req *pb.QueryLoginParamsReq) (*pb.QueryLoginParamsResp, error) {
	db := dbo.Get()
	params := &pb.LoginParams{}
	err := db.QueryRow("select time_zone from user_info where id=?", req.Uid).Scan(&params.TimeZone) // 登陆参数
	return &pb.QueryLoginParamsResp{Params: params}, err
}

func (cc *Cache) AddLoginLog(ctx context.Context, req *pb.AddLoginLogReq) (*pb.EmptyResp, error) {
	uid := req.Uid
	ip := req.Ip
	mac := req.Mac
	imei := req.Imei
	imsi := req.Imsi
	chanId := req.ChanId
	ver := req.ClientVersion

	db := dbo.Get()
	now := time.Now()
	today := now.Format(internal.ShortDateFmt)
	tomorrow := now.Add(24 * time.Hour).Format(internal.ShortDateFmt)
	db.Exec("update online_log set ip=?,mac=?,imei=?,imsi=?,enter_chan_id=?,client_version=?,login_time=now() where uid=? and login_time between ? and ?", ip, mac, imei, imsi, chanId, ver, uid, today, tomorrow)
	db.Exec("insert into online_log(uid,ip,mac,imei,imsi,enter_chan_id,client_version,login_time) select ?,?,?,?,?,?,?,now() from dual where not exists (select 1 from online_log where uid=? and login_time between ? and ?)", uid, ip, mac, imei, imsi, chanId, ver, uid, today, tomorrow)
	return &pb.EmptyResp{}, nil
}

func (cc *Cache) Auth(ctx context.Context, req *pb.AuthReq) (*pb.AuthResp, error) {
	db := dbo.Get()
	mdb := mpool.Get()
	uid := req.Uid

	token := gameutils.CreateToken(int(uid))
	resp := &pb.AuthResp{Token: token}

	if uid > 0 {
		infoResp, err := cc.QueryUserInfo(ctx, &pb.QueryUserInfoReq{Uid: uid})
		if err != nil {
			return nil, err
		}
		// 最近的登陆版本
		db.QueryRow("select client_version,enter_time from online_log where uid=? order by id desc limit 1", uid).Scan(&resp.ClientVersion, &resp.LoginTime)
		// 绑定的平台
		plates := make([]string, 0, 4)
		rs, _ := db.Query("select plate from user_plate where uid=?", uid)
		for rs != nil && rs.Next() {
			var plate string
			rs.Scan(&plate)
			plates = append(plates, plate)
		}
		var plate string
		db.QueryRow("select plate from user_plate where uid=?", uid).Scan(&plate)
		plates = append(plates, plate)
		resp.LoginPlates = plates
		resp.ServerLocation = infoResp.Info.ServerLocation

		// IP白名单
		type clientVersion struct {
			AllowIPs string
		}
		chanrs, _ := mdb.Query("select json_value from gm_client_version where chan_id=?", infoResp.Info.ChanId)
		for chanrs != nil && chanrs.Next() {
			var cv clientVersion
			chanrs.Scan(dbo.JSON(&cv))
			// IP设置了白名单时仅允许名单内的IP访问
			allowIPs := matchIPs.FindAllString(cv.AllowIPs, -1)
			if len(allowIPs) > 0 && util.InArray(allowIPs, req.Ip) == 0 {
				resp.Reason = -3
			}
		}
	}
	return resp, nil
}

func (cc *Cache) LoadBin(ctx context.Context, req *pb.LoadBinReq) (*pb.LoadBinResp, error) {
	uid := req.Uid
	// load bin
	db := dbo.Get()
	rs, err := db.Query("select class,bin from user_bin where uid=?", uid)

	if err != nil {
		return nil, err
	}

	bin := &pb.UserBin{}
	val := reflect.ValueOf(bin)
	val = reflect.Indirect(val)
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		if field.CanInterface() {
			fieldv := field.Interface()
			if _, ok := fieldv.(proto.Message); ok && field.CanSet() {
				field.Set(reflect.New(field.Type().Elem()))
			}
		}
	}
	fields := map[string]proto.Message{
		"hall":    bin.Hall,
		"global":  bin.Global,
		"room":    bin.Room,
		"offline": bin.Offline,
	}

	for rs.Next() {
		var class string
		var buf []byte
		rs.Scan(&class, &buf)
		if field, ok := fields[class]; ok && len(buf) > 0 {
			proto.Unmarshal(buf, field)
		}
	}
	return &pb.LoadBinResp{Bin: bin}, nil
}

func (cc *Cache) SaveBin(ctx context.Context, req *pb.SaveBinReq) (*pb.EmptyResp, error) {
	uid := req.Uid
	bin := req.Bin
	db := dbo.Get()

	fields := map[string]proto.Message{
		"hall":   bin.Hall,
		"global": bin.Global,
		"room":   bin.Room,
	}

	tx, _ := db.Begin()
	curTime := time.Now().Format(internal.LongDateFmt)
	// NOTE any == nil需必须同时满足无类型&空值
	for key, field := range fields {
		if !reflect.ValueOf(field).IsNil() {
			buf, _ := proto.Marshal(field)
			tx.Exec("insert ignore into user_bin(uid,`class`,bin,update_time) values(?,?,?,?)", uid, key, buf, curTime)
			tx.Exec("update user_bin set bin=?, update_time=? where uid=? and `class`=?", buf, curTime, uid, key)
		}
	}

	mergeItems := make(map[int32]int64)
	if bin.Offline != nil {
		buf := make([]byte, 0, 4096)
		db.QueryRow("select bin from user_bin where uid=? and class=? for update", uid, "offline").Scan(&buf)
		offline := &pb.OfflineBin{}
		if len(buf) > 0 {
			proto.Unmarshal(buf, offline)
		}
		bin.Offline.Items = append(bin.Offline.Items, offline.Items...)
		for _, item := range bin.Offline.Items {
			mergeItems[item.Id] += item.Num
		}
		if len(bin.Offline.Items) > 0 {
			bin.Offline.Items = bin.Offline.Items[:0]
		}
		for id, num := range mergeItems {
			if num != 0 {
				bin.Offline.Items = append(bin.Offline.Items, &pb.NumericItem{Id: id, Num: num})
			}
		}

		curTime := time.Now().Format(internal.LongDateFmt)
		buf, _ = proto.Marshal(bin.Offline)
		tx.Exec("insert ignore into user_bin(uid,`class`,bin,update_time) values(?,?,?,?)", uid, "offline", buf, curTime)
		tx.Exec("update user_bin set bin=? where uid=? and `class`=?", buf, uid, "offline")
	}
	tx.Commit()
	return &pb.EmptyResp{}, nil
}

// 玩家访问房间
func (cc *Cache) Visit(ctx context.Context, req *pb.VisitReq) (*pb.EmptyResp, error) {
	db := dbo.Get()

	location := req.ServerLocation
	_, err := db.Exec("update user_info set server_location=? where (server_location = '' or ? = '') and id=?", location, location, req.Uid)
	return &pb.EmptyResp{}, err
}

// 注册账号
// uid = -3 会话已失效
// uid = -2 IP下同路账号过多
// uid = -1 注册时，账号已存在
// uid = 0  用户名或密码错误
// uid > 0  OK
func (cc *Cache) CreateAccount(ctx context.Context, req *pb.CreateAccountReq) (*pb.CreateAccountResp, error) {
	newInfo := req.Info
	if newInfo.Nickname == "" {
		newInfo.Nickname = "null"
	}

	db := dbo.Get()
	oldInfo := &pb.AccountInfo{Uid: newInfo.Uid, ChanId: newInfo.ChanId}
	// 快速登陆或第三方登陆
	if newInfo.OpenId != "" {
		newInfo.Phone = "" // 忽略手机号
		db.QueryRow("select uid from user_plate where open_id=? limit 1", newInfo.OpenId).Scan(&newInfo.Uid)
		userInfo, _ := cc.QueryUserInfo(ctx, &pb.QueryUserInfoReq{Uid: newInfo.Uid})
		util.DeepCopy(oldInfo, userInfo.Info)
	}

	fields := map[string]any{}
	// 更新昵称
	if oldInfo.Uid > 0 && newInfo.Nickname != "" && oldInfo.Nickname == "" {
		fields["nickname"] = newInfo.Nickname
	}
	// 更新头像
	if oldInfo.Uid > 0 && newInfo.Icon != "" {
		fields["icon"] = newInfo.Icon
	}
	// 更新头像
	if oldInfo.Uid > 0 && newInfo.PlateIcon != "" {
		fields["plate_icon"] = newInfo.PlateIcon
	}

	for k, v := range fields {
		db.Exec("update user_info set "+k+"=? where uid=?", v, oldInfo.Uid)
	}

	var newid int64
	if oldInfo.Uid == 0 {
		tx, _ := db.Begin()
		rs, err := tx.Exec("insert ignore into user_plate(uid,plate,open_id) values(0,?,?)", newInfo.Plate, newInfo.OpenId)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
		rowNum, err := rs.RowsAffected()
		if err != nil {
			tx.Rollback()
			return nil, err
		}
		if rowNum > 0 {
			createTime := time.Now().Format(internal.LongDateFmt)
			rs, err = tx.Exec("insert into user_info(nickname,sex,icon,plate_icon,email,ip,chan_id,client_version,mac,imei,imsi,create_time) values(?,?,?,?,?,?,?,?,?,?,?,?)",
				newInfo.Nickname, newInfo.Sex, newInfo.Icon, newInfo.Icon, newInfo.Email, newInfo.Ip,
				newInfo.ChanId, newInfo.ClientVersion, newInfo.Mac, newInfo.Imei, newInfo.Imsi, createTime,
			)
			if err != nil {
				tx.Rollback()
				return nil, err
			}
			newid, _ = rs.LastInsertId()
			oldInfo.Uid = int32(newid)
			tx.Exec("update user_plate set uid=? where open_id=?", newid, newInfo.OpenId)
		}
		tx.Commit()
	}

	return &pb.CreateAccountResp{
		Uid:       oldInfo.Uid,
		NewUserId: int32(newid),
		ChanId:    oldInfo.ChanId,
	}, nil
}

func (cc *Cache) UpdateLoginParams(ctx context.Context, req *pb.UpdateLoginParamsReq) (*pb.EmptyResp, error) {
	db := dbo.Get()
	db.Exec("update user_info set time_zone=? where id=?", req.Params.TimeZone, req.Uid)
	return &pb.EmptyResp{}, nil
}

func (cc *Cache) ClearAccount(ctx context.Context, req *pb.ClearAccountReq) (*pb.EmptyResp, error) {
	db := dbo.Get()
	db.Exec("delete from user_plate where uid=?", req.Uid)
	return &pb.EmptyResp{}, nil
}

func (cc *Cache) BindAccount(ctx context.Context, req *pb.BindAccountReq) (*pb.BindAccountResp, error) {
	db := dbo.Get()
	infoResp, err := cc.QuerySimpleUserInfo(ctx, &pb.QuerySimpleUserInfoReq{OpenId: req.ReserveOpenId})
	if err != nil {
		return nil, err
	}

	if infoResp.Info.Uid == 0 {
		return nil, fmt.Errorf("user with ReserveOpenId:%s is not existed", req.ReserveOpenId)
	}
	// 绑定的平台
	plates := make([]string, 0, 4)
	rs, _ := db.Query("select plate from user_plate where uid=?", infoResp.Info.Uid)
	for rs != nil && rs.Next() {
		var plate string
		rs.Scan(&plate)
		plates = append(plates, plate)
	}

	db.Exec("insert ignore user_plate(uid,plate,open_id) values(?,?,?)", 0, req.AddPlate, req.AddOpenId)
	db.Exec("update user_plate set uid=? where open_id=?", infoResp.Info.Uid, req.AddOpenId)

	response := &pb.BindAccountResp{}
	response.Uid = infoResp.Info.Uid
	response.Plates = plates
	return response, err
}
