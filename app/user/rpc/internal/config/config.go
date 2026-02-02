/**
 * @projectName: CampusHub
 * @package: config
 * @className: Config
 * @author: lijunqi
 * @description: 用户RPC服务配置定义
 * @date: 2026-01-30
 * @version: 1.0
 */

package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

// Config 用户服务配置
type Config struct {
	// RPC服务基础配置（包含服务发现、监听端口、日志等）
	zrpc.RpcServerConf

	// MySQL 数据库配置
	MySQL MySQLConf

	// BizRedis 业务Redis缓存配置（避免与go-zero内置Redis冲突）
	BizRedis redis.RedisConf

	// JWT 认证配置
	JWT JWTConf

	// SMS 短信服务配置
	SMS SMSConf
}

// MySQLConf MySQL数据库配置
type MySQLConf struct {
	// DataSource 数据库连接字符串
	// 格式: user:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=true&loc=Local
	DataSource string
}

// JWTConf JWT认证配置
type JWTConf struct {
	// AccessSecret AccessToken签名密钥（至少32字符）
	AccessSecret string
	// RefreshSecret RefreshToken签名密钥（至少32字符）
	RefreshSecret string
	// AccessExpire AccessToken过期时间（秒），默认7200（2小时）
	AccessExpire int64
	// RefreshExpire RefreshToken过期时间（秒），默认604800（7天）
	RefreshExpire int64
}

// SMSConf 短信服务配置
type SMSConf struct {
	// Provider 短信服务提供商：aliyun, tencent, mock
	Provider string `json:",optional,default=mock"`
	// AccessKey 访问密钥ID
	AccessKey string `json:",optional"`
	// SecretKey 访问密钥Secret
	SecretKey string `json:",optional"`
	// SignName 短信签名
	SignName string `json:",optional"`
	// TemplateCode 短信模板ID
	TemplateCode string `json:",optional"`
}
