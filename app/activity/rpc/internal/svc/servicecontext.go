package svc

import (
	"activity-platform/app/activity/rpc/internal/config"
)

// ServiceContext 活动服务上下文
// TODO: 由 C同学、D同学 补充
type ServiceContext struct {
	Config config.Config
	// DB    *gorm.DB       // TODO: 数据库连接
	// Redis *redis.Client  // TODO: Redis连接
	// MQ    mq.Producer    // TODO: D同学添加消息队列
}

// NewServiceContext 创建服务上下文
func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config: c,
	}
}
