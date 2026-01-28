package config

import "github.com/zeromicro/go-zero/zrpc"

// Config 活动服务配置
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

	// ==================== RPC 客户端配置（服务间通信） ====================
	// Activity 服务需要调用 User 服务获取组织者信息
	UserRpc zrpc.RpcClientConf

	// 消息队列配置（D同学使用）
	MQ struct {
		Brokers []string
		Topic   string
	}

	// ES搜索配置（C同学使用，可选）
	ES struct {
		Addresses []string
		Index     string
	}
}
