package userbasicservicelogic

import (
	"context"

	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"
	"activity-platform/common/errorx"
	"activity-platform/common/utils/encrypt"

	"github.com/zeromicro/go-zero/core/logx"
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
		return nil, errorx.ErrDBError(err)
	}
	if user == nil {
		return nil, errorx.New(errorx.CodeUserNotFound)
	}

	// 2. 校验原密码
	if !encrypt.ComparePassword(in.OriginPassword, user.Password) {
		return nil, errorx.New(errorx.CodePasswordIncorrect)
	}

	// 3. 校验新密码格式
	if !encrypt.ValidatePassword(in.NewPassword) {
		return nil, errorx.NewWithMessage(errorx.CodePasswordInvalid, "密码长度必须为8-20个字符，且包含至少3种字符（大写字母、小写字母、数字、特殊字符）")
	}

	// 4. 加密新密码
	newPasswordHash := encrypt.EncryptPassword(in.NewPassword)

	// 5. 更新密码
	err = l.svcCtx.UserModel.UpdatePassword(l.ctx, user.UserID, newPasswordHash)
	if err != nil {
		l.Logger.Errorf("UpdatePassword error: %v, userId: %d", err, user.UserID)
		return nil, errorx.New(errorx.CodePasswordUpdateFailed)
	}

	return &pb.UpdatePasswordResponse{
		Success: true,
	}, nil
}
