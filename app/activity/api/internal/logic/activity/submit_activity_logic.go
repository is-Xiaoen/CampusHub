// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package activity

import (
	"context"

	"activity-platform/app/activity/api/internal/svc"
	"activity-platform/app/activity/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SubmitActivityLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 提交审核
func NewSubmitActivityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SubmitActivityLogic {
	return &SubmitActivityLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SubmitActivityLogic) SubmitActivity(req *types.SubmitActivityReq) (resp *types.SubmitActivityResp, err error) {
	// todo: add your logic here and delete this line

	return
}
