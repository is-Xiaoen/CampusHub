package logic

import (
	"context"
	"time"

	"activity-platform/app/chat/model"
	"activity-platform/app/chat/rpc/chat"
	"activity-platform/app/chat/rpc/internal/svc"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CreateGroupLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateGroupLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateGroupLogic {
	return &CreateGroupLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// CreateGroup 创建群聊
func (l *CreateGroupLogic) CreateGroup(in *chat.CreateGroupReq) (*chat.CreateGroupResp, error) {
	// 1. 参数验证
	if in.ActivityId == 0 {
		return nil, status.Error(codes.InvalidArgument, "活动ID不能为空")
	}
	if in.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "群聊名称不能为空")
	}
	if in.OwnerId == 0 {
		return nil, status.Error(codes.InvalidArgument, "群主ID不能为空")
	}

	// 设置默认最大成员数
	maxMembers := in.MaxMembers
	if maxMembers <= 0 {
		maxMembers = 500
	}

	// 2. 生成群聊ID
	groupID := uuid.New().String()

	// 3. 创建群聊记录
	group := &model.Group{
		GroupID:     groupID,
		ActivityID:  in.ActivityId,
		Name:        in.Name,
		OwnerID:     in.OwnerId,
		Status:      1, // 1-正常
		MaxMembers:  maxMembers,
		MemberCount: 1, // 初始成员数为1（群主）
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := l.svcCtx.GroupModel.Insert(l.ctx, group); err != nil {
		l.Errorf("创建群聊失败: %v", err)
		return nil, status.Error(codes.Internal, "创建群聊失败")
	}

	// 4. 添加群主为成员
	member := &model.GroupMember{
		GroupID:  groupID,
		UserID:   in.OwnerId,
		Role:     2, // 2-群主
		Status:   1, // 1-正常
		JoinedAt: time.Now(),
	}

	if err := l.svcCtx.GroupMemberModel.Insert(l.ctx, member); err != nil {
		l.Errorf("添加群主失败: %v", err)
		// 这里应该回滚群聊创建，但为了简化先不处理
		return nil, status.Error(codes.Internal, "添加群主失败")
	}

	// 5. 返回结果
	return &chat.CreateGroupResp{
		GroupId: groupID,
	}, nil
}
