package config

import (
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

// Config API 网关配置
type Config struct {
	rest.RestConf

	// JWT 认证配置
	Auth AuthConfig

	// RPC 服务配置
	UserRpc     zrpc.RpcClientConf
	ActivityRpc zrpc.RpcClientConf
	ChatRpc     zrpc.RpcClientConf

	// CORS 跨域配置
	Cors CorsConfig

	// 限流配置
	RateLimit RateLimitConfig
}

// AuthConfig JWT 认证配置
type AuthConfig struct {
	AccessSecret string
	AccessExpire int64
}

// CorsConfig CORS 跨域配置
type CorsConfig struct {
	AllowOrigins []string
	AllowMethods []string
	AllowHeaders []string
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Rate  int
	Burst int
}
