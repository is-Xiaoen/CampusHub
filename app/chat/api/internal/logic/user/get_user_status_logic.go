// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"context"
	"fmt"
	"strconv"

	"activity-platform/app/chat/api/internal/svc"
	"activity-platform/app/chat/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserStatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取用户在线状态
func NewGetUserStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserStatusLogic {
	return &GetUserStatusLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUserStatusLogic) GetUserStatus(req *types.GetUserStatusReq) (resp *types.GetUserStatusResp, err error) {
	// 构建 Redis key
	key := fmt.Sprintf("user:status:%d", req.UserId)

	// 从 Redis 获取用户状态
	statusMap, err := l.svcCtx.Redis.HgetallCtx(l.ctx, key)
	if err != nil {
		l.Errorf("获取用户状态失败: %v", err)
		return &types.GetUserStatusResp{
			Code:    500,
			Message: "获取用户状态失败",
			Data: types.GetUserStatusData{
				IsOnline:      false,
				LastSeen:      0,
				LastOnlineAt:  0,
				LastOfflineAt: 0,
			},
		}, nil
	}

	// 解析状态数据
	isOnline := false
	if statusMap["is_online"] == "true" {
		isOnline = true
	}

	lastSeen, _ := strconv.ParseInt(statusMap["last_seen"], 10, 64)
	lastOnlineAt, _ := strconv.ParseInt(statusMap["last_online_at"], 10, 64)
	lastOfflineAt, _ := strconv.ParseInt(statusMap["last_offline_at"], 10, 64)

	return &types.GetUserStatusResp{
		Code:    0,
		Message: "success",
		Data: types.GetUserStatusData{
			IsOnline:      isOnline,
			LastSeen:      lastSeen,
			LastOnlineAt:  lastOnlineAt,
			LastOfflineAt: lastOfflineAt,
		},
	}, nil
}
