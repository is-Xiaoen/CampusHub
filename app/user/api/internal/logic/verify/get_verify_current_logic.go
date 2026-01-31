// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package verify

import (
	"context"

	"activity-platform/app/user/api/internal/svc"
	"activity-platform/app/user/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetVerifyCurrentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取当前认证进度
func NewGetVerifyCurrentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetVerifyCurrentLogic {
	return &GetVerifyCurrentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetVerifyCurrentLogic) GetVerifyCurrent() (resp *types.GetVerifyCurrentResp, err error) {
	// todo: add your logic here and delete this line

	return
}
