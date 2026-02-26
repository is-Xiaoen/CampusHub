package verifyservicelogic

import (
	"context"
	"encoding/json"
	"time"

	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/common/messaging"

	"github.com/zeromicro/go-zero/core/logx"
)

// PublishVerifyProgress 对外暴露的发布函数：给非 verifyservice 包（例如 cron/定时任务）调用。
// 说明：这里不返回 error，走 best-effort（尽力而为）模式，失败只记录日志。
func PublishVerifyProgress(
	ctx context.Context,              // 调用链上下文（用于日志/链路；允许传 nil）
	svcCtx *svc.ServiceContext,       // 服务上下文（包含 MsgClient 等依赖）
	userID, verifyID int64,           // 事件路由关键信息：发给哪个用户、哪个认证单
	status int8,                      // 状态码（内部用 int8，后续会转为消息结构里的 int32）
	operator string,                  // 操作者标识（人工/系统任务等）
) {
	publishVerifyProgress(ctx, svcCtx, userID, verifyID, status, operator)
	// 仅做转发：将对外 API 与内部实现隔离，便于以后调整实现而不影响外部调用方。
}

// publishVerifyProgress 实际发布认证进度事件（best-effort）：任何失败都不向上抛，只打日志。
func publishVerifyProgress(
	ctx context.Context,
	svcCtx *svc.ServiceContext,
	userID, verifyID int64,
	status int8,
	operator string,
) {
	if ctx == nil {
		ctx = context.Background()
		// 兜底：允许调用方传 nil，保证后续日志与 Publish 不会因为 ctx 为空而 panic。
	}
	logger := logx.WithContext(ctx)
	// 从 ctx 构造 logger：便于携带 trace 信息（如果 ctx 中有）。

	if svcCtx == nil || svcCtx.MsgClient == nil {
		// 依赖未初始化：无法发布消息，按 best-effort 直接跳过，避免影响主流程。
		logger.Infof("[VerifyProgress] 消息客户端未初始化，跳过发布: userId=%d, verifyId=%d, status=%d",
			userID, verifyID, status)
		return
	}

	event := messaging.VerifyProgressEventData{
		UserID:    userID,               // 目标用户：下游（如 WS 推送服务）用它做路由
		VerifyID:  verifyID,             // 认证单 ID：前端/下游用它定位是哪一次认证
		Status:    int32(status),        // 状态：转为消息结构期望的类型（int32）
		Operator:  operator,             // 操作来源：用于审计/排障
		Refresh:   true,                 // 提示前端是否需要刷新/主动拉取详情（业务开关）
		Timestamp: time.Now().Unix(),    // 事件时间戳：用于排序/去重/展示
	}

	payload, err := json.Marshal(event)
	// 将事件序列化为 JSON bytes，作为消息 payload。
	if err != nil {
		// 序列化失败：通常是结构体字段不可序列化或数据异常；best-effort 仅记录并返回。
		logger.Errorf("[VerifyProgress] 序列化失败: userId=%d, verifyId=%d, status=%d, err=%v",
			userID, verifyID, status, err)
		return
	}

	if err := svcCtx.MsgClient.Publish(ctx, messaging.TopicVerifyProgress, payload); err != nil {
		// 发布失败：可能是 MQ 不可用/网络抖动/权限问题；best-effort 仅记录并返回，不影响主流程。
		logger.Errorf("[VerifyProgress] 发布失败: userId=%d, verifyId=%d, status=%d, err=%v",
			userID, verifyID, status, err)
		return
	}
	// 成功：不额外打日志，避免高频事件刷屏（若需要可加 debug 级别日志）。
}