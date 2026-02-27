# WebSocket 服务说明

## 📌 两种运行方式

### 方式一：合并模式（推荐）✅

WebSocket 集成在 chat-api 服务中，共享 8003 端口。

**启动：**
```bash
cd app/chat/api
go run chat.go -f etc/chat-api.yaml
```

**端点：**
- HTTP API: `http://localhost:8003/api/chat/*`
- WebSocket: `ws://localhost:8003/ws`

**优点：**
- ✅ 只需一个端口
- ✅ 客户端连接简单
- ✅ 部署简单
- ✅ 资源占用少

---

### 方式二：独立模式（可选）

WebSocket 独立运行在 8889 端口。

**启动：**
```bash
# 1. 启动 API 服务
cd app/chat/api
go run chat.go -f etc/chat-api.yaml

# 2. 启动独立 WebSocket 服务
cd app/chat/ws
go run websocket.go -f etc/websocket.yaml
```

**端点：**
- HTTP API: `http://localhost:8003/api/chat/*`
- WebSocket: `ws://localhost:8889/ws`

**优点：**
- ✅ 服务分离
- ✅ 可独立扩展
- ✅ 便于调试

---

## 🎯 如何选择

| 场景 | 推荐方式 |
|------|---------|
| 开发环境 | 合并模式 |
| 小规模部署（< 1000 用户） | 合并模式 |
| 大规模部署（> 1000 用户） | 独立模式 |
| 需要独立扩展 WebSocket | 独立模式 |
| 简化部署 | 合并模式 |

---

**默认推荐：合并模式** 🎉

---

## OCR 认证进度实时推送

WebSocket 服务会订阅 Redis Stream `verify:progress`，并向对应用户下发 `verify_progress`：

```json
{
  "type": "verify_progress",
  "message_id": "verify_123_1700000000",
  "timestamp": 1700000000,
  "data": {
    "verify_id": 123,
    "status": 2,
    "refresh": true
  }
}
```

前端联调约定：

1. 收到 `type=verify_progress` 且 `data.refresh=true` 后，立即调用 `GET /api/v1/verify/student/current`。
2. 用接口返回的最新 `status / need_action / verify_data` 刷新页面状态。
3. 可选去抖：同一 `verify_id + status` 在 300ms 内仅触发一次刷新。
