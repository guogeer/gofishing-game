package login

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	"gofishing-game/internal/env"
	"gofishing-game/internal/errcode"
	"gofishing-game/internal/pb"
	"gofishing-game/internal/rpc"

	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
	"github.com/guogeer/quasar/util"
)

var (
	errNoGateway      = errcode.New("invalid_gateway", "no gateway service")
	errInvalidAddr    = errcode.New("invalid_addr", "invalid gateway address")
	errIPLimit        = errcode.New("ip_limit", "ip limit")
	errAuthFailed     = errcode.New("auth_failed", "account or passowrd not match")
	errAccountExisted = errcode.New("account_existed", "account existed")
)

var errHTTPRequest = errors.New("invalid request method")

type Args struct {
	Address string
	ChanId  string `json:"chan_id"`
}

func init() {
	http.HandleFunc("/login", Login)
	http.HandleFunc("/clear_account", ClearAccount)
	http.HandleFunc("/bind_account", BindAccount)
	http.HandleFunc("/query_account", QueryAccount)
}

type Request struct {
	pb.LoginParams
	pb.AccountInfo
	LocalTime      string // 当地时间
	InstallReferer string

	IsUpdate bool // 更新账号信息
}

func writeJSON(w http.ResponseWriter, i any) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	b, err := cmd.Encode("", i)
	if err != nil {
		return
	}
	w.Write(b)
}

func readRequest(name string, r *http.Request, args any) error {
	log.Debugf("request method %s url %s", r.Method, r.URL)
	if r.Method == "GET" {
		return errHTTPRequest
	}
	message, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	r.Body.Close()
	log.Debugf("request body %s", message)

	sign := r.Header.Get("Sign")
	pkg, err := cmd.Decode(message)

	// log.Debugf("request url: %s body: %s header-sign:%s", name, message, sign)
	if err == cmd.ErrInvalidSign && sign == env.Config().Sign {
		err = nil
		if ip := r.Header.Get("Clientip"); ip != "" {
			r.RemoteAddr = ip
		}
	}
	if err != nil {
		return err
	}
	data := pkg.Data
	if err = json.Unmarshal(data, args); err != nil {
		return err
	}
	return nil
}

func Auth(req *pb.AccountInfo) (string, errcode.Error) {
	resp, err := rpc.CacheClient().Auth(context.Background(), &pb.AuthReq{UId: req.UId})
	if err != nil {
		return "", errcode.Retry
	}
	e := errcode.Ok
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

type Session struct {
	UId   int
	NewId int `json:"-"`
	Addr  string
	Token string
	Name  string
	IsReg bool
}

// 创建账号，若账号存在，返回token
// 2020-07-02 有头像的用户，需要定期更新平台信息
func CreateAccount(req *Request) (ss Session, e errcode.Error) {
	gw, err := GetBestGateway()
	if err != nil {
		e = errNoGateway
		return
	}
	if _, _, err := net.SplitHostPort(gw); err != nil {
		e = errInvalidAddr
		return
	}
	host := req.IP
	if strings.Contains(host, ":") {
		host, _, _ = net.SplitHostPort(host)
	}
	if req.InstallReferer != "" {
		var plateResponse struct{ ChanId string }
		if err := requestPlate("/plate/parse_install_referer", cmd.M{"InstallReferer": req.InstallReferer}, &plateResponse); err != nil {
			log.Debugf("parse install refer error: %v", err)
		}
		if plateResponse.ChanId != "" {
			req.ChanId = plateResponse.ChanId
		}
	}

	_, e = Auth(&req.AccountInfo)
	if e != errcode.Ok {
		return
	}
	req.Sex = req.Sex % 2

	if req.ChanId == "robot" {
		host = ""
	}
	if req.Nickname == "" {
		req.Nickname = GetRandName(int(req.Sex))
	}

	req.IP = host
	// FB头像存在过期的问题，直接拉取头像放到本地
	// https://platform-lookaside.fbsbx.com/platform/profilepic/?asid=767409261099757&gaming_photo_type=unified_picture&ext=1662966674&hash=AeQi0wChjaF5NuLrIqo
	if u, err := url.Parse(req.Icon); err == nil {
		if u.Host == "platform-lookaside.fbsbx.com" {
			req.Icon, _ = saveFacebookIcon(u.Query().Get("asid")+".jpg", req.Icon)
			log.Debug("save facebook icon", req.Icon)
		}
	}
	if req.Icon != "" && (req.Plate == "facebook" || req.Plate == "google") {
		req.AccountInfo.PlateIcon = req.Icon
	}

	resp, err := rpc.CacheClient().CreateAccount(context.Background(), &pb.CreateAccountReq{
		AccountInfo: &req.AccountInfo,
	})
	if err != nil {
		e = errcode.Retry
		return
	}
	uid, newid := int(resp.UId), int(resp.NewId)
	//log.Debugf("uid %v CreateAccount newId %v", uid, newid)
	if uid <= 0 {
		e = errAuthFailed
	}
	if uid == -1 && req.Method == "register" {
		e = errAccountExisted
	}
	if uid == -2 {
		e = errIPLimit
	}
	if e != errcode.Ok {
		return
	}

	req.UId = int32(uid)
	token, e := Auth(&req.AccountInfo)
	if e != errcode.Ok {
		return
	}

	// Ok
	ss = Session{
		UId:   uid,
		NewId: newid,
		Token: token,
		Addr:  gw,
		Name:  req.Nickname,
		IsReg: newid > 0,
	}

	if req.ChanId == "robot" {
		return
	}

	rpc.CacheClient().UpdateLoginParams(context.Background(), &pb.LoginParamsReq{UId: int32(uid), Params: &req.LoginParams})
	// 新注册账号
	if newid > 0 {
		r := &pb.ItemReq{
			UId:  int32(uid),
			Uuid: util.GUID(),
			Way:  "sys.new_user",
		}
		for _, rowId := range config.Rows("item") {
			var id, num int
			config.Scan("item", rowId, "ShopID,RegNum", &id, &num)
			if num > 0 {
				pbItem := &pb.Item{
					Id:      int32(id),
					Num:     int64(num),
					Balance: int64(num),
				}
				r.Items = append(r.Items, pbItem)
			}
		}
		//log.Debugf("%v CreateAccount %v", newid, s)
		rpc.CacheClient().AddSomeItem(context.Background(), r)
		rpc.CacheClient().AddSomeItemLog(context.Background(), r)
	}
	rpc.CacheClient().Enter(context.Background(), &req.AccountInfo)
	return
}

type loginResponse struct {
	errcode.Error
	Session
	IP string
}

func Login(w http.ResponseWriter, r *http.Request) {
	var req Request
	var ss Session
	if err := readRequest("Login", r, &req); err != nil {
		log.Warn("read request on login error", err)
		return
	}

	e := errcode.Ok
	defer func() { writeJSON(w, &loginResponse{Error: e, Session: ss, IP: req.IP}) }()

	req.Method = "login"
	req.IP = r.RemoteAddr
	//log.Debugf("Login CreateAccount %v", r.RemoteAddr)
	if ss, e = CreateAccount(&req); e != errcode.Ok {
		return
	}
}

func ClearAccount(w http.ResponseWriter, r *http.Request) {
	var req Request
	err := readRequest("ClearAccount", r, &req)
	if err != nil {
		return
	}
	ecode := errcode.Ok
	accountInfo := &req.AccountInfo
	rpc.CacheClient().ClearAccount(context.Background(), accountInfo)
	writeJSON(w, ecode)
}

type bindAccountRequest struct {
	ReserveOpenId string
	AddPlate      string
	AddOpenId     string
	IsReward      bool
}

func BindAccount(w http.ResponseWriter, r *http.Request) {
	var req bindAccountRequest
	if err := readRequest("BindPlate", r, &req); err != nil {
		log.Warn("read request BindAccount error", err)
		return
	}

	// 切换账号账号登陆，游客绑定到Google/Facebook/Apple
	if req.AddOpenId == "" || req.ReserveOpenId == "" {
		log.Debugf("BindAccount => empty AddOpenId:%s or ReserveOpenId:%s", req.AddOpenId, req.ReserveOpenId)
		return
	}
	_, err := rpc.CacheClient().BindAccount(context.Background(), &pb.BindAccountReq{
		AddOpenId: req.AddOpenId,
	})
	e := errcode.Ok
	if err != nil {
		e = errcode.Retry
	}

	writeJSON(w, e)
}

type queryAccountRequest struct {
	GuestOpenId string
	PlateOpenId string
}

type queryAccountResponse struct {
	GuestUser, PlateUser struct {
		UId   int
		Level int
	}
}

func QueryAccount(w http.ResponseWriter, r *http.Request) {
	var req queryAccountRequest
	var resp queryAccountResponse
	if err := readRequest("QueryAccount", r, &req); err != nil {
		log.Warn("read request QueryAccount error", err)
		return
	}
	if req.GuestOpenId == "" || req.PlateOpenId == "" {
		log.Debugf("QueryAccount => empty GuestOpenId:%s or PlateOpenId:%s", req.GuestOpenId, req.PlateOpenId)
		return
	}
	guestUser, _ := rpc.CacheClient().GetSimpleUserInfo(context.Background(), &pb.Request{OpenId: req.GuestOpenId})
	plateUser, _ := rpc.CacheClient().GetSimpleUserInfo(context.Background(), &pb.Request{OpenId: req.PlateOpenId})
	resp.GuestUser.UId = int(guestUser.UId)
	resp.GuestUser.Level = int(guestUser.Level)
	resp.PlateUser.UId = int(plateUser.UId)
	resp.PlateUser.Level = int(plateUser.Level)
	writeJSON(w, resp)
}
