// ============================================================================
// 配置结构定义
// ============================================================================

package config

import "github.com/zeromicro/go-zero/zrpc"

// Config 服务配置结构
type Config struct {
	zrpc.RpcServerConf        // 嵌入 RPC 服务器基础配置
	MySQL              MySQLConfig
}

// MySQLConfig MySQL 配置
type MySQLConfig struct {
	DataSource string
}
