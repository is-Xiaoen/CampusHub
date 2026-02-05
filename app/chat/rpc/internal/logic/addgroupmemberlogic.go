package logic

import (
	"context"
	"time"

	"activity-platform/app/chat/model"
	"activity-platform/app/chat/rpc/chat"
	"activity-platform/app/chat/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type AddGroupMemberLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAddGroupMemberLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddGroupMemberLogic {
	return &AddGroupMemberLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// AddGroupMember 添加群成员
func (l *AddGroupMemberLogic) AddGroupMember(in *chat.AddGroupMemberReq) (*chat.AddGroupMemberResp, error) {
	// 1. 参数验证
	if in.GroupId == "" {
		return nil, status.Error(codes.InvalidArgument, "群聊ID不能为空")
	}
	if in.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "用户ID不能为空")
	}

	// 设置默认角色
	role := in.Role
	if role <= 0 {
		role = 1 // 默认为普通成员
	}

	// 2. 检查群聊是否存在
	group, err := l.svcCtx.GroupModel.FindOne(l.ctx, in.GroupId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Error(codes.NotFound, "群聊不存在")
		}
		l.Errorf("查询群聊失败: %v", err)
		return nil, status.Error(codes.Internal, "查询群聊失败")
	}

	// 3. 检查群聊是否已满
	if group.MemberCount >= group.MaxMembers {
		return nil, status.Error(codes.ResourceExhausted, "群聊人数已满")
	}

	// 4. 检查用户是否已在群中
	existingMember, err := l.svcCtx.GroupMemberModel.FindOne(l.ctx, in.GroupId, in.UserId)
	if err == nil && existingMember != nil {
		return nil, status.Error(codes.AlreadyExists, "用户已在群中")
	}

	// 5. 添加群成员
	member := &model.GroupMember{
		GroupID:  in.GroupId,
		UserID:   in.UserId,
		Role:     int8(role),
		Status:   1, // 1-正常
		JoinedAt: time.Now(),
	}

	if err := l.svcCtx.GroupMemberModel.Insert(l.ctx, member); err != nil {
		l.Errorf("添加群成员失败: %v", err)
		return nil, status.Error(codes.Internal, "添加群成员失败")
	}

	// 6. 更新群成员数量
	if err := l.svcCtx.GroupModel.IncrementMemberCount(l.ctx, in.GroupId, 1); err != nil {
		l.Errorf("更新群成员数量失败: %v", err)
		// 不影响主流程，只记录日志
	}

	// 7. 返回结果
	return &chat.AddGroupMemberResp{
		Success: true,
	}, nil
}
