package config

import (
	"time"

	"github.com/zeromicro/go-zero/zrpc"
)

// Config 聊天服务配置
type Config struct {
	zrpc.RpcServerConf

	// 数据库配置
	MySQL MySQLConf

	// 业务 Redis 配置（用于缓存等业务逻辑）
	// 注意：zrpc.RpcServerConf 中也有 Redis 字段（用于服务注册），所以这里重命名为 CacheRedis
	CacheRedis RedisConf

	// User RPC 客户端配置（MQ 消费者调用 User 服务处理信用分和 OCR）
	UserRpc zrpc.RpcClientConf

	// 消息中间件配置
	Messaging MessageConf
}

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries      int           `json:",default=3"`
	InitialInterval time.Duration `json:",default=100ms"`
	MaxInterval     time.Duration `json:",default=10s"`
	Multiplier      float64       `json:",default=2.0"`
}

// MySQLConf Mysql配置
type MySQLConf struct {
	// DataSource 数据库连接字符串
	// 格式: user:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=true&loc=Local
	DataSource string
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
