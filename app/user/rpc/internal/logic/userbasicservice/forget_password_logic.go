package userbasicservicelogic

import (
	"context"

	qqemaillogic "activity-platform/app/user/rpc/internal/logic/qqemail"
	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"
	"activity-platform/common/utils/encrypt"

	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type ForgetPasswordLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewForgetPasswordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ForgetPasswordLogic {
	return &ForgetPasswordLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 用户忘记密码
func (l *ForgetPasswordLogic) ForgetPassword(in *pb.ForgetPasswordReq) (*pb.ForgetPasswordResponse, error) {
	// 1. 校验新密码格式
	if !encrypt.ValidatePassword(in.NewPassword) {
		return nil, errorx.NewWithMessage(errorx.CodePasswordInvalid, "密码长度必须为8-20个字符，且包含至少3种字符（大写字母、小写字母、数字、特殊字符）")
	}

	if in.QqEmail == "" {
		return nil, errorx.ErrInvalidParams("qq_email不能为空")
	}
	user, err := l.svcCtx.UserModel.FindByQQEmail(l.ctx, in.QqEmail)
	if err != nil {
		l.Logger.Errorf("FindByQQEmail error: %v, email: %s", err, in.QqEmail)
		return nil, errorx.ErrDBError(err)
	}
	if user == nil {
		return nil, errorx.New(errorx.CodeUserNotFound)
	}

	// 3. 调用 CheckQQEmailLogic 校验验证码
	checkLogic := qqemaillogic.NewCheckQQEmailLogic(l.ctx, l.svcCtx)
	_, err = checkLogic.CheckQQEmail(&pb.CheckQQEmailReq{
		QqEmail: user.QQEmail,
		QqCode:  in.QqCode,
		Scene:   "forget_password",
	})
	if err != nil {
		return nil, err
	}

	// 4. 更新密码（哈希）
	newHash := encrypt.EncryptPassword(in.NewPassword)
	err = l.svcCtx.UserModel.UpdatePassword(l.ctx, user.UserID, newHash)
	if err != nil {
		l.Logger.Errorf("UpdatePassword error: %v, userId: %d", err, user.UserID)
		return nil, errorx.New(errorx.CodePasswordUpdateFailed)
	}

	return &pb.ForgetPasswordResponse{
		Success: true,
	}, nil
}
