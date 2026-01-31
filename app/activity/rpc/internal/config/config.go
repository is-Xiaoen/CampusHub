package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf

	// MySQL配置
	MySQL struct {
		DataSource string
	}

	// Redis配置（限流器使用）
	Redis struct {
		redis.RedisConf
		DB int `json:",optional"`
	}

	// ==================== 高并发、熔断限流配置 ====================
	// 报名活动限流配置
	RegistrationLimit struct {
		Rate  int `json:",default=100"` // 每秒允许的请求数
		Burst int `json:",default=200"` // 突发容量
	}

	// 报名活动熔断配置
	RegistrationBreaker struct {
		Name string `json:",default=activity-registration"` // 熔断器名称
	}

	// ==================== RPC 客户端配置（服务间通信） ====================
	UserRpc zrpc.RpcClientConf
}
