// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package notification

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

func (l *MarkAllReadLogic) MarkAllRead(req *types.MarkAllReadReq) (resp *types.MarkAllReadData, err error) {
	// 调用 RPC 服务标记全部已读
	rpcResp, err := l.svcCtx.ChatRpc.MarkAllRead(l.ctx, &chat.MarkAllReadReq{
		UserId: uint64(req.UserId),
	})
	if err != nil {
		l.Errorf("调用 RPC 标记全部已读失败: %v", err)
		// 处理 gRPC 错误
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.PermissionDenied:
				return nil, errorx.New(errorx.CodeNotificationPermissionDeny)
			default:
				return nil, errorx.NewWithMessage(errorx.CodeRPCError, "标记全部已读失败")
			}
		}
		return nil, errorx.NewWithMessage(errorx.CodeInternalError, "标记全部已读失败")
	}

	return &types.MarkAllReadData{
		UpdatedCount: rpcResp.AffectedCount,
	}, nil
}
