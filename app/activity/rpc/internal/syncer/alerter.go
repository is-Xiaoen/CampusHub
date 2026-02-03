package syncer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// ==================== 告警实现 ====================
//
// 提供多种告警通道：
//   1. LogAlerter - 日志告警（开发/测试环境）
//   2. WebhookAlerter - Webhook 告警（生产环境，对接钉钉/飞书/企业微信）
//   3. CompositeAlerter - 组合告警（同时发送到多个通道）
//
// 企业级设计：
//   - 接口抽象，便于扩展
//   - 限流防刷，避免告警风暴
//   - 告警聚合，相同告警合并

// ==================== 日志告警器（开发环境） ====================

// LogAlerter 日志告警器
//
// 将告警信息输出到日志，适用于开发和测试环境
type LogAlerter struct{}

// NewLogAlerter 创建日志告警器
func NewLogAlerter() *LogAlerter {
	return &LogAlerter{}
}

// SendAlert 发送告警到日志
func (a *LogAlerter) SendAlert(ctx context.Context, level AlertLevel, title, content string) error {
	levelStr := "INFO"
	switch level {
	case AlertLevelWarning:
		levelStr = "WARNING"
	case AlertLevelError:
		levelStr = "ERROR"
	}

	logx.Infof("[ALERT][%s] %s | %s", levelStr, title, content)
	return nil
}

// ==================== Webhook 告警器（生产环境） ====================

// WebhookAlerter Webhook 告警器
//
// 通过 HTTP POST 发送告警到外部服务（钉钉、飞书、企业微信等）
type WebhookAlerter struct {
	webhookURL string        // Webhook URL
	httpClient *http.Client  // HTTP 客户端
	rateLimit  *rateLimiter  // 限流器
}

// WebhookMessage Webhook 消息格式
type WebhookMessage struct {
	MsgType string                 `json:"msgtype"`
	Text    WebhookTextContent     `json:"text,omitempty"`
	Extra   map[string]interface{} `json:"extra,omitempty"`
}

// WebhookTextContent 文本内容
type WebhookTextContent struct {
	Content string `json:"content"`
}

// NewWebhookAlerter 创建 Webhook 告警器
//
// 参数：
//   - webhookURL: Webhook 地址
//   - ratePerMinute: 每分钟最大告警数（防止告警风暴）
func NewWebhookAlerter(webhookURL string, ratePerMinute int) *WebhookAlerter {
	return &WebhookAlerter{
		webhookURL: webhookURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		rateLimit: newRateLimiter(ratePerMinute),
	}
}

// SendAlert 发送告警到 Webhook
func (a *WebhookAlerter) SendAlert(ctx context.Context, level AlertLevel, title, content string) error {
	// 1. 限流检查
	if !a.rateLimit.allow() {
		logx.Infof("[WebhookAlerter] 告警被限流: %s", title)
		return nil // 限流不报错，只记录日志
	}

	// 2. 构建消息
	levelEmoji := "ℹ️"
	switch level {
	case AlertLevelWarning:
		levelEmoji = "⚠️"
	case AlertLevelError:
		levelEmoji = "🚨"
	}

	message := WebhookMessage{
		MsgType: "text",
		Text: WebhookTextContent{
			Content: fmt.Sprintf("%s %s\n\n%s\n\n时间: %s",
				levelEmoji, title, content, time.Now().Format("2006-01-02 15:04:05")),
		},
	}

	// 3. 发送请求
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("序列化告警消息失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("发送告警失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("告警响应异常: status=%d", resp.StatusCode)
	}

	logx.Infof("[WebhookAlerter] 告警发送成功: %s", title)
	return nil
}

// ==================== 组合告警器 ====================

// CompositeAlerter 组合告警器
//
// 同时发送到多个告警通道
type CompositeAlerter struct {
	alerters []Alerter
}

// NewCompositeAlerter 创建组合告警器
func NewCompositeAlerter(alerters ...Alerter) *CompositeAlerter {
	return &CompositeAlerter{
		alerters: alerters,
	}
}

// SendAlert 发送告警到所有通道
func (a *CompositeAlerter) SendAlert(ctx context.Context, level AlertLevel, title, content string) error {
	var lastErr error
	for _, alerter := range a.alerters {
		if err := alerter.SendAlert(ctx, level, title, content); err != nil {
			logx.Errorf("[CompositeAlerter] 发送告警失败: %v", err)
			lastErr = err
		}
	}
	return lastErr
}

// ==================== 限流器 ====================

// rateLimiter 简单的滑动窗口限流器
type rateLimiter struct {
	maxPerMinute int
	timestamps   []int64
}

func newRateLimiter(maxPerMinute int) *rateLimiter {
	if maxPerMinute <= 0 {
		maxPerMinute = 10 // 默认每分钟 10 条
	}
	return &rateLimiter{
		maxPerMinute: maxPerMinute,
		timestamps:   make([]int64, 0, maxPerMinute),
	}
}

func (r *rateLimiter) allow() bool {
	now := time.Now().Unix()
	windowStart := now - 60 // 1 分钟窗口

	// 清理过期记录
	validTimestamps := make([]int64, 0, len(r.timestamps))
	for _, ts := range r.timestamps {
		if ts > windowStart {
			validTimestamps = append(validTimestamps, ts)
		}
	}
	r.timestamps = validTimestamps

	// 检查是否超限
	if len(r.timestamps) >= r.maxPerMinute {
		return false
	}

	// 记录本次请求
	r.timestamps = append(r.timestamps, now)
	return true
}

// ==================== 空告警器（禁用告警） ====================

// NoopAlerter 空告警器
//
// 不发送任何告警，用于禁用告警功能
type NoopAlerter struct{}

// NewNoopAlerter 创建空告警器
func NewNoopAlerter() *NoopAlerter {
	return &NoopAlerter{}
}

// SendAlert 不做任何操作
func (a *NoopAlerter) SendAlert(ctx context.Context, level AlertLevel, title, content string) error {
	return nil
}
