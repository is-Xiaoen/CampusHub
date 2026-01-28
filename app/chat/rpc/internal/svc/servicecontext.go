package svc

import (
	"activity-platform/app/chat/rpc/internal/config"
)

// ServiceContext 聊天服务上下文
// TODO: 由 E同学 补充
type ServiceContext struct {
	Config config.Config
	// DB    *gorm.DB       // TODO: 数据库连接
	// Redis *redis.Client  // TODO: Redis连接
}

// NewServiceContext 创建服务上下文
func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config: c,
	}
}
