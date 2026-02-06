package userbasicservicelogic

import (
	"context"

	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"
	"activity-platform/common/utils/encrypt"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UpdatePasswordLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdatePasswordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdatePasswordLogic {
	return &UpdatePasswordLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 修改用户密码
func (l *UpdatePasswordLogic) UpdatePassword(in *pb.UpdatePasswordReq) (*pb.UpdatePasswordResponse, error) {
	// 1. 查询用户
	user, err := l.svcCtx.UserModel.FindByUserID(l.ctx, in.UserId)
	if err != nil {
		l.Logger.Errorf("FindByUserID error: %v, userId: %d", err, in.UserId)
		return nil, status.Error(codes.Internal, "internal error")
	}
	if user == nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	// 2. 校验原密码
	if !encrypt.ComparePassword(in.OriginPassword, user.Password) {
		return nil, status.Error(codes.InvalidArgument, "old password is incorrect")
	}

	// 3. 校验新密码格式
	if !encrypt.ValidatePassword(in.NewPassword) {
		return nil, status.Error(codes.InvalidArgument, "password must be 8-20 characters long and contain at least 3 types of characters (uppercase, lowercase, number, special)")
	}

	// 4. 加密新密码
	newPasswordHash := encrypt.EncryptPassword(in.NewPassword)

	// 5. 更新密码
	err = l.svcCtx.UserModel.UpdatePassword(l.ctx, user.UserID, newPasswordHash)
	if err != nil {
		l.Logger.Errorf("UpdatePassword error: %v, userId: %d", err, user.UserID)
		return nil, status.Error(codes.Internal, "failed to update password")
	}

	return &pb.UpdatePasswordResponse{
		Success: true,
	}, nil
}
