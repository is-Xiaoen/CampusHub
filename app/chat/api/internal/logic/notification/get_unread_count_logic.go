// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package notification

import (
	"context"
	"strconv"

	"activity-platform/app/chat/api/internal/svc"
	"activity-platform/app/chat/api/internal/types"
	"activity-platform/app/chat/rpc/chat"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/status"
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

func (l *GetUnreadCountLogic) GetUnreadCount(req *types.GetUnreadCountReq) (resp *types.GetUnreadCountData, err error) {
	// 调用 RPC 服务获取未读数量
	rpcResp, err := l.svcCtx.ChatRpc.GetUnreadCount(l.ctx, &chat.GetUnreadCountReq{
		UserId: strconv.FormatInt(req.UserId, 10),
	})
	if err != nil {
		l.Errorf("调用 RPC 获取未读数量失败: %v", err)
		// 处理 gRPC 错误
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			default:
				return nil, errorx.NewWithMessage(errorx.CodeRPCError, "获取未读数量失败")
			}
		}
		return nil, errorx.NewWithMessage(errorx.CodeInternalError, "获取未读数量失败")
	}

	return &types.GetUnreadCountData{
		Count: rpcResp.UnreadCount,
	}, nil
}
