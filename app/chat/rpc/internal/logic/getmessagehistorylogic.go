package logic

import (
	"context"
	"strconv"

	"activity-platform/app/chat/model"
	"activity-platform/app/chat/rpc/chat"
	"activity-platform/app/chat/rpc/internal/svc"
	pb "activity-platform/app/user/rpc/pb/pb"

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

	// 3. 批量获取发送者昵称
	senderNameMap := l.batchFetchSenderNames(messages)

	// 4. 构造响应
	messageList := make([]*chat.Message, 0, len(messages))
	for _, message := range messages {
		senderName := senderNameMap[message.SenderID]
		if senderName == "" {
			senderName = strconv.FormatUint(message.SenderID, 10)
		}
		messageList = append(messageList, &chat.Message{
			MessageId:  message.MessageID,
			GroupId:    message.GroupID,
			SenderId:   message.SenderID,
			SenderName: senderName,
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

// batchFetchSenderNames 批量获取发送者昵称，返回 senderID -> nickname 映射
func (l *GetMessageHistoryLogic) batchFetchSenderNames(messages []*model.Message) map[uint64]string {
	nameMap := make(map[uint64]string)
	if l.svcCtx.UserBasicRpc == nil || len(messages) == 0 {
		return nameMap
	}

	// 收集唯一的发送者 ID
	seen := make(map[uint64]struct{})
	ids := make([]int64, 0)
	for _, msg := range messages {
		if _, ok := seen[msg.SenderID]; !ok {
			seen[msg.SenderID] = struct{}{}
			ids = append(ids, int64(msg.SenderID))
		}
	}

	resp, err := l.svcCtx.UserBasicRpc.GetGroupUser(l.ctx, &pb.GetGroupUserReq{Ids: ids})
	if err != nil {
		l.Errorf("批量获取用户昵称失败: %v", err)
		return nameMap
	}
	for _, u := range resp.Users {
		nameMap[u.Id] = u.Nickname
	}
	return nameMap
}
