// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package message

import (
	"context"
	"fmt"

	"activity-platform/app/chat/api/internal/svc"
	"activity-platform/app/chat/api/internal/types"
	"activity-platform/app/chat/rpc/chat"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GetMessageHistoryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 查询消息历史
func NewGetMessageHistoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMessageHistoryLogic {
	return &GetMessageHistoryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetMessageHistoryLogic) GetMessageHistory(req *types.GetMessageHistoryReq) (resp *types.GetMessageHistoryData, err error) {
	// 调用 RPC 服务获取消息历史
	rpcResp, err := l.svcCtx.ChatRpc.GetMessageHistory(l.ctx, &chat.GetMessageHistoryReq{
		GroupId:  req.GroupId,
		BeforeId: req.BeforeId,
		Limit:    req.Limit,
	})
	if err != nil {
		l.Errorf("调用 RPC 获取消息历史失败: %v", err)
		// 处理 gRPC 错误
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.NotFound:
				return nil, errorx.New(errorx.CodeGroupNotFound)
			case codes.PermissionDenied:
				return nil, errorx.New(errorx.CodeMessageNotInGroup)
			default:
				return nil, errorx.NewWithMessage(errorx.CodeRPCError, "获取消息历史失败")
			}
		}
		return nil, errorx.NewWithMessage(errorx.CodeInternalError, "获取消息历史失败")
	}

	// 转换消息列表
	messages := make([]types.MessageInfo, 0, len(rpcResp.Messages))
	for _, msg := range rpcResp.Messages {
		messages = append(messages, types.MessageInfo{
			MessageId:  msg.MessageId,
			GroupId:    msg.GroupId,
			SenderId:   int64(msg.SenderId),
			SenderName: msg.SenderName,
			MsgType:    msg.MsgType,
			Content:    msg.Content,
			CreatedAt:  formatTimestamp(msg.CreatedAt),
		})
	}

	return &types.GetMessageHistoryData{
		Messages: messages,
		HasMore:  rpcResp.HasMore,
	}, nil
}

// formatTimestamp 将时间戳转换为字符串格式
func formatTimestamp(timestamp int64) string {
	if timestamp == 0 {
		return ""
	}
	return fmt.Sprintf("%d", timestamp)
}
