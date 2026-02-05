package logic

import (
	"context"
	"errors"

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
	userID := in.GetUserId()
	if userID <= 0 {
		return nil, errors.New("用户ID无效")
	}

	count, err := l.svcCtx.ActivityRegistrationModel.CountByUserID(l.ctx, uint64(userID))
	if err != nil {
		return nil, err
	}

	return &activity.GetRegisteredCountResponse{
		Count: int32(count),
	}, nil
}
