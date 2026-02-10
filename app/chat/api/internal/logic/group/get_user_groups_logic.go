// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package group

import (
	"context"

	"activity-platform/app/chat/api/internal/svc"
	"activity-platform/app/chat/api/internal/types"
	"activity-platform/app/chat/rpc/chat"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GetUserGroupsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewGetUserGroupsLogic 获取用户的群聊列表
func NewGetUserGroupsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserGroupsLogic {
	return &GetUserGroupsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUserGroupsLogic) GetUserGroups(req *types.GetUserGroupsReq) (resp *types.GetUserGroupsData, err error) {
	// 调用 RPC 服务获取用户群列表
	rpcResp, err := l.svcCtx.ChatRpc.GetUserGroups(l.ctx, &chat.GetUserGroupsReq{
		UserId:   uint64(req.UserId),
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		l.Errorf("调用 RPC 获取用户群列表失败: %v", err)
		// 处理 gRPC 错误
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.NotFound:
				return nil, errorx.New(errorx.CodeGroupNotFound)
			default:
				return nil, errorx.NewWithMessage(errorx.CodeRPCError, "获取用户群列表失败")
			}
		}
		return nil, errorx.NewWithMessage(errorx.CodeInternalError, "获取用户群列表失败")
	}

	// 转换群聊列表（RPC 现在返回完整的 UserGroupInfo）
	groups := make([]types.UserGroupInfo, 0, len(rpcResp.Groups))
	for _, group := range rpcResp.Groups {
		groups = append(groups, types.UserGroupInfo{
			GroupId:       group.GroupId,
			ActivityId:    int64(group.ActivityId),
			Name:          group.Name,
			OwnerId:       int64(group.OwnerId),
			MemberCount:   group.MemberCount,
			Status:        group.Status,
			Role:          getRoleString(group.Role),
			JoinedAt:      formatTimestamp(group.JoinedAt),
			LastMessage:   group.LastMessage,
			LastMessageAt: formatTimestamp(group.LastMessageAt),
		})
	}

	return &types.GetUserGroupsData{
		Groups:   groups,
		Total:    rpcResp.Total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}
