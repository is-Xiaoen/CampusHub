package logic

import (
	"context"

	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/activity/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetActivityBasicLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetActivityBasicLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetActivityBasicLogic {
	return &GetActivityBasicLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// ==================== 内部接口（供其他微服务调用）====================
func (l *GetActivityBasicLogic) GetActivityBasic(in *activity.GetActivityBasicReq) (*activity.GetActivityBasicResp, error) {
	// todo: add your logic here and delete this line

	return &activity.GetActivityBasicResp{}, nil
}
