package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"activity-platform/common/messaging"

	"github.com/zeromicro/go-zero/core/logx"
)

// Producer 活动服务消息发布器
// nil 安全：Producer 或 Client 为 nil 时所有方法静默返回
type Producer struct {
	client *messaging.Client
}

// NewProducer 创建消息发布器
func NewProducer(client *messaging.Client) *Producer {
	if client == nil {
		return nil
	}
	return &Producer{client: client}
}

// publishAsync 异步发布事件（核心方法）
// - 开新 goroutine，不阻塞调用方
// - defer recover 防 panic 传播
// - 3 秒超时防 goroutine 泄漏
// - 发布失败只记日志，不影响主业务
func (p *Producer) publishAsync(topic string, payload interface{}) {
	if p == nil || p.client == nil {
		return
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logx.Errorf("[MQ-Producer] panic recovered: topic=%s, err=%v", topic, r)
			}
		}()

		data, err := json.Marshal(payload)
		if err != nil {
			logx.Errorf("[MQ-Producer] 序列化失败: topic=%s, err=%v", topic, err)
			return
		}

		pubCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		if err := p.client.Publish(pubCtx, topic, data); err != nil {
			logx.Errorf("[MQ-Producer] 发布失败: topic=%s, err=%v", topic, err)
			return
		}

		logx.Infof("[MQ-Producer] 发布成功: topic=%s, size=%d", topic, len(data))
	}()
}

// ==================== 活动事件（Chat MQ 消费）====================

// PublishActivityCreated 发布活动创建事件
func (p *Producer) PublishActivityCreated(ctx context.Context, activityID uint64, creatorID uint64, title string) {
	p.publishAsync(messaging.TopicActivityCreated, messaging.ActivityCreatedEvent{
		ActivityID: fmt.Sprintf("%d", activityID),
		CreatorID:  fmt.Sprintf("%d", creatorID),
		Title:      title,
		CreatedAt:  time.Now(),
	})
}

// PublishMemberJoined 发布用户报名事件
func (p *Producer) PublishMemberJoined(ctx context.Context, activityID uint64, userID uint64) {
	p.publishAsync(messaging.TopicActivityMemberJoined, messaging.ActivityMemberJoinedEvent{
		ActivityID: fmt.Sprintf("%d", activityID),
		UserID:     fmt.Sprintf("%d", userID),
		JoinedAt:   time.Now(),
	})
}

// PublishMemberLeft 发布用户取消报名事件
func (p *Producer) PublishMemberLeft(ctx context.Context, activityID uint64, userID uint64) {
	p.publishAsync(messaging.TopicActivityMemberLeft, messaging.ActivityMemberLeftEvent{
		ActivityID: fmt.Sprintf("%d", activityID),
		UserID:     fmt.Sprintf("%d", userID),
		LeftAt:     time.Now(),
	})
}

// PublishActivityCancelled 发布活动取消事件
func (p *Producer) PublishActivityCancelled(ctx context.Context, activityID uint64, cancelledBy uint64, reason string) {
	p.publishAsync(messaging.TopicActivityCancelled, messaging.ActivityCancelledEvent{
		ActivityID:  fmt.Sprintf("%d", activityID),
		CancelledBy: fmt.Sprintf("%d", cancelledBy),
		Reason:      reason,
		CancelledAt: time.Now(),
	})
}

// ==================== 信用事件（User MQ 消费）====================
// Credit 事件需要 RawMessage 包装，ID 是 int64

// PublishCreditEvent 发布信用事件
func (p *Producer) PublishCreditEvent(ctx context.Context, eventType string, activityID int64, userID int64) {
	if p == nil || p.client == nil {
		return
	}

	// 1. 构造内层 CreditEventData
	creditData := messaging.CreditEventData{
		Type:       eventType,
		ActivityID: activityID,
		UserID:     userID,
		Timestamp:  time.Now().Unix(),
	}

	// 2. 序列化内层为 JSON 字符串
	innerJSON, err := json.Marshal(creditData)
	if err != nil {
		logx.Errorf("[MQ-Producer] 序列化 CreditEventData 失败: %v", err)
		return
	}

	// 3. 包装为 RawMessage（User MQ handler 期望的格式）
	rawMsg := messaging.RawMessage{
		Type: messaging.MsgTypeCreditChange,
		Data: string(innerJSON),
	}

	p.publishAsync(messaging.TopicCreditEvent, rawMsg)
}

// Close 关闭 Producer 底层客户端
func (p *Producer) Close() error {
	if p == nil || p.client == nil {
		return nil
	}
	return p.client.Close()
}
