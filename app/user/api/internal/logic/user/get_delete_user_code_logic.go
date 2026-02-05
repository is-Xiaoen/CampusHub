// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"context"

	"activity-platform/app/user/api/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetDeleteUserCodeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取注销用户QQ验证码
func NewGetDeleteUserCodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDeleteUserCodeLogic {
	return &GetDeleteUserCodeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetDeleteUserCodeLogic) GetDeleteUserCode() error {
	// todo: add your logic here and delete this line

	return nil
}
