package handler

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"

	"activity-platform/app/chat/ws/hub"
	"activity-platform/app/chat/ws/internal/svc"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 允许跨域
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// WebSocketHandler WebSocket 连接处理器
func WebSocketHandler(svcCtx *svc.ServiceContext, h *hub.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 升级 HTTP 连接为 WebSocket
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logx.Errorf("升级连接失败: %v", err)
			return
		}

		// 创建客户端
		client := hub.NewClient(h, conn)

		// 注册客户端
		h.Register() <- client

		// 启动读写协程
		go client.WritePump()
		go client.ReadPump()

		logx.Info("新的 WebSocket 连接已建立")
	}
}
