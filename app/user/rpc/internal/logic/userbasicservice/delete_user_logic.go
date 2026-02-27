package userbasicservicelogic

import (
	"context"
	"fmt"
	"time"

	"activity-platform/app/activity/rpc/activity"
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

	if l.svcCtx.ActivityRpc == nil {
		return nil, errorx.NewWithMessage(errorx.CodeServiceUnavailable, "活动服务不可用")
	}

	pendingResp, err := l.svcCtx.ActivityRpc.GetActivityList(l.ctx, &activity.GetActivityListRequest{
		Page:     1,
		PageSize: 1,
		Type:     "pending",
		UserId:   in.UserId,
	})
	if err != nil {
		return nil, errorx.ErrRPCError(err)
	}
	if pendingResp.GetTotal() > 0 {
		return nil, errorx.NewWithMessage(errorx.CodeForbidden, "存在待参加活动，无法注销")
	}

	page := int32(1)
	pageSize := int32(50)
	for {
		publishedResp, err := l.svcCtx.ActivityRpc.GetUserPublishedActivities(l.ctx, &activity.GetUserPublishedActivitiesReq{
			UserId:   in.UserId,
			Page:     page,
			PageSize: pageSize,
			Status:   -2,
		})
		if err != nil {
			return nil, errorx.ErrRPCError(err)
		}
		for _, item := range publishedResp.GetList() {
			if item.Status >= 0 && item.Status <= 3 {
				return nil, errorx.NewWithMessage(errorx.CodeForbidden, "存在未结束活动，无法注销")
			}
		}
		pagination := publishedResp.GetPagination()
		if pagination == nil || page >= pagination.GetTotalPages() {
			break
		}
		page++
	}

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
