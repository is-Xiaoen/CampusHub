// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"context"

	"activity-platform/app/user/api/internal/svc"
	"activity-platform/app/user/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateInterestLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 修改兴趣
func NewUpdateInterestLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateInterestLogic {
	return &UpdateInterestLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateInterestLogic) UpdateInterest(req *types.UpdateInterestReq) (resp *types.UpdateInterestResp, err error) {
	// todo: add your logic here and delete this line

	return
}
