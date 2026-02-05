package userbasicservicelogic

import (
	"context"

	"activity-platform/app/user/model"
	qqemaillogic "activity-platform/app/user/rpc/internal/logic/qqemail"
	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type DeleteUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteUserLogic {
	return &DeleteUserLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 用户注销自己
func (l *DeleteUserLogic) DeleteUser(in *pb.DeleteUserReq) (*pb.DeleteUserResponse, error) {
	// 1. 查询用户获取邮箱
	user, err := l.svcCtx.UserModel.FindByUserID(l.ctx, in.UserId)
	if err != nil {
		l.Logger.Errorf("FindByUserID error: %v, userId: %d", err, in.UserId)
		return nil, status.Error(codes.Internal, "internal error")
	}
	if user == nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	// 2. 调用 CheckQQEmailLogic 校验验证码
	checkLogic := qqemaillogic.NewCheckQQEmailLogic(l.ctx, l.svcCtx)
	_, err = checkLogic.CheckQQEmail(&pb.CheckQQEmailReq{
		QqEmail: user.QQEmail,
		QqCode:  in.QqCode,
		Scene:   "delete_user",
	})
	if err != nil {
		return nil, err
	}

	// 3. 更新用户状态为注销
	user.Status = model.UserStatusDeleted
	err = l.svcCtx.UserModel.Update(l.ctx, user)
	if err != nil {
		l.Logger.Errorf("Update user status error: %v, userId: %d", err, in.UserId)
		return nil, status.Error(codes.Internal, "failed to delete user")
	}

	return &pb.DeleteUserResponse{
		Success: true,
	}, nil
}
