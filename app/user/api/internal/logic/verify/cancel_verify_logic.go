// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package verify

import (
	"context"

	"activity-platform/app/user/api/internal/svc"
	"activity-platform/app/user/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CancelVerifyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 取消认证申请
func NewCancelVerifyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CancelVerifyLogic {
	return &CancelVerifyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CancelVerifyLogic) CancelVerify(req *types.CancelVerifyReq) (resp *types.CancelVerifyResp, err error) {
	// todo: add your logic here and delete this line

	return
}
