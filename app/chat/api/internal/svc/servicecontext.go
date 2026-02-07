// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"activity-platform/app/chat/api/internal/config"
)

type ServiceContext struct {
	Config config.Config
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config: c,
	}
}
