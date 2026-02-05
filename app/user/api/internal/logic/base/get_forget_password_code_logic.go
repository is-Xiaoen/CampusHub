// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package base

import (
	"context"

	"activity-platform/app/user/api/internal/svc"
	"activity-platform/app/user/rpc/client/qqemail"
	ctxUtils "activity-platform/common/utils/context"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetForgetPasswordCodeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取忘记密码QQ验证码
func NewGetForgetPasswordCodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetForgetPasswordCodeLogic {
	return &GetForgetPasswordCodeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetForgetPasswordCodeLogic) GetForgetPasswordCode() error {
	userId, err := ctxUtils.GetUserIdFromCtx(l.ctx)
	if err != nil {
		return err
	}

	_, err = l.svcCtx.QQEmailRpc.SendQQEmail(l.ctx, &qqemail.SendQQEmailReq{
		UserId: userId,
		Scene:  "forget_password",
	})
	if err != nil {
		return err
	}

	return nil
}
