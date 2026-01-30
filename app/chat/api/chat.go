// ============================================================================
// 聊天服务 API 入口
// ============================================================================
//
// 负责人：马华恩（E同学）
//
// 说明：
//   chat-api 是聊天服务的 HTTP 接口层，负责：
//   - 聊天室列表
//   - 消息历史查询
//   - 消息发送（HTTP 方式）
//
//   注意：WebSocket 连接在 chat/ws 目录单独实现
//
// 启动命令：
//   go run chat.go -f etc/chat-api.yaml
//
// 代码生成：
//   cd app/chat/api
//   goctl api go -api desc/chat.api -dir . -style go_zero
//
// ============================================================================

package main

import (
	"flag"
	"fmt"

	"activity-platform/common/response"

	// TODO(马华恩): goctl 生成代码后取消注释
	// "activity-platform/app/chat/api/internal/config"
	// "activity-platform/app/chat/api/internal/handler"
	// "activity-platform/app/chat/api/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/chat-api.yaml", "配置文件路径")

func main() {
	flag.Parse()

	// ============================================================================
	// 重要：设置全局错误处理器（必须在 server.Start() 之前）
	// ============================================================================
	response.SetupGlobalErrorHandler()
	// response.SetupGlobalOkHandler() // 可选：统一成功响应格式
	// ============================================================================

	// TODO(马华恩): goctl 生成代码后，取消下方注释，删除临时代码
	//
	// var c config.Config
	// conf.MustLoad(*configFile, &c)
	//
	// server := rest.MustNewServer(c.RestConf)
	// defer server.Stop()
	//
	// ctx := svc.NewServiceContext(c)
	// handler.RegisterHandlers(server, ctx)
	//
	// fmt.Printf("Starting chat-api server at %s:%d...\n", c.Host, c.Port)
	// server.Start()

	// ============ 临时代码（goctl 生成后删除）============
	var c struct {
		rest.RestConf
	}
	conf.MustLoad(*configFile, &c)
	fmt.Printf("chat-api 骨架已就绪，等待 goctl 生成代码...\n")
	fmt.Printf("请执行: cd app/chat/api && goctl api go -api desc/chat.api -dir . -style go_zero\n")
	// ============ 临时代码结束 ============
}
