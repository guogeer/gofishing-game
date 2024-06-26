package rpc

import (
	"context"
	"io"
	"time"

	"gofishing-game/internal/env"
	"gofishing-game/internal/pb"

	"github.com/guogeer/quasar/v2/cmd"
	"github.com/guogeer/quasar/v2/config"
	"github.com/guogeer/quasar/v2/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var defaultCacheClient pb.CacheClient

func init() {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	addr, err := cmd.RequestServerAddr("cache")
	if err != nil {
		log.Fatalf("cache or router service is unavailable %v", err)
	}
	log.Infof("request server cache addr: %s", addr)

	for {
		conn, err := grpc.Dial(addr, opts...)
		if err == nil {
			defaultCacheClient = pb.NewCacheClient(conn)
			break
		}
		time.Sleep(5 * time.Second)
	}
	log.Debugf("connect rpc server successfully.")
	// 优先加载本地配置
	config.LoadLocalTables(env.Config().TablePath)
	// 如果DB存在相同的配置表，将覆盖替换本地的拷贝
	loadRemoteTables()

	cmd.Bind("func_effectConfigTable", funcEffectConfigTable, (*tableArgs)(nil), cmd.WithoutQueue())
}

func CacheClient() pb.CacheClient {
	return defaultCacheClient
}

/*func LoadRemoteScripts() error {
	stream, err := CacheClient().LoadAllScript(context.Background(), &pb.Request{})
	if err != nil {
		return err
	}
	for {
		f, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		// 移除脚本远程加载
		if err = script.LoadString(f.Name, f.Body); err != nil {
			return err
		}
	}
	return nil
}
*/

func loadRemoteTables() {
	stream, err := CacheClient().LoadAllTable(context.Background(), &pb.EmptyReq{})
	if err != nil {
		log.Fatalf("load all table config %v", err)
	}
	for {
		table, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("pull table config %v", err)
		}
		err = config.LoadTable(table.Name, []byte(table.Content))
		if err != nil {
			log.Fatalf("load table config %v", err)
		}
	}
}

type tableArgs struct {
	Name   string   `json:"name,omitempty"`
	Tables []string `json:"tables,omitempty"`
}

// 直接使用TCP协议发送，会有64K大小限制
func funcEffectConfigTable(ctx *cmd.Context, data any) {
	args := data.(*tableArgs)
	for _, name := range args.Tables {
		resp, err := CacheClient().LoadTable(context.Background(), &pb.LoadTableReq{Name: name})
		if err != nil {
			log.Errorf("load table %s error: %v", name, err)
			return
		}

		log.Info("effect config table", name)
		log.Info(resp.File.Content)
		config.LoadTable(name, []byte(resp.File.Content))
	}
}
