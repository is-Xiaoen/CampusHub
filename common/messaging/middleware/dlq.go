package middleware

import (
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	wmMiddleware "github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// dlqMessagesTotal DLQ 消息计数器
	dlqMessagesTotal *prometheus.CounterVec

	// dlqPublishErrorsTotal DLQ 发布失败计数器
	dlqPublishErrorsTotal *prometheus.CounterVec
)

func init() {
	dlqMessagesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "messaging_dlq_messages_total",
			Help: "Total number of messages sent to DLQ",
		},
		[]string{"service", "topic", "reason"},
	)

	dlqPublishErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "messaging_dlq_publish_errors_total",
			Help: "Total number of DLQ publish errors",
		},
		[]string{"service", "topic"},
	)
}

// DLQMiddlewareConfig DLQ 中间件配置
type DLQMiddlewareConfig struct {
	Publisher        message.Publisher
	ServiceName      string
	OriginalTopic    string
	HandlerName      string
	DLQTopic         string
	OnlyNonRetryable bool             // 如果为 true，只有不可重试错误进入 DLQ
	IsRetryableFunc  func(error) bool // 判断错误是否可重试的函数
}

// NewDLQMiddleware 创建 DLQ 中间件
// 包装 Watermill 的 PoisonQueueWithFilter，添加元数据增强和监控
func NewDLQMiddleware(config DLQMiddlewareConfig) (message.HandlerMiddleware, error) {
	if config.Publisher == nil {
		return nil, fmt.Errorf("DLQ publisher cannot be nil")
	}
	if config.DLQTopic == "" {
		return nil, fmt.Errorf("DLQ topic cannot be empty")
	}

	// 创建基础 PoisonQueue 中间件
	var basePoisonQueue message.HandlerMiddleware
	var err error

	// 如果启用 OnlyNonRetryable，使用 PoisonQueueWithFilter
	if config.OnlyNonRetryable && config.IsRetryableFunc != nil {
		basePoisonQueue, err = wmMiddleware.PoisonQueueWithFilter(
			config.Publisher,
			config.DLQTopic,
			func(err error) bool {
				// 只有不可重试错误才进入 DLQ
				return !config.IsRetryableFunc(err)
			},
		)
	} else {
		// 否则使用标准 PoisonQueue（所有错误都进入 DLQ）
		basePoisonQueue, err = wmMiddleware.PoisonQueue(config.Publisher, config.DLQTopic)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create poison queue: %w", err)
	}

	// 包装中间件，添加元数据增强和监控
	return func(h message.HandlerFunc) message.HandlerFunc {
		// 首先创建一个包装器，在错误时添加元数据
		metadataEnhancer := func(msg *message.Message) ([]*message.Message, error) {
			msgs, err := h(msg)

			// 如果有错误，添加 DLQ 元数据（在 PoisonQueue 发布之前）
			if err != nil {
				shouldGoToDLQ := true
				if config.OnlyNonRetryable && config.IsRetryableFunc != nil {
					shouldGoToDLQ = !config.IsRetryableFunc(err)
				}

				if shouldGoToDLQ {
					// 添加 DLQ 元数据
					msg.Metadata.Set("dlq_reason", err.Error())
					msg.Metadata.Set("dlq_timestamp", time.Now().Format(time.RFC3339))
					msg.Metadata.Set("dlq_original_topic", config.OriginalTopic)
					msg.Metadata.Set("dlq_handler_name", config.HandlerName)

					// 记录 Prometheus 指标
					reason := "error"
					if config.IsRetryableFunc != nil && !config.IsRetryableFunc(err) {
						reason = "non_retryable"
					} else {
						reason = "max_retries_exceeded"
					}
					dlqMessagesTotal.WithLabelValues(config.ServiceName, config.OriginalTopic, reason).Inc()
				}
			}

			return msgs, err
		}

		// 然后应用 PoisonQueue 中间件（它会在错误时发布消息到 DLQ）
		return basePoisonQueue(metadataEnhancer)
	}, nil
}
