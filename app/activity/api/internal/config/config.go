// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package config

import (
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	rest.RestConf

	// JWT 认证配置
	Auth struct {
		AccessSecret string
		AccessExpire int64
	}

	// RPC 服务配置
	ActivityRpc zrpc.RpcClientConf // 活动服务 RPC 客户端
}
