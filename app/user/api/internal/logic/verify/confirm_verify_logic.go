// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package verify

import (
	"context"

	"activity-platform/app/user/api/internal/svc"
	"activity-platform/app/user/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ConfirmVerifyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 用户确认/修改认证信息
func NewConfirmVerifyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ConfirmVerifyLogic {
	return &ConfirmVerifyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ConfirmVerifyLogic) ConfirmVerify(req *types.ConfirmVerifyReq) (resp *types.ConfirmVerifyResp, err error) {
	// todo: add your logic here and delete this line

	return
}
