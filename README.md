### 项目计划

过往棋牌、休闲项目代码总结，按如下计划更新
- 移除了相关的业务（√）
- 项目可以在本地正常运行（√）
- 添加可体验的小游戏demo/fingerGuessing（√）
- 开放管理后台。增加付费到账demo（×）
- 开放斗地主demo/doudizhu。旧线上生产项目代码，合并到新项目中。因缺少前端配合测试，完成度95%（√）
- 开放各种地方麻将demo/mahjong。旧线上生产项目代码，合并到新项目中。因缺少前端配合测试，完成度90%（√）
- 开放牛牛（×）
- 开放跑得快（×）
- 开放十三水demo（×）
- 开放压大小（×）
- 优化demo。支持九宫格方式展示


### 命名风格

- 驼峰。首字母小写
	- 网络协议ID、协议字段
	- protobuf字段
	- lua脚本
	- 配置表表名、列名
- 驼峰。首字母大写
	- protobuf消息名
- 下划线
	- SQL表名、字段、索引等

### 部署

1、初始化grpc脚本
```sh
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2
export PATH="$PATH:$(go env GOPATH)/bin"
sudo apt install protobuf-compiler
protoc --proto_path=./ --go-grpc_out=./ --go-grpc_opt=paths=source_relative --go_out=./ --go_opt=paths=source_relative internal/pb/*.proto
```
2、安装依赖的服务
```sh
go install github.com/guogeer/quasar/gateway@latest
go install github.com/guogeer/quasar/router@latest
# 若未设置$GOPATH
cp ~/go/bin/gateway gateway_server
cp ~/go/bin/router router_server
# 若设置了$GOPATH
# cp $GOPATH/bin/gateway gateway_server
# cp $GOPATH/bin/router router_server
# 初始化配置
cp config_bak.yaml config.yaml #根据实际部署修改配置
nohup ./router_server 1>/dev/null 2>>error.log &
nohup ./gateway_server 1>/dev/null 2>>error.log &
```
3、启动业务（调试模式）

3.1 创建go.work
```go
go 1.21.1

use (
	./gofishing-game
	./gofishing-game/cache
	./gofishing-game/hall
	./gofishing-game/login
	./gofishing-game/migrate/fingerguessing
	./gofishing-game/migrate/doudizhu
	./gofishing-game/migrate/mahjong
	./gofishing-plate
	./quasar
	./quasar/gateway
	./quasar/router
)

```
3.2 启动服务
```sh
# go run ./quasar/gateway
# go run ./quasar/router
go run ./gofishing-game/cache
go run ./gofishing-game/hall
go run ./gofishing-game/login
go run ./gofishing-game/migrate/fingerGuessing
go run ./gofishing-game/migrate/mahjong
go run ./gofishing-game/migrate/doudizhu
```
4、调试工具
新增了client.html调试工具
- 自动登录并连接网关
- 打开控制台可以看到消息历史
- 可以模拟消息请求
- 支持url自定义参数open_id（默认test001）、addr（默认localhost:9501）
新增了fingerguessing/demo/fingerguessing.html体验页面，可体验石头剪刀布
