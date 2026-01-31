package logic

import (
	"context"

	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/activity/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type SubmitActivityLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSubmitActivityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SubmitActivityLogic {
	return &SubmitActivityLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// ==================== 状态操作接口 ====================
func (l *SubmitActivityLogic) SubmitActivity(in *activity.SubmitActivityReq) (*activity.SubmitActivityResp, error) {
	// todo: add your logic here and delete this line

	return &activity.SubmitActivityResp{}, nil
}
