package config

import (
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

// Config WebSocket 服务配置
type Config struct {
	rest.RestConf

	// Chat RPC 配置
	ChatRpc zrpc.RpcClientConf

	// Redis 配置
	Redis RedisConf

	// JWT 认证配置
	Auth AuthConf

	// WebSocket 配置
	WebSocket WebSocketConf
}

// RedisConf Redis 配置
type RedisConf struct {
	Host string
	Pass string
	DB   int
}

// AuthConf 认证配置
type AuthConf struct {
	AccessSecret string
	AccessExpire int64
}

// WebSocketConf WebSocket 配置
type WebSocketConf struct {
	// 最大连接数
	MaxConnections int `json:",default=10000"`
	// 读取超时（秒）
	ReadTimeout int `json:",default=60"`
	// 写入超时（秒）
	WriteTimeout int `json:",default=10"`
	// 心跳间隔（秒）
	HeartbeatInterval int `json:",default=30"`
}
