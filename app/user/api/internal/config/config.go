/**
 * @projectName: CampusHub
 * @package: config
 * @className: Config
 * @author: lijunqi
 * @description: User API 服务配置定义
 * @date: 2026-01-30
 * @version: 1.1
 */

package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

// Config User API 服务配置
// 注意：OCR 配置已迁移到 RPC 层（user.yaml）
type Config struct {
	rest.RestConf

	// JWT 认证配置
	Auth struct {
		AccessSecret string
		AccessExpire int64
	}

	// User RPC 服务配置
	UserRpc zrpc.RpcClientConf

	// BizRedis 业务Redis配置（避免与go-zero内置Redis冲突）
	BizRedis redis.RedisConf

	// MySQL 配置
	MySQL struct {
		DataSource string
	}
}
