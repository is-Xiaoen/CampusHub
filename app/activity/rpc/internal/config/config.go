package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf // go-zero RPC 服务配置（含 Etcd、Log、Telemetry 等）
	// 注意：RpcServerConf 内置了 Redis redis.RedisKeyConf 用于 Auth 认证
	// 如果不开启 Auth，可以不配置

	// 数据存储
	MySQL    MySQLConfig     // MySQL 配置
	BizRedis redis.RedisConf // 业务 Redis（缓存、分布式锁、热门活动等）

	// RPC 客户端（服务间调用）
	UserRpc zrpc.RpcClientConf // User 服务 RPC 客户端

	// ==================== Elasticsearch 配置 ====================
	Elasticsearch ESConfig `json:",optional"` // ES 配置（可选，不配置则禁用搜索）

	// ==================== 高并发、熔断限流配置 ====================
	RegistrationLimit struct {
		Rate  int `json:",default=100"` // 每秒允许的请求数
		Burst int `json:",default=200"` // 突发容量
	}

	RegistrationBreaker struct {
		Name string `json:",default=activity-registration"` // 熔断器名称
	}
}

// ESConfig Elasticsearch 配置
//
// 配置说明：
// - Enabled: 是否启用 ES 搜索（false 时降级到 MySQL LIKE）
// - Hosts: ES 集群地址列表
// - IndexName: 活动索引名称
//
// 示例配置：
//
//	Elasticsearch:
//	  Enabled: true
//	  Hosts:
//	    - "http://localhost:9200"
//	  IndexName: activities
type ESConfig struct {
	Enabled       bool     `json:",default=false"`                   // 是否启用 ES
	Hosts         []string `json:",default=[http://localhost:9200]"` // ES 地址
	Username      string   `json:",optional"`                        // 认证用户名（可选）
	Password      string   `json:",optional"`                        // 认证密码（可选）
	IndexName     string   `json:",default=activities"`              // 索引名
	MaxRetries    int      `json:",default=3"`                       // 最大重试次数
	HealthTimeout int      `json:",default=5"`                       // 健康检查超时（秒）
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
