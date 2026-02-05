package logic

import (
	"context"

	"activity-platform/app/chat/rpc/chat"
	"activity-platform/app/chat/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GetOfflineMessagesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetOfflineMessagesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetOfflineMessagesLogic {
	return &GetOfflineMessagesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetOfflineMessages 获取离线消息
func (l *GetOfflineMessagesLogic) GetOfflineMessages(in *chat.GetOfflineMessagesReq) (*chat.GetOfflineMessagesResp, error) {
	// 1. 参数验证
	if in.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "用户ID不能为空")
	}
	if in.AfterTime <= 0 {
		return nil, status.Error(codes.InvalidArgument, "时间戳必须大于0")
	}

	// 2. 查询离线消息
	messages, err := l.svcCtx.MessageModel.FindOfflineMessages(l.ctx, in.UserId, in.AfterTime)
	if err != nil {
		l.Errorf("查询离线消息失败: %v", err)
		return nil, status.Error(codes.Internal, "查询离线消息失败")
	}

	// 3. 构造响应
	messageList := make([]*chat.Message, 0, len(messages))
	for _, message := range messages {
		messageList = append(messageList, &chat.Message{
			MessageId:  message.MessageID,
			GroupId:    message.GroupID,
			SenderId:   message.SenderID,
			SenderName: message.SenderID, // 暂时使用 SenderID
			MsgType:    int32(message.MsgType),
			Content:    message.Content,
			ImageUrl:   message.ImageURL,
			Status:     int32(message.Status),
			CreatedAt:  message.CreatedAt.Unix(),
		})
	}

	return &chat.GetOfflineMessagesResp{
		Messages: messageList,
	}, nil
}
