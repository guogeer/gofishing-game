package internal

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/url"
	"quasar/utils"
	"strings"
	"time"

	"gofishing-game/internal/errcode"
	"gofishing-game/internal/pb"
	"gofishing-game/internal/rpc"

	"github.com/guogeer/quasar/api"
	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
)

var (
	errNoGateway      = errcode.New("invalid_gateway", "no gateway service")
	errInvalidAddr    = errcode.New("invalid_addr", "invalid gateway address")
	errIPLimit        = errcode.New("ip_limit", "ip limit")
	errAuthFailed     = errcode.New("auth_failed", "account or passowrd not match")
	errAccountExisted = errcode.New("account_existed", "account existed")
)

func init() {
	codec := &api.CmdMessageCodec{}
	api.Add("POST", "/api/v1/login", login, (*loginReq)(nil)).SetCodec(codec)
	api.Add("POST", "/api/v1/clearAccount", clearAccount, (*clearAccountReq)(nil)).SetCodec(codec)
	api.Add("POST", "/api/v1/bindAccount", bindAccount, (*bindAccountReq)(nil)).SetCodec(codec)
	api.Add("POST", "/api/v1/queryQccount", queryAccount, (*queryAccountReq)(nil)).SetCodec(codec)
}

func Auth(req *pb.AccountInfo) (string, errcode.Error) {
	resp, err := rpc.CacheClient().Auth(context.Background(), &pb.AuthReq{Uid: req.Uid})
	if err != nil {
		return "", errcode.Retry
	}
	var e errcode.Error
	if resp.Reason == -2 {
		e = errIPLimit
	}
	return resp.Token, e
}

func GetBestGateway() (string, error) {
	data, err := cmd.Request("router", "C2S_GetBestGateway", nil)
	if err != nil {
		return "", err
	}

	response := struct{ Addr string }{}
	if err := json.Unmarshal(data, &response); err != nil {
		return "", err
	}
	return response.Addr, nil
}

type loginSession struct {
	Uid   int    `json:"uid"`
	NewId int    `json:"-"`
	Addr  string `json:"addr"`
	Token string `json:"token"`
	Name  string `json:"name"`
	IsReg bool   `json:"isReg"`
}

// 创建账号，若账号存在，返回token
// 2020-07-02 有头像的用户，需要定期更新平台信息
func CreateAccount(method string, account *pb.AccountInfo, params *pb.LoginParams) (*loginSession, error) {
	gw, err := GetBestGateway()
	if err != nil {
		return nil, errNoGateway
	}
	if _, _, err := net.SplitHostPort(gw); err != nil {
		return nil, errInvalidAddr
	}

	host := account.Ip
	if strings.Contains(host, ":") {
		host, _, _ = net.SplitHostPort(host)
	}

	if _, e := Auth(&pb.AccountInfo{Address: account.Address}); e != nil {
		return nil, e
	}
	account.Sex = account.Sex % 2

	if account.ChanId == "robot" {
		host = ""
	}
	if account.Nickname == "" {
		account.Nickname = GetRandName(int(account.Sex))
	}

	account.Ip = host
	// FB头像存在过期的问题，直接拉取头像放到本地
	// https://platform-lookaside.fbsbx.com/platform/profilepic/?asid=767409261099757&gaming_photo_type=unified_picture&ext=1662966674&hash=AeQi0wChjaF5NuLrIqo
	if u, err := url.Parse(account.Icon); err == nil {
		if u.Host == "platform-lookaside.fbsbx.com" {
			account.Icon, _ = saveFacebookIcon(u.Query().Get("asid")+".jpg", account.Icon)
			log.Debug("save facebook icon", account.Icon)
		}
	}
	if account.Icon != "" && (account.Plate == "facebook" || account.Plate == "google") {
		account.PlateIcon = account.Icon
	}

	resp, err := rpc.CacheClient().CreateAccount(context.Background(), &pb.CreateAccountReq{Info: account})
	if err != nil {
		return nil, err
	}
	uid, newid := int(resp.Uid), int(resp.NewUserId)
	//log.Debugf("uid %v CreateAccount newId %v", uid, newid)
	if uid <= 0 {
		return nil, errAuthFailed
	}
	if uid == -1 && method == "register" {
		return nil, errAccountExisted
	}
	if uid == -2 {
		return nil, errIPLimit
	}

	account.Uid = int32(uid)
	token, e := Auth(account)
	if e != nil {
		return nil, e
	}

	// Ok
	ss := &loginSession{
		Uid:   uid,
		NewId: newid,
		Token: token,
		Addr:  gw,
		Name:  account.Nickname,
		IsReg: newid > 0,
	}

	if account.ChanId == "robot" {
		return ss, nil
	}

	rpc.CacheClient().UpdateLoginParams(context.Background(), &pb.UpdateLoginParamsReq{Uid: int32(uid), Params: params})
	// 新注册账号
	if newid > 0 {
		var items []*pb.NumericItem
		for _, rowId := range config.Rows("item") {
			var id, num int
			config.Scan("item", rowId, "id,regNum", &id, &num)
			if num > 0 {
				pbItem := &pb.NumericItem{
					Id:      int32(id),
					Num:     int64(num),
					Balance: int64(num),
				}
				items = append(items, pbItem)
			}
		}
		//log.Debugf("%v CreateAccount %v", newid, s)
		rpc.CacheClient().AddSomeItem(context.Background(), &pb.AddSomeItemReq{Uid: int32(uid), Items: items})
		rpc.CacheClient().AddSomeItemLog(context.Background(), &pb.AddSomeItemLogReq{
			Uid:      int32(uid),
			Uuid:     utils.GUID(),
			Way:      "sys.new_user",
			Items:    items,
			CreateTs: time.Now().Unix(),
		})
	}
	rpc.CacheClient().AddLoginLog(context.Background(), &pb.AddLoginLogReq{Uid: int32(uid), LoginTime: time.Now().Format("2006-01-02 15:04:05")})
	return ss, nil
}

type loginReq struct {
	Plate         string  `json:"plate"`
	OpenId        string  `json:"openId"`
	Sex           int     `json:"sex"`
	Icon          string  `json:"icon"`
	Nickname      string  `json:"nickname"`
	TimeZone      float64 `json:"timeZone"`
	LocalTime     string  `json:"localTime"`
	Address       string  `json:"address"`
	Account       string  `json:"account"`
	Password      string  `json:"password"`
	ChanId        string  `json:"chanId"`
	Imsi          string  `json:"imsi"`
	Imei          string  `json:"imei"`
	Mac           string  `json:"mac"`
	Phone         string  `json:"phone"`
	OsVersion     string  `json:"osVersion"`
	NetMode       string  `json:"netMode"`
	ClientVersion string  `json:"clientVersion"`
	PhoneBrand    string  `json:"phoneBrand"`
	IosIDFA       string  `json:"iosIDFA"`
	Email         string  `json:"email"`
}

func login(c *api.Context, data any) (any, error) {
	args := data.(*loginReq)

	loginParams := &pb.LoginParams{}
	accountInfo := &pb.AccountInfo{Ip: c.Request.RemoteAddr}
	utils.DeepCopy(accountInfo, args)
	utils.DeepCopy(loginParams, args)
	ss, e := CreateAccount("login", accountInfo, loginParams)
	if e != nil {
		return nil, errors.New(e.Error())
	}
	return struct {
		*loginSession
		IP string `json:"ip"`
	}{loginSession: ss, IP: accountInfo.Ip}, nil
}

type clearAccountReq struct {
	Uid int32 `json:"uid"`
}

func clearAccount(c *api.Context, data any) (any, error) {
	args := data.(*clearAccountReq)
	_, err := rpc.CacheClient().ClearAccount(context.Background(), &pb.ClearAccountReq{Uid: args.Uid})
	return nil, err
}

type bindAccountReq struct {
	ReserveOpenId string `json:"reserveOpenId"`
	AddPlate      string `json:"addPlate"`
	AddOpenId     string `json:"addOpenId"`
	IsReward      bool   `json:"isReward"`
}

func bindAccount(c *api.Context, data any) (any, error) {
	args := data.(*bindAccountReq)

	// 切换账号账号登陆，游客绑定到Google/Facebook/Apple
	if args.AddOpenId == "" || args.ReserveOpenId == "" {
		log.Debugf("BindAccount => empty AddOpenId:%s or ReserveOpenId:%s", args.AddOpenId, args.ReserveOpenId)
		return nil, errors.New("empty addOpenId or reserveOpenId")
	}
	_, err := rpc.CacheClient().BindAccount(context.Background(), &pb.BindAccountReq{
		AddOpenId: args.AddOpenId,
	})

	return nil, err
}

type queryAccountReq struct {
	GuestOpenId string
	PlateOpenId string
}

type queryAccountResp struct {
	GuestUser struct {
		Uid   int `json:"uid"`
		Level int `json:"level"`
	} `json:"guestUser"`

	PlateUser struct {
		Uid   int `json:"uid"`
		Level int `json:"level"`
	} `json:"plateUser"`
}

func queryAccount(c *api.Context, data any) (any, error) {
	args := data.(*queryAccountReq)

	if args.GuestOpenId == "" || args.PlateOpenId == "" {
		log.Debugf("QueryAccount => empty GuestOpenId:%s or PlateOpenId:%s", args.GuestOpenId, args.PlateOpenId)
		return nil, errors.New("invalid guestOpenId or plateOpenId")
	}
	guestUser, _ := rpc.CacheClient().QuerySimpleUserInfo(context.Background(), &pb.QuerySimpleUserInfoReq{OpenId: args.GuestOpenId})
	plateUser, _ := rpc.CacheClient().QuerySimpleUserInfo(context.Background(), &pb.QuerySimpleUserInfoReq{OpenId: args.PlateOpenId})

	resp := queryAccountResp{}
	resp.GuestUser.Uid = int(guestUser.Info.Uid)
	resp.GuestUser.Level = int(guestUser.Info.Level)
	resp.PlateUser.Uid = int(plateUser.Info.Uid)
	resp.PlateUser.Level = int(plateUser.Info.Level)
	return cmd.M{}, nil
}
