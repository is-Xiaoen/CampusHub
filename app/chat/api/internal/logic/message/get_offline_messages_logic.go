// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package message

import (
	"context"

	"activity-platform/app/chat/api/internal/svc"
	"activity-platform/app/chat/api/internal/types"
	"activity-platform/app/chat/rpc/chat"
	pb "activity-platform/app/user/rpc/pb/pb"
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

	// 批量获取发送者头像
	avatarMap := l.batchFetchSenderAvatars(rpcResp.Messages)

	// 转换消息列表
	messages := make([]types.MessageInfo, 0, len(rpcResp.Messages))
	for _, msg := range rpcResp.Messages {
		messages = append(messages, types.MessageInfo{
			MessageId:    msg.MessageId,
			GroupId:      msg.GroupId,
			SenderId:     int64(msg.SenderId),
			SenderName:   msg.SenderName,
			SenderAvatar: avatarMap[msg.SenderId],
			MsgType:      msg.MsgType,
			Content:      msg.Content,
			ImageUrl:     msg.ImageUrl,
			CreatedAt:    formatTimestamp(msg.CreatedAt),
		})
	}

	return &types.GetOfflineMessagesData{
		Messages: messages,
	}, nil
}

// batchFetchSenderAvatars 批量获取发送者头像，返回 senderID -> avatarUrl 映射
func (l *GetOfflineMessagesLogic) batchFetchSenderAvatars(msgs []*chat.Message) map[uint64]string {
	avatarMap := make(map[uint64]string)
	if len(msgs) == 0 {
		return avatarMap
	}

	// 收集唯一的发送者 ID
	seen := make(map[uint64]struct{})
	ids := make([]int64, 0)
	for _, msg := range msgs {
		if _, ok := seen[msg.SenderId]; !ok {
			seen[msg.SenderId] = struct{}{}
			ids = append(ids, int64(msg.SenderId))
		}
	}

	resp, err := l.svcCtx.UserRpc.GetGroupUser(l.ctx, &pb.GetGroupUserReq{Ids: ids})
	if err != nil {
		l.Errorf("批量获取用户头像失败: %v", err)
		return avatarMap
	}
	for _, u := range resp.Users {
		avatarMap[u.Id] = u.AvatarUrl
	}
	return avatarMap
}
