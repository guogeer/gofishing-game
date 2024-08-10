package main

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"time"

	"github.com/guogeer/quasar/v2/utils"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"gofishing-game/cache/models"
	"gofishing-game/internal"
	"gofishing-game/internal/dbo"
	"gofishing-game/internal/gameutils"
	"gofishing-game/internal/pb"

	"github.com/guogeer/quasar/v2/log"
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
	db.Model(models.Mail{}).Where("recv_uid=? and `status`=0", uid).Count(&resp.NewMailNum) // 新邮件数
	return resp, nil
}
func (cc *Cache) QueryLoginParams(ctx context.Context, req *pb.QueryLoginParamsReq) (*pb.QueryLoginParamsResp, error) {
	db := dbo.Get()

	var userInfo models.UserInfo
	db.Where("id=?", req.Uid).Take(&userInfo) // 登陆参数
	return &pb.QueryLoginParamsResp{Params: &pb.LoginParams{TimeZone: float64(userInfo.TimeZone)}}, nil
}

// 近每天保存最近的一条登录日志
func (cc *Cache) AddLoginLog(ctx context.Context, req *pb.AddLoginLogReq) (*pb.EmptyResp, error) {
	db := dbo.Get()
	db.Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Model(models.OnlineLog{}).Create(&models.OnlineLog{
		Uid:           int(req.Uid),
		CurDate:       time.Now().Format(internal.ShortDateFmt),
		Ip:            req.Ip,
		Mac:           req.Mac,
		Imei:          req.Imei,
		Imsi:          req.Imsi,
		ChanId:        req.ChanId,
		ClientVersion: req.ClientVersion,
		LoginTime:     time.Now(),
	})
	return &pb.EmptyResp{}, nil
}

// TODO 匹配版本
func MatchClientVersion(channels []models.ClientVersion, version string) models.ClientVersion {
	return models.ClientVersion{}
}

func (cc *Cache) Auth(ctx context.Context, req *pb.AuthReq) (*pb.AuthResp, error) {
	db := dbo.Get()
	uid := req.Uid

	token := gameutils.CreateToken(int(uid))
	resp := &pb.AuthResp{Token: token}

	if uid > 0 {
		infoResp, err := cc.QueryUserInfo(ctx, &pb.QueryUserInfoReq{Uid: uid})
		if err != nil {
			return nil, err
		}
		// 最近的登陆版本
		onlineLog := models.OnlineLog{}
		db.Last(&onlineLog, uid)
		resp.LoginTime = onlineLog.LoginTime.Format(internal.LongDateFmt)
		// 绑定的平台
		var plates []models.UserPlate
		db.Find(&plates, uid)
		for _, row := range plates {
			resp.LoginPlates = append(resp.LoginPlates, row.Plate)
		}
		resp.ServerLocation = infoResp.Info.ServerLocation

		var clientVersions []models.ClientVersion
		db.Find(&clientVersions, "chan_id=?", infoResp.Info.ChanId)
		cv := MatchClientVersion(clientVersions, resp.ClientVersion)
		// IP设置了白名单时仅允许名单内的IP访问
		allowIPs := matchIPs.FindAllString(cv.AllowIP, -1)
		if len(allowIPs) > 0 && utils.InArray(allowIPs, req.Ip) == 0 {
			resp.Reason = -3
		}

	}
	return resp, nil
}

func (cc *Cache) LoadBin(ctx context.Context, req *pb.LoadBinReq) (*pb.LoadBinResp, error) {
	db := dbo.Get()

	var userBins []models.UserBin
	db.Where("uid=?", req.Uid).Find(&userBins)

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

	for _, item := range userBins {
		if field, ok := fields[item.Class]; ok && len(item.Bin) > 0 {
			proto.Unmarshal(item.Bin, field)
			dbo.PB(field).Scan(item.Bin)
		}
	}
	return &pb.LoadBinResp{Bin: bin}, nil
}

func (cc *Cache) SaveBin(ctx context.Context, req *pb.SaveBinReq) (*pb.EmptyResp, error) {
	db := dbo.Get()

	fields := map[string]proto.Message{
		"hall":   req.Bin.Hall,
		"global": req.Bin.Global,
		"room":   req.Bin.Room,
	}

	db.Transaction(func(tx *gorm.DB) error {
		// NOTE any == nil需必须同时满足无类型&空值
		for key, field := range fields {
			if !reflect.ValueOf(field).IsNil() {
				buf, _ := proto.Marshal(field)
				tx.Clauses(clause.OnConflict{UpdateAll: true}).Create(&models.UserBin{Uid: int(req.Uid), Class: key, Bin: buf})
			}
		}

		mergeItems := map[int32]int64{}
		if req.Bin.Offline != nil {
			var userBin models.UserBin
			tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("uid=? and class=?", req.Uid, "offline").Take(&userBin)

			newOffline := &pb.OfflineBin{}
			if len(userBin.Bin) > 0 {
				proto.Unmarshal(userBin.Bin, newOffline)
			}
			newOffline.Items = append(newOffline.Items, req.Bin.Offline.Items...)
			for _, item := range newOffline.Items {
				mergeItems[item.Id] += item.Num
			}
			if len(newOffline.Items) > 0 {
				newOffline.Items = newOffline.Items[:0]
			}
			for id, num := range mergeItems {
				if num != 0 {
					newOffline.Items = append(newOffline.Items, &pb.NumericItem{Id: id, Num: num})
				}
			}

			buf, _ := proto.Marshal(newOffline)
			tx.Clauses(clause.OnConflict{UpdateAll: true}).Create(&models.UserBin{Uid: int(req.Uid), Class: "offline", Bin: buf})
		}
		return nil
	})
	return &pb.EmptyResp{}, nil
}

// 玩家访问房间
func (cc *Cache) Visit(ctx context.Context, req *pb.VisitReq) (*pb.EmptyResp, error) {
	db := dbo.Get()

	err := db.Model(models.UserInfo{}).Where("(server_location = '' or ? = '') and id=?", req.ServerLocation, req.Uid).UpdateColumn("server_location", req.ServerLocation).Error
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

		var userPlate models.UserPlate
		db.Where("open_id=?", newInfo.OpenId).Take(&userPlate)
		newInfo.Uid = int32(userPlate.Uid)
		userInfo, _ := cc.QueryUserInfo(ctx, &pb.QueryUserInfoReq{Uid: newInfo.Uid})
		utils.DeepCopy(oldInfo, userInfo.Info)
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
		db.Transaction(func(tx *gorm.DB) error {
			userInfo := models.UserInfo{
				Nickname:      newInfo.Nickname,
				Sex:           int(newInfo.Sex),
				Icon:          newInfo.Icon,
				PlateIcon:     newInfo.Icon,
				Email:         newInfo.Email,
				Ip:            newInfo.Ip,
				ChanId:        newInfo.ChanId,
				ClientVersion: newInfo.ClientVersion,
				Mac:           newInfo.Mac,
				Imei:          newInfo.Imei,
				Imsi:          newInfo.Imsi,
			}
			if err := tx.Create(&userInfo).Error; err != nil {
				return err
			}

			oldInfo.Uid = int32(userInfo.Id)
			if err := tx.Create(&models.UserPlate{Uid: userInfo.Id, Plate: newInfo.Plate, OpenId: newInfo.OpenId}).Error; err != nil {
				return err
			}
			newid = int64(userInfo.Id)
			return nil
		})
	}

	return &pb.CreateAccountResp{
		Uid:       oldInfo.Uid,
		NewUserId: int32(newid),
		ChanId:    oldInfo.ChanId,
	}, nil
}

func (cc *Cache) UpdateLoginParams(ctx context.Context, req *pb.UpdateLoginParamsReq) (*pb.EmptyResp, error) {
	db := dbo.Get()
	db.Where("uid=?", req.Uid).Updates(models.UserInfo{TimeZone: float32(req.Params.TimeZone)})
	return &pb.EmptyResp{}, nil
}

func (cc *Cache) ClearAccount(ctx context.Context, req *pb.ClearAccountReq) (*pb.EmptyResp, error) {
	db := dbo.Get()
	db.Where("uid=?", req.Uid).Delete(models.UserPlate{})
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

	var userPlates []models.UserPlate
	db.Where("uid=?", infoResp.Info.Uid).Find(&userPlates)

	var plates []string
	for _, userPlate := range userPlates {
		plates = append(plates, userPlate.Plate)
	}

	db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&models.UserPlate{Uid: int(infoResp.Info.Uid), Plate: req.AddPlate, OpenId: req.AddOpenId})
	return &pb.BindAccountResp{Uid: infoResp.Info.Uid, Plates: plates}, err
}
