package logic

import (
	"context"

	"activity-platform/app/chat/rpc/chat"
	"activity-platform/app/chat/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GetGroupMembersLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetGroupMembersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetGroupMembersLogic {
	return &GetGroupMembersLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetGroupMembers 获取群成员列表
func (l *GetGroupMembersLogic) GetGroupMembers(in *chat.GetGroupMembersReq) (*chat.GetGroupMembersResp, error) {
	// 1. 参数验证
	if in.GroupId == "" {
		return nil, status.Error(codes.InvalidArgument, "群聊ID不能为空")
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
		pageSize = 100 // 限制最大每页数量
	}

	// 2. 查询群成员列表
	members, total, err := l.svcCtx.GroupMemberModel.FindByGroupID(l.ctx, in.GroupId, page, pageSize)
	if err != nil {
		l.Errorf("查询群成员列表失败: %v", err)
		return nil, status.Error(codes.Internal, "查询群成员列表失败")
	}

	// 3. 构造响应
	memberList := make([]*chat.GroupMember, 0, len(members))
	for _, member := range members {
		memberList = append(memberList, &chat.GroupMember{
			UserId:   member.UserID,
			GroupId:  member.GroupID,
			Role:     int32(member.Role),
			JoinedAt: member.JoinedAt.Unix(),
		})
	}

	return &chat.GetGroupMembersResp{
		Members: memberList,
		Total:   int32(total),
	}, nil
}
