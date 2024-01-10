package service

import (
	"container/list"
	"context"
	"net"
	"time"

	"gofishing-game/internal/errcode"
	"gofishing-game/internal/pb"
	"gofishing-game/internal/rpc"

	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/log"
)

const maxLoginQueue = 99 // 同时处理的登陆请求
var (
	errNoResponse     = errcode.New("no_response", "no response")
	errEnterOtherGame = errcode.New("enter_other_game", "重复登录")
)

type enterArgs struct {
	Uid         int    `json:"uid,omitempty"`
	Token       string `json:"token,omitempty"`
	LeaveServer string `json:"leaveServer,omitempty"`
	SubId       int    `json:"subId,omitempty"`
}

func init() {
	cmd.BindWithName("FUNC_Leave", funcAutoLeave, (*enterArgs)(nil))
	cmd.BindWithName("Enter", funcEnter, (*enterArgs)(nil))
}

type enterRequest struct {
	Auth        *pb.AuthResp
	Data        *pb.EnterGameResp `json:"-"`
	LoginParams *pb.LoginParams   `json:"-"`

	UId         int
	SubId       int // 房间有场次概念
	Token       string
	LeaveServer string // 离开旧场景进入新的场景
	ServerName  string

	e          *list.Element
	session    *cmd.Session
	expireTime time.Time // 过期时间
	oldSubId   int
	clientIP   string
	startTime  time.Time
	isOnline   bool
}

func (args *enterRequest) IsFirst() bool {
	return args.Data != nil
}

// 登陆队列：1、防止同时出现多个玩家异步耗时操作；2、控制同时登陆的用户数
type enterQueue struct {
	m      map[int]*enterRequest
	waitq  list.List // 正在处理的登陆队列
	isQuit bool      // 正在退出游戏
}

var gEnterQueue = newEnterQueue()

func newEnterQueue() *enterQueue {
	eq := &enterQueue{
		m: make(map[int]*enterRequest),
	}
	return eq
}

func (eq *enterQueue) Check(uid int) {
	if args, ok := eq.m[uid]; ok {
		eq.LoadAndEnter(args.UId)
	}
}

// 清理登陆队列中超时的请求
func (eq *enterQueue) clean() {
	now := time.Now()
	for i := 0; i < 8; i++ {
		e := eq.waitq.Front()
		if e == nil {
			break
		}

		enterReq := e.Value.(*enterRequest)
		if now.Before(enterReq.expireTime) {
			break
		}
		eq.Remove(enterReq.UId)
	}
}

func (eq *enterQueue) Remove(uid int) {
	if data, ok := eq.m[uid]; ok {
		delete(eq.m, uid)
		if e := data.e; e != nil {
			eq.waitq.Remove(e)
			data.e = nil
		}
	}
}

func (eq *enterQueue) PushBack(args *enterRequest) errcode.Error {
	uid, subId, ss := args.UId, args.SubId, args.session
	if eq.isQuit {
		return errcode.New("service_maintain", "服务器正在维护中...")
	}

	// 玩家已经在游戏中
	if old, ok := gAllPlayers[uid]; ok && args.LeaveServer != old.enterReq.ServerName {
		args.isOnline = true
		if old.IsBusy() {
			return errNoResponse
		}
		if oldSubId := old.enterReq.SubId; subId != oldSubId {
			args.oldSubId = oldSubId
			return errEnterOtherGame
		}
		// CLIENT 屏蔽相同链接的多个登陆请求
		if oldss := old.enterReq.session; oldss != nil {
			if oldss.Id == ss.Id {
				return errNoResponse
			}
			old.WriteJSON("Enter", errcode.New("enter_already", "账号已登录"))
			delete(gGatewayPlayers, oldss.Id)
		}
		if old.enterReq.Token != args.Token {
			return errcode.New("invalid_token", "会话无效")
		}
	}
	// 限制同时登陆的请求数
	if _, ok := eq.m[uid]; !ok && eq.waitq.Len() >= maxLoginQueue {
		return errcode.New("too_much_login", "登录需要排队")
	}

	// 清理登陆队列中过期的请求
	eq.clean()
	if last, ok := eq.m[uid]; ok && last.Token == args.Token {
		eq.Remove(uid)
	}
	if _, ok := eq.m[uid]; ok {
		return errcode.Retry
	}

	// 保存或替换登陆请求
	eq.m[uid] = args
	args.e = eq.waitq.PushBack(args)

	// 处理登陆队列
	eq.LoadAndEnter(uid)
	return errcode.Ok
}

func (eq *enterQueue) LoadAndEnter(uid int) {
	args := eq.m[uid]
	ip, subId, token := args.clientIP, args.SubId, args.Token

	go func() {
		var enterGameResp *pb.EnterGameResp
		var loginParamsResp *pb.QueryLoginParamsResp
		auth, err := rpc.CacheClient().Auth(context.Background(), &pb.AuthReq{Uid: int32(uid), Ip: ip})
		if err != nil {
			return
		}

		if !args.isOnline {
			// 不请求离开，玩家直接离开进入游戏
			if args.LeaveServer != "" && auth.ServerName != "" {
				cmd.Request(auth.ServerName, "FUNC_Leave", cmd.M{"UId": uid})
			}

			rpc.CacheClient().Visit(context.Background(), &pb.VisitReq{Uid: int32(uid), SubId: int32(subId), ServerName: GetName()})
			// token相同，并且玩家不在游戏中
			if token == auth.Token {
				enterGameResp, _ = rpc.CacheClient().EnterGame(context.Background(), &pb.EnterGameReq{Uid: int32(uid)})
				loginParamsResp, _ = rpc.CacheClient().QueryLoginParams(context.TODO(), &pb.QueryLoginParamsReq{Uid: int32(uid)})
			}
		}
		rpc.OnResponse(func() {
			if args, ok := eq.m[uid]; ok {
				args.Data = enterGameResp
				args.LoginParams = loginParamsResp.Params
				args.Auth = auth
				eq.Pop(args)
			}
		})
	}()
}

func (eq *enterQueue) Pop(req *enterRequest) {
	uid := req.UId
	ss := req.session
	e := eq.TryEnter(req)

	eq.Remove(uid)
	// 更新网关地址
	var matchServer string
	if errcode.IsOk(e) {
		matchServer = GetName()
	} else if e == errEnterOtherGame {
		matchServer = req.Data.UserInfo.ServerName
	}
	if matchServer != "" {
		ss.WriteJSON("FUNC_SwitchServer", cmd.M{"MatchServer": matchServer, "ServerName": req.ServerName, "UId": uid})
	}

	// 首次成功进入或者重连
	if errcode.IsOk(e) {
		p := gAllPlayers[uid]
		p.IP = req.clientIP
		p.enterReq.session = ss
		gGatewayPlayers[ss.Id] = p

		p.OnEnter()
		log.Debugf("player %d enter all cost %v", p.Id, time.Since(req.startTime))

		p.enterReq.Data = nil
	} else if !req.IsFirst() {
		// 重连进入失败
		WriteMessage(ss, req.ServerName, "Enter", e)
	} else {
		// 首次进入失败
		mySubId := req.Data.UserInfo.SubId
		delete(gAllPlayers, uid)
		delete(gGatewayPlayers, ss.Id)
		go func() {
			if e != errEnterOtherGame {
				rpc.CacheClient().Visit(context.Background(), &pb.VisitReq{Uid: int32(uid)})
			}
			WriteMessage(ss, req.ServerName, "Enter", cmd.M{"Key": e, "Msg": e.Message(), "SubId": mySubId})
		}()
	}
}

func (eq *enterQueue) TryEnter(args *enterRequest) errcode.Error {
	uid, subId, token := args.UId, args.SubId, args.Token
	// 游戏正在关闭存档
	if eq.isQuit {
		return errcode.Retry
	}
	// 玩家已在游戏中
	if player, ok := gAllPlayers[uid]; ok {
		if player.IsBusy() {
			return errcode.Retry
		}
		return errcode.Ok
	}
	// 验证失败
	if token != args.Auth.Token {
		return errcode.Retry
	}
	// 无效的玩家
	if args.Data == nil || args.Data.UserInfo == nil {
		return errcode.Retry
	}

	mySubId := int(args.Data.UserInfo.SubId)
	myServer := args.Data.UserInfo.ServerName
	if myServer != "" && !(mySubId == subId && GetName() == myServer) {
		args.oldSubId = mySubId
		return errEnterOtherGame
	}
	comer := createPlayer()
	comer.Id = uid
	comer.enterReq = args

	gAllPlayers[uid] = comer
	// gGatewayPlayers[ss.Id] = comer
	return comer.Enter() // enter
}

func funcEnter(ctx *cmd.Context, data any) {
	args := data.(*enterArgs)
	clientIP, _, _ := net.SplitHostPort(ctx.ClientAddr)

	enterReq := &enterRequest{
		UId:         args.Uid,
		SubId:       args.SubId,
		Token:       args.Token,
		LeaveServer: args.LeaveServer,
		ServerName:  ctx.ServerName,

		clientIP:   clientIP,
		session:    &cmd.Session{Id: ctx.Ssid, Out: ctx.Out},
		expireTime: time.Now().Add(30 * time.Second),
		startTime:  time.Now(),
	}

	if args.Token == "" {
		return
	}
	// 忽略同一个链接登陆不同的账号
	ply := GetGatewayPlayer(ctx.Ssid)
	if ply != nil && ply.Id != enterReq.UId {
		return
	}

	e := gEnterQueue.PushBack(enterReq)
	if e != errcode.Ok && e != errNoResponse {
		WriteMessage(enterReq.session, enterReq.ServerName, "Enter", cmd.M{"Err": e, "SubId": enterReq.oldSubId})
	}
	log.Infof("player %d enter %s:%d user+robot num %d cost(except rpc) %v", enterReq.UId, enterReq.ServerName, enterReq.SubId, len(gAllPlayers), time.Since(enterReq.startTime))
}

func funcAutoLeave(ctx *cmd.Context, data any) {
	args := data.(*enterArgs)
	uid := args.Uid
	ply := GetPlayer(uid)
	// log.Debugf("player %d auto leave", uid)

	if ply == nil {
		ctx.Out.WriteJSON("FUNC_Leave", cmd.M{"UId": uid})
	} else {
		ply.Leave2(ctx, errcode.Ok)
	}
}
