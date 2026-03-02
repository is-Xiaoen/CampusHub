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
	if in.UserId == 0 {
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

	return &chat.GetOfflineMessagesResp{
		Messages: messageList,
	}, nil
}

// batchFetchSenderNames 批量获取发送者昵称，返回 senderID -> nickname 映射
func (l *GetOfflineMessagesLogic) batchFetchSenderNames(messages []*model.Message) map[uint64]string {
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
