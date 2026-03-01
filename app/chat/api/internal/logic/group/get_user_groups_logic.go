// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package group

import (
	"context"

	"activity-platform/app/activity/rpc/activityservice"
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

	// 批量获取活动封面图（弱依赖，失败不影响主流程）
	coverUrlMap := make(map[int64]string)
	if l.svcCtx.ActivityRpc != nil && len(rpcResp.Groups) > 0 {
		ids := make([]int64, 0, len(rpcResp.Groups))
		for _, g := range rpcResp.Groups {
			if g.ActivityId != 0 {
				ids = append(ids, int64(g.ActivityId))
			}
		}
		if len(ids) > 0 {
			actResp, actErr := l.svcCtx.ActivityRpc.BatchGetActivityBasic(l.ctx, &activityservice.BatchGetActivityBasicReq{Ids: ids})
			if actErr != nil {
				l.Infof("[WARN] 批量获取活动基本信息失败，cover_url 降级为空: %v", actErr)
			} else {
				for _, a := range actResp.Activities {
					coverUrlMap[a.Id] = a.CoverUrl
				}
			}
		}
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
			CoverUrl:      coverUrlMap[int64(group.ActivityId)],
		})
	}

	return &types.GetUserGroupsData{
		Groups:   groups,
		Total:    rpcResp.Total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}
