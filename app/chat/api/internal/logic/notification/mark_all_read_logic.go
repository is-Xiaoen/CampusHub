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

type MarkAllReadLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 标记全部已读
func NewMarkAllReadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MarkAllReadLogic {
	return &MarkAllReadLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *MarkAllReadLogic) MarkAllRead(req *types.MarkAllReadReq) (resp *types.MarkAllReadResp, err error) {
	// 调用 RPC 服务标记全部已读
	rpcResp, err := l.svcCtx.ChatRpc.MarkAllRead(l.ctx, &chat.MarkAllReadReq{
		UserId: strconv.FormatInt(req.UserId, 10),
	})
	if err != nil {
		l.Errorf("调用 RPC 标记全部已读失败: %v", err)
		return &types.MarkAllReadResp{
			Code:    500,
			Message: fmt.Sprintf("标记全部已读失败: %v", err),
			Data: types.MarkAllReadData{
				UpdatedCount: 0,
			},
		}, nil
	}

	return &types.MarkAllReadResp{
		Code:    0,
		Message: "全部标记成功",
		Data: types.MarkAllReadData{
			UpdatedCount: rpcResp.AffectedCount,
		},
	}, nil
}
