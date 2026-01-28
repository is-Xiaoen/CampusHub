package svc

import (
	"activity-platform/app/user/rpc/internal/config"
)

// ServiceContext 用户服务上下文
// TODO: 由 B同学 补充数据库、Redis客户端
type ServiceContext struct {
	Config config.Config
	// DB     *gorm.DB          // TODO: 添加数据库连接
	// Redis  *redis.Client     // TODO: 添加Redis连接
}

// NewServiceContext 创建服务上下文
func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config: c,
	}
}
