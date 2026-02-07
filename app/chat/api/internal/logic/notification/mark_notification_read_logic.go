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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MarkNotificationReadLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 标记已读
func NewMarkNotificationReadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MarkNotificationReadLogic {
	return &MarkNotificationReadLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *MarkNotificationReadLogic) MarkNotificationRead(req *types.MarkNotificationReadReq) (resp *types.MarkNotificationReadData, err error) {
	// 调用 RPC 服务标记通知已读
	rpcResp, err := l.svcCtx.ChatRpc.MarkNotificationRead(l.ctx, &chat.MarkNotificationReadReq{
		UserId:          strconv.FormatInt(req.UserId, 10),
		NotificationIds: req.NotificationIds,
	})
	if err != nil {
		l.Errorf("调用 RPC 标记通知已读失败: %v", err)
		// 处理 gRPC 错误
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.NotFound:
				return nil, errorx.New(errorx.CodeNotificationNotFound)
			case codes.PermissionDenied:
				return nil, errorx.New(errorx.CodeNotificationPermissionDeny)
			default:
				return nil, errorx.NewWithMessage(errorx.CodeRPCError, "标记已读失败")
			}
		}
		return nil, errorx.NewWithMessage(errorx.CodeInternalError, "标记已读失败")
	}

	return &types.MarkNotificationReadData{
		UpdatedCount: rpcResp.UpdatedCount,
	}, nil
}
