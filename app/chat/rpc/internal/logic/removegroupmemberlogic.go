package logic

import (
	"context"
	"errors"

	"activity-platform/app/chat/rpc/chat"
	"activity-platform/app/chat/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type RemoveGroupMemberLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRemoveGroupMemberLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RemoveGroupMemberLogic {
	return &RemoveGroupMemberLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// RemoveGroupMember 移除群成员
func (l *RemoveGroupMemberLogic) RemoveGroupMember(in *chat.RemoveGroupMemberReq) (*chat.RemoveGroupMemberResp, error) {
	// 1. 参数验证
	if in.GroupId == "" {
		return nil, status.Error(codes.InvalidArgument, "群聊ID不能为空")
	}
	if in.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "用户ID不能为空")
	}

	// 2. 检查群聊是否存在
	_, err := l.svcCtx.GroupModel.FindOne(l.ctx, in.GroupId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "群聊不存在")
		}
		l.Errorf("查询群聊失败: %v", err)
		return nil, status.Error(codes.Internal, "查询群聊失败")
	}

	// 3. 检查成员是否存在
	member, err := l.svcCtx.GroupMemberModel.FindOne(l.ctx, in.GroupId, in.UserId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Error(codes.NotFound, "用户不在群中")
		}
		l.Errorf("查询群成员失败: %v", err)
		return nil, status.Error(codes.Internal, "查询群成员失败")
	}

	// 4. 检查是否为群主（群主不能被移除）
	if member.Role == 2 {
		return nil, status.Error(codes.PermissionDenied, "不能移除群主")
	}

	// 5. 移除群成员（软删除）
	if err := l.svcCtx.GroupMemberModel.Delete(l.ctx, in.GroupId, in.UserId); err != nil {
		l.Errorf("移除群成员失败: %v", err)
		return nil, status.Error(codes.Internal, "移除群成员失败")
	}

	// 6. 更新群成员数量
	if err := l.svcCtx.GroupModel.IncrementMemberCount(l.ctx, in.GroupId, -1); err != nil {
		l.Errorf("更新群成员数量失败: %v", err)
		// 不影响主流程，只记录日志
	}

	// 7. 返回结果
	return &chat.RemoveGroupMemberResp{
		Success: true,
	}, nil
}
