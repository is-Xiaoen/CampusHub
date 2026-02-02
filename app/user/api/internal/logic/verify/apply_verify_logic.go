// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package verify

import (
	"context"

	"activity-platform/app/user/api/internal/svc"
	"activity-platform/app/user/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ApplyVerifyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 提交学生认证申请
func NewApplyVerifyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ApplyVerifyLogic {
	return &ApplyVerifyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ApplyVerifyLogic) ApplyVerify(req *types.ApplyVerifyReq) (resp *types.ApplyVerifyResp, err error) {
	// todo: add your logic here and delete this line

	return
}
