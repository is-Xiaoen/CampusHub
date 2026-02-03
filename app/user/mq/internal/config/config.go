/**
 * @projectName: CampusHub
 * @package: config
 * @className: Config
 * @author: lijunqi
 * @description: 用户MQ服务配置定义（基于Redis Stream）
 * @date: 2026-01-30
 * @version: 1.0
 */

package config

import (
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

// Config MQ服务配置
type Config struct {
	// 服务基础配置
	service.ServiceConf

	// MySQL 数据库配置
	MySQL MySQLConf

	// BizRedis 业务Redis配置（同时用于缓存和 Stream，避免与go-zero内置冲突）
	BizRedis redis.RedisConf

	// Redis Stream 消费者配置
	// [待确认] 需要队友确定具体字段
	Stream StreamConf
}

// MySQLConf MySQL数据库配置
type MySQLConf struct {
	// DataSource 数据库连接字符串
	DataSource string
}

// StreamConf Redis Stream 配置
// [待确认] 以下字段需要和队友确认
type StreamConf struct {
	// ==================== 基础配置 ====================

	// Key Stream 的 Key 名称
	// 例如: "user:mq:stream" 或按主题分: "user:credit:stream"
	// [待确认] 是单个 Stream 还是多个 Stream？
	Key string

	// Group 消费者组名称
	// 同一个 Group 内的消费者会分摊消息（负载均衡）
	Group string

	// Consumer 当前消费者名称（集群部署时每个实例不同）
	// 例如: "consumer-1", "consumer-2"
	Consumer string

	// ==================== 可选配置 ====================

	// BatchSize 每次拉取的消息数量
	// [待确认] 队友的实现是否支持批量拉取？
	BatchSize int `json:",optional,default=10"`

	// BlockTimeout 阻塞等待超时时间（毫秒）
	// 0 表示一直阻塞直到有新消息
	// [待确认] 队友的实现使用什么阻塞策略？
	BlockTimeout int `json:",optional,default=5000"`

	// Workers 并发处理协程数
	// [待确认] 队友的实现是否支持并发消费？
	Workers int `json:",optional,default=1"`
}
