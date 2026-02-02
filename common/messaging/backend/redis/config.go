package redis

import (
	"crypto/tls"
	"time"
)

// Config Redis 后端配置
type Config struct {
	// Addr Redis 地址（必填）
	// 格式: "host:port"，例如 "localhost:6379"
	Addr string

	// Password Redis 密码（可选）
	Password string

	// DB 数据库编号（默认 0）
	DB int

	// PoolSize 连接池大小（默认 10）
	PoolSize int

	// MinIdleConns 最小空闲连接数（默认 5）
	MinIdleConns int

	// DialTimeout 连接超时（默认 5s）
	DialTimeout time.Duration

	// ReadTimeout 读超时（默认 3s）
	ReadTimeout time.Duration

	// WriteTimeout 写超时（默认 3s）
	WriteTimeout time.Duration

	// MaxRetries 最大重试次数（默认 3）
	MaxRetries int

	// TLSConfig TLS 配置（可选）
	TLSConfig *tls.Config
}

// DefaultConfig 返回默认配置
func DefaultConfig() Config {
	return Config{
		Addr:         "localhost:6379",
		Password:     "",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		MaxRetries:   3,
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.Addr == "" {
		return ErrInvalidConfig("addr是必填项")
	}
	if c.PoolSize <= 0 {
		c.PoolSize = 10
	}
	if c.MinIdleConns <= 0 {
		c.MinIdleConns = 5
	}
	if c.DialTimeout <= 0 {
		c.DialTimeout = 5 * time.Second
	}
	if c.ReadTimeout <= 0 {
		c.ReadTimeout = 3 * time.Second
	}
	if c.WriteTimeout <= 0 {
		c.WriteTimeout = 3 * time.Second
	}
	if c.MaxRetries < 0 {
		c.MaxRetries = 3
	}
	return nil
}
