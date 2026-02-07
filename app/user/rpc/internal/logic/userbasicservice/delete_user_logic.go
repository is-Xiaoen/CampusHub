package userbasicservicelogic

import (
	"context"
	"fmt"
	"time"

	"activity-platform/app/user/model"
	qqemaillogic "activity-platform/app/user/rpc/internal/logic/qqemail"
	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"

	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
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
		return nil, errorx.ErrDBError(err)
	}
	if user == nil {
		return nil, errorx.New(errorx.CodeUserNotFound)
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
	// 修改邮箱为原邮箱+注销时间戳，避免唯一索引冲突并保留记录
	user.QQEmail = fmt.Sprintf("%s_%d", user.QQEmail, time.Now().Unix())

	err = l.svcCtx.UserModel.Update(l.ctx, user)
	if err != nil {
		l.Logger.Errorf("Update user status error: %v, userId: %d", err, in.UserId)
		return nil, errorx.New(errorx.CodeUserDeleteFailed)
	}

	return &pb.DeleteUserResponse{
		Success: true,
	}, nil
}
