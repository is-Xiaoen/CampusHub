/**
 * @projectName: CampusHub
 * @package: config
 * @className: Config
 * @author: lijunqi
 * @description: 用户MQ服务配置定义（基于 Watermill Redis Stream）
 * @date: 2026-01-30
 * @version: 1.0
 */

package config

import (
	"time"

	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

// Config MQ服务配置
type Config struct {
	// 服务基础配置
	service.ServiceConf

	// MySQL 数据库配置
	MySQL MySQLConf

	// BizRedis 业务Redis配置（用于缓存操作）
	BizRedis redis.RedisConf

	// Messaging 消息中间件配置（复用 common/messaging）
	Messaging MessagingConf
}

// MySQLConf MySQL数据库配置
type MySQLConf struct {
	// DataSource 数据库连接字符串
	DataSource string
}

// MessagingConf 消息中间件配置（对应 common/messaging.Config）
type MessagingConf struct {
	// Redis 配置
	Redis RedisConf

	// Topic 订阅的主题（Stream Key）
	Topic string `json:",default=credit:events"`

	// ConsumerGroup 消费者组名称
	ConsumerGroup string `json:",default=user-mq-group"`

	// EnableMetrics 是否启用 Prometheus 指标
	EnableMetrics bool `json:",default=true"`

	// EnableGoZero 是否启用 Go-Zero trace_id 传播
	EnableGoZero bool `json:",default=true"`

	// Retry 重试配置
	Retry RetryConf
}

// RedisConf Redis 连接配置
type RedisConf struct {
	Addr     string `json:",default=localhost:6379"`
	Password string `json:",optional"`
	DB       int    `json:",default=0"`
}

// RetryConf 重试配置
type RetryConf struct {
	MaxRetries      int           `json:",default=3"`
	InitialInterval time.Duration `json:",default=100ms"`
	MaxInterval     time.Duration `json:",default=10s"`
	Multiplier      float64       `json:",default=2.0"`
}
