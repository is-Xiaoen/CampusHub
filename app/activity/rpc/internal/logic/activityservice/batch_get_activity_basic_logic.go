package activityservicelogic

import (
	"context"

	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/activity/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type BatchGetActivityBasicLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewBatchGetActivityBasicLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchGetActivityBasicLogic {
	return &BatchGetActivityBasicLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 批量获取活动基本信息
func (l *BatchGetActivityBasicLogic) BatchGetActivityBasic(in *activity.BatchGetActivityBasicReq) (*activity.BatchGetActivityBasicResp, error) {
	// todo: add your logic here and delete this line

	return &activity.BatchGetActivityBasicResp{}, nil
}
