package logic

import (
	"context"

	"activity-platform/app/chat/rpc/chat"
	"activity-platform/app/chat/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GetUserGroupsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetUserGroupsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserGroupsLogic {
	return &GetUserGroupsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetUserGroups 获取用户群列表
func (l *GetUserGroupsLogic) GetUserGroups(in *chat.GetUserGroupsReq) (*chat.GetUserGroupsResp, error) {
	// 1. 参数验证
	if in.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "用户ID不能为空")
	}

	// 设置默认分页参数
	page := in.Page
	if page <= 0 {
		page = 1
	}
	pageSize := in.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// 2. 查询用户加入的群成员记录
	members, total, err := l.svcCtx.GroupMemberModel.FindByUserID(l.ctx, in.UserId, page, pageSize)
	if err != nil {
		l.Errorf("查询用户群列表失败: %v", err)
		return nil, status.Error(codes.Internal, "查询用户群列表失败")
	}

	// 3. 根据群ID查询群信息，并组装用户特定信息
	groupList := make([]*chat.UserGroupInfo, 0, len(members))
	for _, member := range members {
		group, err := l.svcCtx.GroupModel.FindOne(l.ctx, member.GroupID)
		if err != nil {
			l.Errorf("查询群聊信息失败, groupId=%s: %v", member.GroupID, err)
			continue // 跳过查询失败的群
		}

		// 4. 查询该群的最后一条消息
		lastMessage := ""
		lastMessageAt := int64(0)
		messages, err := l.svcCtx.MessageModel.FindByGroupID(l.ctx, member.GroupID, "", 1)
		if err == nil && len(messages) > 0 {
			lastMsg := messages[0]
			lastMessage = lastMsg.Content
			lastMessageAt = lastMsg.CreatedAt.Unix()
		}

		groupList = append(groupList, &chat.UserGroupInfo{
			GroupId:       group.GroupID,
			ActivityId:    group.ActivityID,
			Name:          group.Name,
			OwnerId:       group.OwnerID,
			Status:        int32(group.Status),
			MaxMembers:    group.MaxMembers,
			MemberCount:   group.MemberCount,
			CreatedAt:     group.CreatedAt.Unix(),
			Role:          int32(member.Role),
			JoinedAt:      member.JoinedAt.Unix(),
			LastMessage:   lastMessage,
			LastMessageAt: lastMessageAt,
		})
	}

	return &chat.GetUserGroupsResp{
		Groups: groupList,
		Total:  int32(total),
	}, nil
}
