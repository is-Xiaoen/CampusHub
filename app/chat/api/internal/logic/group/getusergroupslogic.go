// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package group

import (
	"context"

	"activity-platform/app/chat/api/internal/svc"
	"activity-platform/app/chat/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserGroupsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取用户的群聊列表
func NewGetUserGroupsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserGroupsLogic {
	return &GetUserGroupsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUserGroupsLogic) GetUserGroups(req *types.GetUserGroupsReq) (resp *types.GetUserGroupsResp, err error) {
	// todo: add your logic here and delete this line

	return
}
