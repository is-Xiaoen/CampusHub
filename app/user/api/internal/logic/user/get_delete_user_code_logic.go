// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"context"

	"activity-platform/app/user/api/internal/svc"
	"activity-platform/app/user/rpc/client/qqemail"
	ctxUtils "activity-platform/common/utils/context"

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
	userId, err := ctxUtils.GetUserIdFromCtx(l.ctx)
	if err != nil {
		return err
	}

	user, err := l.svcCtx.UserModel.FindByUserID(l.ctx, userId)
	if err != nil {
		l.Logger.Errorf("FindByUserID error: %v, userId: %d", err, userId)
		return err
	}
	if user == nil {
		return nil // User not found, but we shouldn't reveal this info or maybe return error? User exists in token context though.
	}

	_, err = l.svcCtx.QQEmailRpc.SendQQEmail(l.ctx, &qqemail.SendQQEmailReq{
		QqEmail: user.QQEmail,
		Scene:   "delete_user",
	})
	if err != nil {
		return err
	}

	return nil
}
