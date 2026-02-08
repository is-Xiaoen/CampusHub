package logic

import (
	"context"
	"strconv"

	"activity-platform/app/chat/rpc/chat"
	"activity-platform/app/chat/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GetMessageHistoryLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetMessageHistoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMessageHistoryLogic {
	return &GetMessageHistoryLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetMessageHistory 获取历史消息
func (l *GetMessageHistoryLogic) GetMessageHistory(in *chat.GetMessageHistoryReq) (*chat.GetMessageHistoryResp, error) {
	// 1. 参数验证
	if in.GroupId == "" {
		return nil, status.Error(codes.InvalidArgument, "群聊ID不能为空")
	}

	// 设置默认查询数量
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// 2. 查询历史消息
	messages, err := l.svcCtx.MessageModel.FindByGroupID(l.ctx, in.GroupId, in.BeforeId, limit)
	if err != nil {
		l.Errorf("查询历史消息失败: %v", err)
		return nil, status.Error(codes.Internal, "查询历史消息失败")
	}

	// 3. 构造响应
	messageList := make([]*chat.Message, 0, len(messages))
	for _, message := range messages {
		messageList = append(messageList, &chat.Message{
			MessageId:  message.MessageID,
			GroupId:    message.GroupID,
			SenderId:   message.SenderID,
			SenderName: strconv.FormatUint(message.SenderID, 10), // 暂时使用 SenderID，后续可以调用用户服务获取
			MsgType:    int32(message.MsgType),
			Content:    message.Content,
			ImageUrl:   message.ImageURL,
			Status:     int32(message.Status),
			CreatedAt:  message.CreatedAt.Unix(),
		})
	}

	// 判断是否还有更多消息
	hasMore := len(messages) >= int(limit)

	return &chat.GetMessageHistoryResp{
		Messages: messageList,
		HasMore:  hasMore,
	}, nil
}
