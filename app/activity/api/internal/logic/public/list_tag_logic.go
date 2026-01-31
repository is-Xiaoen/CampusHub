// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package public

import (
	"context"

	"activity-platform/app/activity/api/internal/svc"
	"activity-platform/app/activity/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListTagLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 标签列表
func NewListTagLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListTagLogic {
	return &ListTagLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListTagLogic) ListTag(req *types.ListTagReq) (resp *types.ListTagResp, err error) {
	// todo: add your logic here and delete this line

	return
}
