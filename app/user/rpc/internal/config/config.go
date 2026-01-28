package config

import "github.com/zeromicro/go-zero/zrpc"

// Config 用户服务配置
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

	// JWT配置
	JWT struct {
		AccessSecret  string
		RefreshSecret string
		AccessExpire  int64
		RefreshExpire int64
	}

	// 短信配置
	SMS struct {
		Provider  string // aliyun, tencent
		AccessKey string
		SecretKey string
		SignName  string
		TemplateCode string
	}
}
