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

	"activity-platform/app/chat/api/internal/config"
	"activity-platform/app/chat/api/internal/handler"
	"activity-platform/app/chat/api/internal/svc"
	"activity-platform/common/response"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/chat-api.yaml", "配置文件路径")

func main() {
	// 解析命令行参数
	flag.Parse()

	// ============================================================================
	// 重要：设置全局错误处理器（必须在 server.Start() 之前）
	// ============================================================================
	response.SetupGlobalErrorHandler()
	// response.SetupGlobalOkHandler() // 可选：统一成功响应格式
	// ============================================================================

	// 加载配置文件
	var c config.Config
	conf.MustLoad(*configFile, &c)

	// 创建 REST 服务器
	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	// 初始化服务上下文（包含 RPC 客户端、数据库连接等依赖）
	ctx := svc.NewServiceContext(c)

	// 注册所有 HTTP 路由处理器
	handler.RegisterHandlers(server, ctx)

	// 启动服务器
	fmt.Printf("正在启动 chat-api 服务，监听地址：%s:%d...\n", c.Host, c.Port)
	server.Start()
}
