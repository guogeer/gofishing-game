### 项目计划

过往棋牌、休闲项目代码总结，按如下计划更新
- 移除了相关的业务（√）
- 项目可以在本地正常运行（×）
- 添加可体验的小游戏demo（×）
- 开放管理后台等仓库（×）

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