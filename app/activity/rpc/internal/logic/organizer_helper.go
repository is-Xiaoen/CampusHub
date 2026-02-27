package logic

import (
	"context"

	"activity-platform/app/activity/rpc/internal/svc"
	userpb "activity-platform/app/user/rpc/pb/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

// OrganizerInfo 组织者最新信息（从用户服务实时获取）
type OrganizerInfo struct {
	Name   string
	Avatar string
}

// fetchOrganizerMap 批量获取组织者最新的昵称和头像
//
// 活动表中 organizer_name/organizer_avatar 是创建时写入的快照，
// 用户修改头像/昵称后不会自动同步到活动表。
// 此函数通过 User RPC 实时获取最新信息，用于覆盖过期的快照数据。
// RPC 调用失败时返回空 map，调用方自动降级使用快照数据。
func fetchOrganizerMap(ctx context.Context, svcCtx *svc.ServiceContext, organizerIDs []uint64) map[uint64]*OrganizerInfo {
	result := make(map[uint64]*OrganizerInfo)
	if len(organizerIDs) == 0 {
		return result
	}

	// 去重
	seen := make(map[uint64]bool)
	uniqueIDs := make([]int64, 0, len(organizerIDs))
	for _, id := range organizerIDs {
		if id > 0 && !seen[id] {
			seen[id] = true
			uniqueIDs = append(uniqueIDs, int64(id))
		}
	}

	if len(uniqueIDs) == 0 {
		return result
	}

	// 复用 GetGroupUser 接口批量获取用户信息
	resp, err := svcCtx.UserBasicRpc.GetGroupUser(ctx, &userpb.GetGroupUserReq{
		Ids: uniqueIDs,
	})
	if err != nil {
		logx.WithContext(ctx).Infof("[fetchOrganizerMap] 获取组织者信息失败（降级使用快照数据）: %v", err)
		return result
	}

	for _, user := range resp.Users {
		result[user.Id] = &OrganizerInfo{
			Name:   user.Nickname,
			Avatar: user.AvatarUrl,
		}
	}

	return result
}
