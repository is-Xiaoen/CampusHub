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

type GetRegisterCodeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取注册QQ验证码
func NewGetRegisterCodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetRegisterCodeLogic {
	return &GetRegisterCodeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetRegisterCodeLogic) GetRegisterCode() error {
	userId, err := ctxUtils.GetUserIdFromCtx(l.ctx)
	if err != nil {
		return err
	}

	_, err = l.svcCtx.QQEmailRpc.SendQQEmail(l.ctx, &qqemail.SendQQEmailReq{
		UserId: userId,
		Scene:  "register",
	})
	if err != nil {
		return err
	}

	return nil
}
