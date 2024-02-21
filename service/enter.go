package service

import (
	"container/list"
	"context"
	"net"
	"strings"
	"time"

	"gofishing-game/internal/errcode"
	"gofishing-game/internal/gameutils"
	"gofishing-game/internal/pb"
	"gofishing-game/internal/rpc"

	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/log"
)

const maxLoginQueue = 99 // 同时处理的登陆请求

var errEnterOtherGame = errcode.New("enter_other_game", "enter other game")

type enterRequest struct {
	AuthResp        *pb.AuthResp
	EnterGameResp   *pb.EnterGameResp
	LoginParamsResp *pb.QueryLoginParamsResp

	Uid         int
	Token       string
	LeaveServer string // 离开旧场景进入新的场景
	ServerName  string
	RawData     []byte

	e          *list.Element
	session    *cmd.Session
	expireTime time.Time // 过期时间
	clientIP   string
	startTime  time.Time
	isOnline   bool
}

func (args *enterRequest) IsOnline() bool {
	return args.isOnline
}

// 登陆队列：1、防止同时出现多个玩家异步耗时操作；2、控制同时登陆的用户数
type enterQueue struct {
	m      map[int]*enterRequest
	waitq  list.List // 正在处理的登陆队列
	isQuit bool      // 正在退出游戏

	locationFunc func(int) string
}

var defaultEnterQueue = newEnterQueue()

func newEnterQueue() *enterQueue {
	eq := &enterQueue{
		m: make(map[int]*enterRequest),
		locationFunc: func(uid int) string {
			return GetServerId()
		},
	}

	return eq
}

func GetEnterQueue() *enterQueue {
	return defaultEnterQueue
}

func (eq *enterQueue) SetLocationFunc(f func(int) string) {
	eq.locationFunc = f
}

func (eq *enterQueue) getServerLocation(uid int) string {
	return eq.locationFunc(uid)
}

func (eq *enterQueue) GetRequest(uid int) *enterRequest {
	return eq.m[uid]
}

func (eq *enterQueue) Check(uid int) {
	if args, ok := eq.m[uid]; ok {
		eq.loadAndEnter(args.Uid)
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
		eq.removeUser(enterReq.Uid)
	}
}

func (eq *enterQueue) removeUser(uid int) {
	if data, ok := eq.m[uid]; ok {
		delete(eq.m, uid)
		if e := data.e; e != nil {
			eq.waitq.Remove(e)
			data.e = nil
		}
	}
}

func (eq *enterQueue) PushBack(ctx *cmd.Context, token, leaveServer string, data []byte) (*cmd.Session, errcode.Error) {
	clientIP, _, _ := net.SplitHostPort(ctx.ClientAddr)
	enterReq := &enterRequest{
		Token:       token,
		LeaveServer: leaveServer,
		ServerName:  ctx.ServerName,
		RawData:     data,

		clientIP:   clientIP,
		session:    &cmd.Session{Id: ctx.Ssid, Out: ctx.Out},
		expireTime: time.Now().Add(30 * time.Second),
		startTime:  time.Now(),
	}
	if eq.isQuit {
		return enterReq.session, errcode.New("service_maintain", "服务器正在维护中...")
	}

	uid, err := gameutils.ValidateToken(token)
	if err != nil {
		return enterReq.session, err
	}
	enterReq.Uid = uid

	// 忽略同一个链接登陆不同的账号
	ply := GetGatewayPlayer(ctx.Ssid)
	if ply != nil && ply.Id != uid {
		return nil, nil
	}

	// 玩家已经在游戏中
	if old, ok := gAllPlayers[uid]; ok && enterReq.LeaveServer != GetServerId() {
		enterReq.isOnline = true
		if old.IsBusy() {
			return nil, nil
		}

		// CLIENT 屏蔽相同链接的多个登陆请求
		if oldss := old.session; oldss != nil {
			if oldss.Id == enterReq.session.Id {
				return nil, nil
			}
			old.WriteJSON("enter", errcode.New("enter_already", "账号已登录"))
			delete(gGatewayPlayers, oldss.Id)
		}
	}
	// 限制同时登陆的请求数
	if _, ok := eq.m[uid]; !ok && eq.waitq.Len() >= maxLoginQueue {
		return enterReq.session, errcode.New("too_much_login", "登录需要排队")
	}

	// 清理登陆队列中过期的请求
	eq.clean()
	if last, ok := eq.m[uid]; ok && last.Token == enterReq.Token {
		eq.removeUser(uid)
	}
	if _, ok := eq.m[uid]; ok {
		return enterReq.session, errcode.New("in_login_queue", "user is in login queue")
	}

	// 保存或替换登陆请求
	eq.m[uid] = enterReq
	enterReq.e = eq.waitq.PushBack(enterReq)

	// 处理登陆队列
	eq.loadAndEnter(uid)
	return enterReq.session, nil
}
func (eq *enterQueue) loadAndEnter(uid int) {
	args := eq.m[uid]
	ip := args.Token
	serverLocation := eq.getServerLocation(uid)

	go func() {
		auth, _ := rpc.CacheClient().Auth(context.Background(), &pb.AuthReq{Uid: int32(uid), Ip: ip})
		values := strings.Split(auth.ServerLocation, ":")
		if args.LeaveServer != "" && values[0] != "" {
			cmd.Request(values[0], "FUNC_Leave", cmd.M{"uid": uid})
		}

		rpc.CacheClient().Visit(context.Background(), &pb.VisitReq{Uid: int32(uid), ServerLocation: serverLocation})
		enterGameResp, _ := rpc.CacheClient().EnterGame(context.Background(), &pb.EnterGameReq{Uid: int32(uid)})
		loginParamsResp, _ := rpc.CacheClient().QueryLoginParams(context.TODO(), &pb.QueryLoginParamsReq{Uid: int32(uid)})
		rpc.OnResponse(func() {
			if args, ok := eq.m[uid]; ok {
				args.EnterGameResp = enterGameResp
				args.LoginParamsResp = loginParamsResp
				args.AuthResp = auth
				eq.pop(args)
			}
		})
	}()
}

func (eq *enterQueue) pop(req *enterRequest) {
	uid := req.Uid
	ss := req.session
	e := eq.TryEnter(req)

	defer eq.removeUser(uid)
	// 更新网关地址
	var matchServerId string
	if e == nil {
		matchServerId = GetServerId()
	} else if e == errEnterOtherGame {
		values := strings.Split(req.EnterGameResp.UserInfo.ServerLocation, ":")
		matchServerId = values[0]
	}
	if matchServerId != "" {
		ss.WriteJSON("FUNC_SwitchServer", cmd.M{"matchServerId": matchServerId, "serverName": req.ServerName, "uid": uid})
	}

	// 首次成功进入或者重连
	if e == nil {
		p := gAllPlayers[uid]
		p.IP = req.clientIP
		p.session = ss
		gGatewayPlayers[ss.Id] = p

		p.OnEnter()
		log.Debugf("player %d enter all cost %v", p.Id, time.Since(req.startTime))
	} else if req.isOnline {
		// 重连进入失败
		WriteMessage(ss, "enter", e)
	} else {
		// 首次进入失败
		delete(gAllPlayers, uid)
		delete(gGatewayPlayers, ss.Id)
		go func() {
			if e != errEnterOtherGame {
				rpc.CacheClient().Visit(context.Background(), &pb.VisitReq{Uid: int32(uid)})
			}
			WriteMessage(ss, "enter", e)
		}()
	}
}

func (eq *enterQueue) TryEnter(args *enterRequest) errcode.Error {
	// 游戏正在关闭存档
	if eq.isQuit {
		return errcode.New("sys_quit", "server is maintaining")
	}
	// 玩家已在游戏中
	if player, ok := gAllPlayers[args.Uid]; ok {
		if player.IsBusy() {
			return errcode.New("busy_user", "user is busy")
		}
		return nil
	}
	// 无效的玩家
	if args.EnterGameResp == nil || args.EnterGameResp.UserInfo == nil {
		return errcode.New("invalid_user", "user data is invalid")
	}

	comer := createPlayer(args.Uid)
	gAllPlayers[args.Uid] = comer
	return comer.Enter()
}
