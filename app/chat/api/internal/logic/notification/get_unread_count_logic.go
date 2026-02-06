// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package notification

import (
	"context"
	"fmt"
	"strconv"

	"activity-platform/app/chat/api/internal/svc"
	"activity-platform/app/chat/api/internal/types"
	"activity-platform/app/chat/rpc/chat"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUnreadCountLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewGetUnreadCountLogic 获取未读数量
func NewGetUnreadCountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUnreadCountLogic {
	return &GetUnreadCountLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUnreadCountLogic) GetUnreadCount(req *types.GetUnreadCountReq) (resp *types.GetUnreadCountResp, err error) {
	// 调用 RPC 服务获取未读数量
	rpcResp, err := l.svcCtx.ChatRpc.GetUnreadCount(l.ctx, &chat.GetUnreadCountReq{
		UserId: strconv.FormatInt(req.UserId, 10),
	})
	if err != nil {
		l.Errorf("调用 RPC 获取未读数量失败: %v", err)
		return &types.GetUnreadCountResp{
			Code:    500,
			Message: fmt.Sprintf("获取未读数量失败: %v", err),
			Data: types.GetUnreadCountData{
				Count: 0,
			},
		}, nil
	}

	return &types.GetUnreadCountResp{
		Code:    0,
		Message: "success",
		Data: types.GetUnreadCountData{
			Count: rpcResp.UnreadCount,
		},
	}, nil
}
