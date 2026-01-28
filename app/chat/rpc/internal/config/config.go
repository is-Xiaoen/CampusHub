package config

import "github.com/zeromicro/go-zero/zrpc"

// Config 聊天服务配置
type Config struct {
	zrpc.RpcServerConf

	// 数据库配置
	MySQL struct {
		DataSource string
	}

	// Redis配置
	Redis struct {
		Host string
		Pass string
		DB   int
	}

	// 消息队列配置
	MQ struct {
		Brokers []string
		Topic   string
	}
}
