// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package message

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

type GetOfflineMessagesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取离线消息
func NewGetOfflineMessagesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetOfflineMessagesLogic {
	return &GetOfflineMessagesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetOfflineMessagesLogic) GetOfflineMessages(req *types.GetOfflineMessagesReq) (resp *types.GetOfflineMessagesData, err error) {
	// 调用 RPC 服务获取离线消息
	rpcResp, err := l.svcCtx.ChatRpc.GetOfflineMessages(l.ctx, &chat.GetOfflineMessagesReq{
		UserId:    uint64(req.UserId),
		AfterTime: req.AfterTime,
	})
	if err != nil {
		l.Errorf("调用 RPC 获取离线消息失败: %v", err)
		// 处理 gRPC 错误
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.NotFound:
				return nil, errorx.New(errorx.CodeMessageNotFound)
			default:
				return nil, errorx.NewWithMessage(errorx.CodeRPCError, "获取离线消息失败")
			}
		}
		return nil, errorx.NewWithMessage(errorx.CodeInternalError, "获取离线消息失败")
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

	return &types.GetOfflineMessagesData{
		Messages: messages,
	}, nil
}
