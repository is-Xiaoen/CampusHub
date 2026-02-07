// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"context"

	"activity-platform/app/chat/api/internal/svc"
	"activity-platform/app/chat/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserStatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取用户在线状态
func NewGetUserStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserStatusLogic {
	return &GetUserStatusLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUserStatusLogic) GetUserStatus(req *types.GetUserStatusReq) (resp *types.GetUserStatusResp, err error) {
	// todo: add your logic here and delete this line

	return
}
