package userbasicservicelogic

import (
	"context"

	qqemaillogic "activity-platform/app/user/rpc/internal/logic/qqemail"
	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"
	"activity-platform/common/utils/encrypt"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
		return nil, status.Error(codes.InvalidArgument, "password must be 8-20 characters and include at least 3 types (uppercase, lowercase, number, special)")
	}

	// 2. 查询用户
	user, err := l.svcCtx.UserModel.FindByUserID(l.ctx, in.UserId)
	if err != nil {
		l.Logger.Errorf("FindByUserID error: %v, userId: %d", err, in.UserId)
		return nil, status.Error(codes.Internal, "internal error")
	}
	if user == nil {
		return nil, status.Error(codes.NotFound, "user not found")
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
		return nil, status.Error(codes.Internal, "failed to update password")
	}

	return &pb.ForgetPasswordResponse{
		Success: true,
	}, nil
}
