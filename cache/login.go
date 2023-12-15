package cache

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"reflect"
	"regexp"
	"time"

	"gofishing-game/internal/dbo"
	"gofishing-game/internal/env"
	"gofishing-game/internal/pb"

	"github.com/guogeer/quasar/log"
	"github.com/guogeer/quasar/util"
	"google.golang.org/protobuf/proto"
)

var matchIPs = regexp.MustCompile(`[0-9.]+`)

type Cache struct {
	pb.UnimplementedCacheServer
}

var tokenKey = "lolbye2023" + env.Config().Sign

func generateToken(uid int) string {
	sign := fmt.Sprintf("%s_%d", tokenKey, uid)
	sum := md5.Sum([]byte(sign))
	hexSum := hex.EncodeToString(sum[:])
	return hexSum
}

func (cc *Cache) EnterGame(ctx context.Context, req *pb.Request) (*pb.EnterGameResp, error) {
	db := dbo.Get()
	uid := req.UId
	resp := &pb.EnterGameResp{
		LoginParams: &pb.LoginParams{},
	}

	info, err := cc.queryUserInfo(ctx, uid)
	if err != nil {
		return nil, err
	}

	bin, err := cc.LoadBin(ctx, &pb.Request{UId: int32(uid)})
	if err != nil {
		log.Errorf("load player %d bin %v", uid, err)
	}
	resp.Bin = bin
	resp.UserInfo = info

	db.QueryRow("select count(*) from `mail` where recv_uid=? and `status`=0", uid).Scan(&resp.NewMailNum)         // 邮件
	db.QueryRow("select account_info from user_info where uid=?", uid).Scan(dbo.JSON(resp.LoginParams))            // 登陆参数
	db.QueryRow("select expire_millis from charge_subscription where uid=?", uid).Scan(&resp.SubscriptionExpireTs) // 订阅
	return resp, nil
}

func (cc *Cache) Enter(ctx context.Context, req *pb.AccountInfo) (*pb.Response, error) {
	uid := req.UId
	ip := req.IP
	mac := req.Mac
	imei := req.Imei
	imsi := req.Imsi
	addr := ""
	chanId := req.ChanId
	ver := req.Version

	db := dbo.Get()
	now := time.Now()
	today := now.Format("2006-01-02")
	tomorrow := now.Add(24 * time.Hour).Format("2006-01-02")
	db.Exec("update online_log set ip=?,mac=?,imei=?,imsi=?,address=?,enter_chan_id=?,client_version=?,enter_time=now() where uid=? and enter_time between ? and ?", ip, mac, imei, imsi, addr, chanId, ver, uid, today, tomorrow)
	db.Exec("insert into online_log(uid,ip,mac,imei,imsi,address,enter_chan_id,client_version,enter_time) select ?,?,?,?,?,?,?,?,now() from dual where not exists (select 1 from online_log where uid=? and enter_time between ? and ?)", uid, ip, mac, imei, imsi, addr, chanId, ver, uid, today, tomorrow)
	return &pb.Response{}, nil
}

func (cc *Cache) Auth(ctx context.Context, req *pb.AuthReq) (*pb.AuthResp, error) {
	db := dbo.Get()
	mdb := mpool.Get()
	uid := req.UId
	token := generateToken(int(uid))
	resp := &pb.AuthResp{Token: token}

	if uid > 0 {
		info, err := cc.queryUserInfo(ctx, uid)
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
		resp.ServerName = info.ServerName

		// IP白名单
		type clientVersion struct {
			AllowIPs string
		}
		chanrs, _ := mdb.Query("select json_value from gm_client_version where chan_id=?", info.ChanId)
		for chanrs != nil && chanrs.Next() {
			var cv clientVersion
			chanrs.Scan(dbo.JSON(&cv))
			// IP设置了白名单时仅允许名单内的IP访问
			allowIPs := matchIPs.FindAllString(cv.AllowIPs, -1)
			if len(allowIPs) > 0 && util.InArray(allowIPs, req.IP) == 0 {
				resp.Reason = -3
			}
		}
	}
	return resp, nil
}

func (cc *Cache) LoadBin(ctx context.Context, req *pb.Request) (*pb.UserBin, error) {
	uid := req.UId
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
		"stat":    bin.Stat,
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
	return bin, nil
}

func (cc *Cache) SaveBin(ctx context.Context, req *pb.SaveBinReq) (*pb.Response, error) {
	uid := req.UId
	bin := req.Bin
	db := dbo.Get()

	fields := map[string]proto.Message{
		"hall":   bin.Hall,
		"global": bin.Global,
		"room":   bin.Room,
		"stat":   bin.Stat,
	}

	tx, _ := db.Begin()
	// NOTE any == nil需必须同时满足无类型&空值
	for key, field := range fields {
		if !reflect.ValueOf(field).IsNil() {
			buf, _ := proto.Marshal(field)
			tx.Exec("insert ignore into user_bin(uid,`class`,bin) values(?,?,?)", uid, key, buf)
			tx.Exec("update user_bin set bin=? where uid=? and `class`=?", buf, uid, key)
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
				bin.Offline.Items = append(bin.Offline.Items, &pb.Item{Id: id, Num: num})
			}
		}

		buf, _ = proto.Marshal(bin.Offline)
		tx.Exec("insert ignore into user_bin(uid,`class`,bin) values(?,?,?)", uid, "offline", buf)
		tx.Exec("update user_bin set bin=? where uid=? and `class`=?", buf, uid, "offline")
	}
	tx.Commit()
	return &pb.Response{}, nil
}

// 玩家访问房间
func (cc *Cache) Visit(ctx context.Context, req *pb.VisitReq) (*pb.Response, error) {
	db := dbo.Get()
	_, err := db.Exec("update user_info set account_info=json_set(account_info,'$.SubId',?,'$.ServerName',?) where (ifnull(account_info->>'$.ServerName','') = '' or ? = '') and uid=?", req.SubId, req.ServerName, req.ServerName, req.UId)
	return &pb.Response{}, err
}

type dbAccountInfo struct {
	ChanId     string
	Nickname   string
	Sex        int
	Icon       string
	OS         string
	NetMode    string
	Version    string
	PhoneBrand string
	SubId      int32
	IP         string
	PlateIcon  string
}

// 注册账号
// UId = -3 会话已失效
// UId = -2 IP下同路账号过多
// UId = -1 注册时，账号已存在
// UId = 0  用户名或密码错误
// UId > 0  OK
func (cc *Cache) CreateAccount(ctx context.Context, req *pb.CreateAccountReq) (*pb.CreateAccountResp, error) {
	newInfo := req.AccountInfo
	if newInfo.Nickname == "" {
		newInfo.Nickname = "null"
	}

	db := dbo.Get()
	oldInfo := &pb.AccountInfo{UId: newInfo.UId, ChanId: newInfo.ChanId}
	// 快速登陆或第三方登陆
	if newInfo.OpenId != "" {
		newInfo.Phone = "" // 忽略手机号
		db.QueryRow("select uid from user_plate where open_id=? limit 1", newInfo.OpenId).Scan(&newInfo.UId)
		userInfo, _ := cc.queryUserInfo(ctx, newInfo.UId)
		util.DeepCopy(oldInfo, userInfo)
	}

	fields := map[string]any{}
	// 更新昵称
	if oldInfo.UId > 0 && newInfo.Nickname != "" && oldInfo.Nickname == "" {
		fields["Nickname"] = newInfo.Nickname
	}
	// 更新头像
	if oldInfo.UId > 0 && newInfo.Icon != "" {
		fields["Icon"] = newInfo.Icon
	}
	// 更新头像
	if oldInfo.UId > 0 && newInfo.PlateIcon != "" {
		fields["PlateIcon"] = newInfo.PlateIcon
	}

	for k, v := range fields {
		db.Exec("update user_info set account_info=json_set(account_info,'$."+k+"',?) where uid=?", v, oldInfo.UId)
	}

	var newid int64
	if oldInfo.UId == 0 {
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
			dbInfo := &dbAccountInfo{}
			util.DeepCopy(dbInfo, newInfo)
			rs, err = tx.Exec("insert into user_info(account_info) values(?)", dbo.JSON(dbInfo))
			if err != nil {
				tx.Rollback()
				return nil, err
			}
			newid, _ = rs.LastInsertId()
			oldInfo.UId = int32(newid)
			tx.Exec("update user_plate set uid=? where open_id=?", newid, newInfo.OpenId)
		}
		tx.Commit()
	}

	return &pb.CreateAccountResp{
		UId:    oldInfo.UId,
		NewId:  int32(newid),
		ChanId: oldInfo.ChanId,
	}, nil
}

func (cc *Cache) UpdateLoginParams(ctx context.Context, req *pb.LoginParamsReq) (*pb.Response, error) {
	db := dbo.Get()
	tx, _ := db.Begin()
	val := reflect.ValueOf(req.Params).Elem()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		if !(field.Kind() == reflect.String && field.IsZero()) && field.CanInterface() {
			tx.Exec("update user_info set account_info=json_set(account_info,'$."+val.Type().Field(i).Name+"',?) where uid=?", field.Interface(), req.UId)
		}
	}
	tx.Commit()
	return &pb.Response{}, nil
}

func (cc *Cache) ClearAccount(ctx context.Context, req *pb.AccountInfo) (*pb.Response, error) {
	db := dbo.Get()
	// db.Exec("update user_info set account_info='{}' where uid=?", req.UId)
	db.Exec("delete from user_plate where uid=?", req.UId)
	return &pb.Response{}, nil
}

func (cc *Cache) BindAccount(ctx context.Context, req *pb.BindAccountReq) (*pb.BindAccountResp, error) {
	db := dbo.Get()
	reserveUser, err := cc.GetSimpleUserInfo(ctx, &pb.Request{OpenId: req.ReserveOpenId})
	if err != nil {
		return nil, err
	}

	if reserveUser.UId == 0 {
		return nil, fmt.Errorf("user with ReserveOpenId:%s is not existed", req.ReserveOpenId)
	}
	// 绑定的平台
	plates := make([]string, 0, 4)
	rs, _ := db.Query("select plate from user_plate where uid=?", reserveUser.UId)
	for rs != nil && rs.Next() {
		var plate string
		rs.Scan(&plate)
		plates = append(plates, plate)
	}

	db.Exec("insert ignore user_plate(uid,plate,open_id) values(?,?,?)", 0, req.AddPlate, req.AddOpenId)
	db.Exec("update user_plate set uid=? where open_id=?", reserveUser.UId, req.AddOpenId)

	response := &pb.BindAccountResp{}
	response.UId = reserveUser.UId
	response.Plates = plates
	return response, err
}
