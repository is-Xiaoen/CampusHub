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

	// MySQL 数据库配置（必填）
	MySQL MySQLConf

	// BizRedis 业务Redis缓存配置（必填，避免与go-zero内置Redis冲突）
	BizRedis redis.RedisConf

	// JWT 认证配置（必填）
	JWT JWTConf

	Captcha CaptchaConf

	// ActivityRpc 活动服务RPC客户端配置（必填）
	ActivityRpc zrpc.RpcClientConf

	// SMS 短信服务配置（可选，默认 mock 模式）
	SMS SMSConf `json:",optional"`

	// Ocr OCR 识别服务配置（可选，不配置则禁用）
	Ocr OcrConf `json:",optional"`

	// Messaging 消息发布器配置（用于发布认证事件到 MQ）
	Messaging MessagingConf `json:",optional"`

	// Email 邮件服务配置
	Email EmailConf `json:",optional"`

	// Qiniu 七牛云配置
	Qiniu QiniuConf `json:",optional"`
}

// MessagingConf 消息发布器配置
type MessagingConf struct {
	// Redis 配置（用于 Watermill Redis Stream 发布）
	Redis RedisStreamConf
}

// RedisStreamConf Redis Stream 连接配置
type RedisStreamConf struct {
	Addr     string `json:",default=localhost:6379"`
	Password string `json:",optional"`
	DB       int    `json:",default=0"`
}

// QiniuConf 七牛云配置
type QiniuConf struct {
	AccessKey string
	SecretKey string
	Bucket    string
	Domain    string
	Zone      string
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

type CaptchaConf struct {
	CaptchaId  string
	CaptchaKey string
}

// SMSConf 短信服务配置
type SMSConf struct {
	// Provider 短信服务提供商：aliyun, tencent, mock
	Provider string `json:",default=mock"`
	// AccessKey 访问密钥ID（mock 模式可选）
	AccessKey string `json:",optional"`
	// SecretKey 访问密钥Secret（mock 模式可选）
	SecretKey string `json:",optional"`
	// SignName 短信签名（mock 模式可选）
	SignName string `json:",optional"`
	// TemplateCode 短信模板ID（mock 模式可选）
	TemplateCode string `json:",optional"`
}

// OcrConf OCR 识别服务配置
type OcrConf struct {
	// Tencent 腾讯云 OCR 配置
	Tencent TencentOcrConf `json:",optional"`
	// Aliyun 阿里云 OCR 配置
	Aliyun AliyunOcrConf `json:",optional"`
}

// TencentOcrConf 腾讯云 OCR 配置
type TencentOcrConf struct {
	// Enabled 是否启用
	Enabled bool `json:",default=false"`
	// SecretId 密钥ID
	SecretId string `json:",optional"`
	// SecretKey 密钥Key
	SecretKey string `json:",optional"`
	// Region 地域（如 ap-guangzhou）
	Region string `json:",optional"`
	// Endpoint 服务端点
	Endpoint string `json:",optional"`
	// Timeout 超时时间（秒）
	Timeout int `json:",default=30"`
}

type EmailConf struct {
	Host     string
	Port     int
	Username string
	Password string
	FromName string
	Subject  string
}

// AliyunOcrConf 阿里云 OCR 配置
type AliyunOcrConf struct {
	// Enabled 是否启用
	Enabled bool `json:",default=false"`
	// AccessKeyId 访问密钥ID
	AccessKeyId string `json:",optional"`
	// AccessKeySecret 访问密钥Secret
	AccessKeySecret string `json:",optional"`
	// Endpoint 服务端点
	Endpoint string `json:",optional"`
	// Timeout 超时时间（秒）
	Timeout int `json:",default=30"`
}
