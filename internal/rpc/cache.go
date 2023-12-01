package rpc

import (
	"context"
	"io"
	"time"

	"gofishing-game/internal/env"
	"gofishing-game/internal/pb"

	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
	"google.golang.org/grpc"
)

var defaultCacheClient pb.CacheClient

type authWithPerRPCCredentials map[string]string

func (auth authWithPerRPCCredentials) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return auth, nil
}

func (auth authWithPerRPCCredentials) RequireTransportSecurity() bool {
	return false
}

func init() {
	auth := authWithPerRPCCredentials(map[string]string{
		"Sign": env.Config().Sign,
	})
	opts := []grpc.DialOption{
		grpc.WithPerRPCCredentials(auth),
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
	// 优先加载本地配置
	config.LoadLocalTables("tables")
	// 如果DB存在相同的配置表，将覆盖替换本地的拷贝
	loadRemoteTables()

	// 加载屏蔽词
	words := make([]string, 0, 4096)
	for _, row := range config.Rows("Fuck") {
		if s, ok := config.String("Fuck", row, "Fuck"); ok {
			words = append(words, s)
		}
	}

	cmd.BindWithoutQueue("FUNC_EffectConfigTable", funcEffectConfigTable, (*tableArgs)(nil))
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
	stream, err := CacheClient().LoadAllTable(context.Background(), &pb.Request{})
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
	Name   string
	Tables []string
}

// 直接使用TCP协议发送，会有64K大小限制
func funcEffectConfigTable(ctx *cmd.Context, data any) {
	args := data.(*tableArgs)
	for _, name := range args.Tables {
		table, err := CacheClient().LoadTable(context.Background(), &pb.Request{Name: name})
		if err != nil {
			log.Errorf("load table %s error: %v", name, err)
			return
		}

		log.Info("effect config table", name)
		log.Info(table.Content)
		config.LoadTable(name, []byte(table.Content))
	}
}
