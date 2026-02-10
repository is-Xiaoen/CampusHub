// ============================================================================
// 聊天服务 API + WebSocket 合并版本
// ============================================================================
//
// 负责人：马华恩（E同学）
//
// 说明：
//   chat-api 是聊天服务的 HTTP 接口层，负责：
//   - 聊天室列表
//   - 消息历史查询
//   - 消息发送（HTTP 方式）
//   - WebSocket 实时连接（集成在同一端口）
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
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	// API 相关
	"activity-platform/app/chat/api/internal/config"
	"activity-platform/app/chat/api/internal/handler"
	"activity-platform/app/chat/api/internal/svc"
	"activity-platform/common/response"

	// WebSocket 相关
	wsIntegration "activity-platform/app/chat/ws/integration"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
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
	response.SetupGlobalOkHandler()
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

	// ============================================================================
	// WebSocket 功能集成
	// ============================================================================
	var wsService *wsIntegration.WebSocketService

	if c.WebSocket.Enabled {
		// 创建 WebSocket 配置
		wsConfig := wsIntegration.WebSocketConfig{
			RedisHost:         c.Redis.Host,
			RedisPass:         c.Redis.Pass,
			RedisDB:           0,
			ChatRpcConfig:     c.ChatRpc,
			UserRpcConfig:     c.UserRpc,
			JwtSecret:         c.Auth.AccessSecret,
			MaxConnections:    c.WebSocket.MaxConnections,
			ReadTimeout:       c.WebSocket.ReadTimeout,
			WriteTimeout:      c.WebSocket.WriteTimeout,
			HeartbeatInterval: c.WebSocket.HeartbeatInterval,
		}

		// 创建 WebSocket 服务
		var err error
		wsService, err = wsIntegration.NewWebSocketService(wsConfig)
		if err != nil {
			logx.Errorf("创建 WebSocket 服务失败: %v", err)
			panic(err)
		}

		// 启动 WebSocket Hub
		hubCtx, hubCancel := context.WithCancel(context.Background())
		defer hubCancel()
		wsService.Start(hubCtx)

		// 注册 WebSocket 路由
		server.AddRoute(rest.Route{
			Method:  http.MethodGet,
			Path:    "/ws",
			Handler: wsService.GetWebSocketHandler(),
		})

		// WebSocket 健康检查
		server.AddRoute(rest.Route{
			Method:  http.MethodGet,
			Path:    "/ws/health",
			Handler: wsService.GetHealthHandler(),
		})

		// 在线用户统计
		server.AddRoute(rest.Route{
			Method:  http.MethodGet,
			Path:    "/ws/stats",
			Handler: wsService.GetStatsHandler(),
		})

		logx.Infof("WebSocket 路由: ws://%s:%d/ws", c.Host, c.Port)
	} else {
		logx.Info("WebSocket 功能已禁用")
	}

	// ============================================================================
	// 启动服务器
	// ============================================================================
	fmt.Printf("正在启动 chat-api 服务，监听地址：%s:%d...\n", c.Host, c.Port)
	fmt.Printf("API 路由: http://%s:%d/api/chat/*\n", c.Host, c.Port)
	if c.WebSocket.Enabled {
		fmt.Printf("WebSocket: ws://%s:%d/ws\n", c.Host, c.Port)
	}

	// 优雅关闭
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
		<-quit

		logx.Info("正在关闭服务器...")

		// 关闭 WebSocket 服务
		if wsService != nil {
			if err := wsService.Close(); err != nil {
				logx.Errorf("关闭 WebSocket 服务失败: %v", err)
			}
		}

		logx.Info("服务器已停止")
		os.Exit(0)
	}()

	server.Start()
}
