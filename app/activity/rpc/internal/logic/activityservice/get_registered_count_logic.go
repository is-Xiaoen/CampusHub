package activityservicelogic

import (
	"context"

	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/activity/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetRegisteredCountLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetRegisteredCountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetRegisteredCountLogic {
	return &GetRegisteredCountLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetRegisteredCount 获取报名数量
func (l *GetRegisteredCountLogic) GetRegisteredCount(in *activity.GetRegisteredCountRequest) (*activity.GetRegisteredCountResponse, error) {
	// todo: add your logic here and delete this line

	return &activity.GetRegisteredCountResponse{}, nil
}
