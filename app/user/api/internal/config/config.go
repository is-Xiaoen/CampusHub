/**
 * @projectName: CampusHub
 * @package: config
 * @className: Config
 * @author: lijunqi
 * @description: User API 服务配置定义
 * @date: 2026-01-30
 * @version: 1.0
 */

package config

import (
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

// Config User API 服务配置
type Config struct {
	rest.RestConf

	// JWT 认证配置
	Auth struct {
		AccessSecret string
		AccessExpire int64
	}

	// User RPC 服务配置
	UserRpc zrpc.RpcClientConf
}
