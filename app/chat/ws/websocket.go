// ============================================================================
// WebSocket 独立服务（可选）
// ============================================================================
//
// 说明：
//   这是 WebSocket 服务的独立运行版本，监听独立端口（默认 8889）。
//
//   ⚠️ 注意：WebSocket 功能已集成到 chat-api 服务中（推荐使用）
//   如果使用合并版本（app/chat/api/chat.go），则不需要运行此服务。
//
// 使用场景：
//   1. 需要将 WebSocket 与 API 服务分离部署
//   2. 需要独立扩展 WebSocket 服务
//   3. 调试和测试 WebSocket 功能
//
// 启动命令：
//   go run websocket.go -f etc/websocket.yaml
//
// 对比：
//   • 独立版本（本文件）：WebSocket 单独运行在 8889 端口
//   • 合并版本（推荐）：WebSocket 集成在 chat-api 的 8003 端口
//
// ============================================================================

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"

	"activity-platform/app/chat/ws/hub"
	"activity-platform/app/chat/ws/internal/config"
	"activity-platform/app/chat/ws/internal/handler"
	"activity-platform/app/chat/ws/internal/logic"
	"activity-platform/app/chat/ws/internal/svc"
)

var configFile = flag.String("f", "etc/websocket.yaml", "the config file")

func main() {
	flag.Parse()

	// 加载配置
	var c config.Config
	conf.MustLoad(*configFile, &c)

	// 创建服务上下文
	svcCtx := svc.NewServiceContext(c)

	// 创建消息处理器
	messageHandler := logic.NewMessageLogic(context.Background(), svcCtx)

	// 创建 Hub
	h := hub.NewHub(messageHandler, svcCtx.MessagingClient, svcCtx.RedisClient)

	// 启动 Hub
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go h.Run(ctx)

	// 创建 HTTP 服务器
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", handler.WebSocketHandler(svcCtx, h))

	// 健康检查
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// 在线用户数查询
	mux.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := fmt.Fprintf(w, `{"online_users":%d}`, h.GetOnlineUserCount())
		if err != nil {
			return
		}
	})

	// 获取用户状态
	mux.HandleFunc("/api/users/status", handler.GetUserStatusHandler(h))

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", c.Host, c.Port),
		Handler: mux,
	}

	// 启动服务器
	go func() {
		logx.Infof("WebSocket 服务启动在 %s:%d", c.Host, c.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logx.Errorf("服务器错误: %v", err)
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	logx.Info("正在关闭服务器...")
	cancel()

	// 停止接收新连接
	if err := server.Shutdown(context.Background()); err != nil {
		logx.Errorf("服务器关闭错误: %v", err)
	}

	// 等待消息保存队列处理完成（重要！）
	if svcCtx.SaveQueue != nil {
		logx.Info("等待消息保存队列处理完成...")
		svcCtx.SaveQueue.Stop()
	}

	// 关闭消息中间件客户端
	err := svcCtx.MessagingClient.Close()
	if err != nil {
		return
	}

	logx.Info("服务器已停止")
}
