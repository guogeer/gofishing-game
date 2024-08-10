package main

import (
	"flag"
	"fmt"
	"net"

	"gofishing-game/cache/models"
	"gofishing-game/internal/dbo"
	"gofishing-game/internal/env"
	"gofishing-game/internal/pb"

	"github.com/guogeer/quasar/v2/cmd"
	"github.com/guogeer/quasar/v2/log"
	"google.golang.org/grpc"
)

var port = flag.Int("port", 9000, "cache server port")

func main() {
	flag.Parse()

	// init database
	t := env.Config().DB.Game
	dbo.SetSource(t.User, t.Password, t.Addr, t.Name)
	db, _ := dbo.Get().DB()
	if n := t.MaxIdleConns; n > 0 {
		db.SetMaxIdleConns(n)
	}
	if n := t.MaxOpenConns; n > 0 {
		db.SetMaxOpenConns(n)
	}
	dbo.Get().AutoMigrate(
		models.ClientVersion{},
		models.Dict{},
		models.ItemLog{},
		models.Mail{},
		models.OnlineLog{},
		models.Script{},
		models.Table{},
		models.UserBin{},
		models.UserInfo{},
		models.UserPlate{},
	)

	go func() { Tick() }()

	addr := fmt.Sprintf(":%v", *port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	log.Infof("start cache server and listen %s", addr)
	cmd.RegisterService(&cmd.ServiceConfig{Name: "cache", Addr: addr})
	log.Debug("register service ok")
	//opts := []grpc.ServerOption{
	//	grpc.UnaryInterceptor(ensureValidToken),
	//}
	opts := []grpc.ServerOption{}
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterCacheServer(grpcServer, &Cache{})
	grpcServer.Serve(lis)
}
