package config

import (
	"time"

	"github.com/zeromicro/go-zero/zrpc"
)

// Config 消费者服务配置
type Config struct {
	Name string
	Mode string

	// Chat RPC 客户端配置
	ChatRpc zrpc.RpcClientConf

	// Redis配置
	Redis RedisConf

	// 消息中间件配置
	Messaging MessageConf
}

// RedisConf Redis配置
type RedisConf struct {
	Host string
	Pass string
	DB   int
}

// MessageConf 消息中间件配置
type MessageConf struct {
	ServiceName   string      // 服务名称（用作消费者组名）
	EnableMetrics bool        // 启用指标
	EnableGoZero  bool        // 启用 Go-Zero trace_id 传播
	Retry         RetryConfig // 重试配置
}

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries      int           `json:",default=3"`
	InitialInterval time.Duration `json:",default=100ms"`
	MaxInterval     time.Duration `json:",default=10s"`
	Multiplier      float64       `json:",default=2.0"`
}
