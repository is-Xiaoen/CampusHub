// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package message

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
	// 将请求参数中的 UserId 转换为字符串
	userId := strconv.FormatInt(req.UserId, 10)

	// 调用 RPC 服务获取离线消息
	rpcResp, err := l.svcCtx.ChatRpc.GetOfflineMessages(l.ctx, &chat.GetOfflineMessagesReq{
		UserId:    userId,
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
			SenderId:   mustParseInt64(msg.SenderId),
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

// mustParseInt64 将字符串转换为 int64，失败返回 0
func mustParseInt64(s string) int64 {
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}

// formatTimestamp 将时间戳转换为字符串格式
func formatTimestamp(timestamp int64) string {
	if timestamp == 0 {
		return ""
	}
	return strconv.FormatInt(timestamp, 10)
}
