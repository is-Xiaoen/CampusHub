package metrics

import (
	"time"

	"CampusHub/common/messaging"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// PrometheusCollector Prometheus 指标收集器
type PrometheusCollector struct {
	// 发布指标
	publishDuration *prometheus.HistogramVec
	publishTotal    *prometheus.CounterVec
	publishErrors   *prometheus.CounterVec

	// 处理指标
	processDuration *prometheus.HistogramVec
	processTotal    *prometheus.CounterVec
	processErrors   *prometheus.CounterVec

	// 重试指标
	retryTotal   *prometheus.CounterVec
	retrySuccess *prometheus.CounterVec

	// DLQ 指标
	dlqTotal *prometheus.CounterVec
	dlqSize  *prometheus.GaugeVec

	// 消息指标
	messageCount *prometheus.CounterVec
	messageSize  *prometheus.HistogramVec
}

// NewPrometheusCollector 创建 Prometheus 指标收集器
func NewPrometheusCollector(namespace string) messaging.MetricsCollector {
	if namespace == "" {
		namespace = "messaging"
	}

	return &PrometheusCollector{
		// 发布指标
		publishDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "publish_duration_seconds",
				Help:      "Message publish duration in seconds",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"topic"},
		),
		publishTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "publish_total",
				Help:      "Total number of published messages",
			},
			[]string{"topic", "status"},
		),
		publishErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "publish_errors_total",
				Help:      "Total number of publish errors",
			},
			[]string{"topic"},
		),

		// 处理指标
		processDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "process_duration_seconds",
				Help:      "Message process duration in seconds",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"topic"},
		),
		processTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "process_total",
				Help:      "Total number of processed messages",
			},
			[]string{"topic", "status"},
		),
		processErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "process_errors_total",
				Help:      "Total number of process errors",
			},
			[]string{"topic"},
		),

		// 重试指标
		retryTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "retry_total",
				Help:      "Total number of retry attempts",
			},
			[]string{"topic", "attempt"},
		),
		retrySuccess: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "retry_success_total",
				Help:      "Total number of successful retries",
			},
			[]string{"topic"},
		),

		// DLQ 指标
		dlqTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "dlq_total",
				Help:      "Total number of messages moved to DLQ",
			},
			[]string{"topic", "reason"},
		),
		dlqSize: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "dlq_size",
				Help:      "Current size of DLQ",
			},
			[]string{"topic"},
		),

		// 消息指标
		messageCount: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "message_count_total",
				Help:      "Total number of messages",
			},
			[]string{"topic", "status"},
		),
		messageSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "message_size_bytes",
				Help:      "Message size in bytes",
				Buckets:   prometheus.ExponentialBuckets(100, 10, 6), // 100B to 10MB
			},
			[]string{"topic"},
		),
	}
}

// RecordPublish 记录消息发布指标
func (c *PrometheusCollector) RecordPublish(topic string, duration time.Duration, err error) {
	c.publishDuration.WithLabelValues(topic).Observe(duration.Seconds())

	status := "success"
	if err != nil {
		status = "error"
		c.publishErrors.WithLabelValues(topic).Inc()
	}

	c.publishTotal.WithLabelValues(topic, status).Inc()
}

// RecordPublishBatch 记录批量发布指标
func (c *PrometheusCollector) RecordPublishBatch(topic string, count int, duration time.Duration, err error) {
	c.publishDuration.WithLabelValues(topic).Observe(duration.Seconds())

	status := "success"
	if err != nil {
		status = "error"
		c.publishErrors.WithLabelValues(topic).Inc()
	}

	c.publishTotal.WithLabelValues(topic, status).Add(float64(count))
}

// RecordProcess 记录消息处理指标
func (c *PrometheusCollector) RecordProcess(topic string, duration time.Duration, err error) {
	c.processDuration.WithLabelValues(topic).Observe(duration.Seconds())

	status := "success"
	if err != nil {
		status = "error"
		c.processErrors.WithLabelValues(topic).Inc()
	}

	c.processTotal.WithLabelValues(topic, status).Inc()
}

// RecordRetry 记录重试指标
func (c *PrometheusCollector) RecordRetry(topic string, attempt int, success bool) {
	c.retryTotal.WithLabelValues(topic, string(rune(attempt))).Inc()

	if success {
		c.retrySuccess.WithLabelValues(topic).Inc()
	}
}

// RecordDLQ 记录死信队列指标
func (c *PrometheusCollector) RecordDLQ(topic string, reason string) {
	c.dlqTotal.WithLabelValues(topic, reason).Inc()
}

// IncrementMessageCount 增加消息计数
func (c *PrometheusCollector) IncrementMessageCount(topic string, status string) {
	c.messageCount.WithLabelValues(topic, status).Inc()
}

// ObserveMessageSize 观察消息大小
func (c *PrometheusCollector) ObserveMessageSize(topic string, size int) {
	c.messageSize.WithLabelValues(topic).Observe(float64(size))
}

// SetDLQSize 设置 DLQ 大小
func (c *PrometheusCollector) SetDLQSize(topic string, size int64) {
	c.dlqSize.WithLabelValues(topic).Set(float64(size))
}
