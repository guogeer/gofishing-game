package service

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"runtime"
	"time"

	"github.com/guogeer/quasar/utils"

	// "gofishing-game/internal/errcode"
	"gofishing-game/internal/pb"
	"gofishing-game/internal/rpc"

	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/log"
)

var port = flag.Int("port", 9510, "server port")
var serverId = flag.String("server_id", "", "server id")

// 异常退出时保存玩家的数据
func saveAllPlayers() {
	startTime := time.Now()
	for _, player := range gAllPlayers {
		bin := &pb.UserBin{}
		player.dataObj.saveAll(bin)
		rpc.CacheClient().SaveBin(context.Background(), &pb.SaveBinReq{Uid: int32(player.Id), Bin: bin})
		log.Infof("player %d force save data", player.Id)
	}
	log.Infof("server %s quit and save data cost %v", GetServerId(), time.Since(startTime))
}

func Start() {
	// 正常关闭进程
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer func() {
		stop()
		if err := recover(); err != nil {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			log.Error(err)
			log.Errorf("%s", buf)
		}
		saveAllPlayers()
	}()

	flag.Parse()
	loadAllScripts() // 加载脚本

	if *port == 0 {
		panic("server port is zero")
	}
	// 向路由注册服务
	addr := fmt.Sprintf(":%d", *port)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen %v", err)
	}

	srv := &cmd.Server{Addr: addr}
	go func() { srv.Serve(l) }()

	cmd.RegisterService(&cmd.ServiceConfig{
		Id:   GetServerId(),
		Name: GetServerName(),
		Addr: addr,
	})
	globalData.load()
	log.Infof("server id %s name %s start ok.", GetServerId(), GetServerName())

	for {
		select {
		default:
		case <-ctx.Done():
			log.Infof("server %s recv signal SIGINT and quit", GetServerId())
			return
		}
		utils.GetTimerSet().RunOnce()
		rpc.RunOnce() // 无等待
		// handle network message
		cmd.RunOnce() // 无消息时会等待
	}
}
