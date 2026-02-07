// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	rest.RestConf
	Auth struct {
		AccessSecret string
		AccessExpire int64
	}
	ChatRpc   zrpc.RpcClientConf
	UserRpc   zrpc.RpcClientConf `json:",optional"`
	Redis     redis.RedisConf
	WebSocket WebSocketConf `json:",optional"` // WebSocket 配置（可选）
}

// WebSocketConf WebSocket 配置
type WebSocketConf struct {
	// 是否启用 WebSocket（默认启用）
	Enabled bool `json:",default=true"`
	// 最大连接数
	MaxConnections int `json:",default=10000"`
	// 读取超时（秒）
	ReadTimeout int `json:",default=60"`
	// 写入超时（秒）
	WriteTimeout int `json:",default=10"`
	// 心跳间隔（秒）
	HeartbeatInterval int `json:",default=30"`
}
