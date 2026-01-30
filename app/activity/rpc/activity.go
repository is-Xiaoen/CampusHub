package main

import (
	"flag"
)

var configFile = flag.String("f", "etc/activity.yaml", "配置文件路径")

func main() {

}

// 活动服务 RPC 入口
// 说明：
//   activity-rpc 是活动服务的 gRPC 服务层，负责：
//   - 活动 CRUD + 状态机
//   - 活动列表/详情/搜索
//   - 分类标签管理
//   - 跨服务调用接口（供 User/Chat 服务调用）
//
// 启动命令：
//   go run activity.go -f etc/activity.yaml
//
// 代码生成：
//   cd app/activity/rpc
//   goctl rpc protoc activity.proto --go_out=. --go-grpc_out=. --zrpc_out=. --style go_zero
