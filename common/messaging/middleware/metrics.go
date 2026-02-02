package middleware

import (
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// messagesProcessed 消息处理计数器
	messagesProcessed *prometheus.CounterVec

	// processingDuration 消息处理耗时直方图
	processingDuration *prometheus.HistogramVec

	// messagesInFlight 正在处理的消息数量
	messagesInFlight *prometheus.GaugeVec
)

func init() {
	messagesProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "messaging_messages_processed_total",
			Help: "Total number of messages processed",
		},
		[]string{"service", "topic", "status"},
	)

	processingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "messaging_processing_duration_seconds",
			Help:    "Message processing duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "topic"},
	)

	messagesInFlight = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "messaging_messages_in_flight",
			Help: "Number of messages currently being processed",
		},
		[]string{"service", "topic"},
	)
}

// NewMetricsMiddleware 创建 Prometheus 指标中间件
func NewMetricsMiddleware(serviceName string) message.HandlerMiddleware {
	return func(h message.HandlerFunc) message.HandlerFunc {
		return func(msg *message.Message) ([]*message.Message, error) {
			// 从 metadata 获取 topic，如果没有则使用 "unknown"
			topic := msg.Metadata.Get("topic")
			if topic == "" {
				topic = "unknown"
			}

			// 记录开始时间
			start := time.Now()

			// 增加正在处理的消息数
			messagesInFlight.WithLabelValues(serviceName, topic).Inc()
			defer messagesInFlight.WithLabelValues(serviceName, topic).Dec()

			// 调用处理器
			msgs, err := h(msg)

			// 记录处理耗时
			duration := time.Since(start).Seconds()
			processingDuration.WithLabelValues(serviceName, topic).Observe(duration)

			// 记录处理状态
			status := "success"
			if err != nil {
				status = "error"
			}
			messagesProcessed.WithLabelValues(serviceName, topic, status).Inc()

			return msgs, err
		}
	}
}
