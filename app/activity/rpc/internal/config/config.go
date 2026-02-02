package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf // go-zero RPC 服务配置（含 Etcd、Log、Telemetry 等）

	// 数据存储
	MySQL MySQLConfig     // MySQL 配置
	Redis redis.RedisConf // Redis 配置（go-zero 内置结构）

	// RPC 客户端（服务间调用）
	UserRpc zrpc.RpcClientConf // User 服务 RPC 客户端
}

// MySQLConfig 数据库配置
type MySQLConfig struct {
	Host            string `json:",default=127.0.0.1"`
	Port            int    `json:",default=3306"`
	Username        string
	Password        string
	Database        string
	MaxOpenConns    int `json:",default=100"`  // 最大打开连接数
	MaxIdleConns    int `json:",default=10"`   // 最大空闲连接数
	ConnMaxLifetime int `json:",default=3600"` // 连接生命周期（秒）
}

// ==================== 高并发、熔断限流配置 ====================
type RegistrationLimit struct {
	Rate  int `json:",default=100"` // 每秒允许的请求数
	Burst int `json:",default=200"` // 突发容量
}
type RegistrationBreaker struct {
	Name string `json:",default=activity-registration"` // 熔断器名称
}
