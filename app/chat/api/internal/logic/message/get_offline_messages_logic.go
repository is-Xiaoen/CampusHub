// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package message

import (
	"context"
	"fmt"
	"strconv"

	"activity-platform/app/chat/api/internal/svc"
	"activity-platform/app/chat/api/internal/types"
	"activity-platform/app/chat/rpc/chat"

	"github.com/zeromicro/go-zero/core/logx"
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

func (l *GetOfflineMessagesLogic) GetOfflineMessages(req *types.GetOfflineMessagesReq) (resp *types.GetOfflineMessagesResp, err error) {
	// 从 context 中获取用户 ID（JWT Token 中的信息）
	userIdValue := l.ctx.Value("userId")
	if userIdValue == nil {
		l.Errorf("无法从 JWT Token 中获取用户 ID")
		return &types.GetOfflineMessagesResp{
			Code:    401,
			Message: "未授权：无法获取用户信息",
			Data: types.GetOfflineMessagesData{
				Messages: []types.MessageInfo{},
			},
		}, nil
	}

	// 将 userId 转换为字符串
	var userId string
	switch v := userIdValue.(type) {
	case string:
		userId = v
	case int64:
		userId = strconv.FormatInt(v, 10)
	case float64:
		userId = strconv.FormatInt(int64(v), 10)
	default:
		l.Errorf("无法解析用户 ID，类型: %T", userIdValue)
		return &types.GetOfflineMessagesResp{
			Code:    401,
			Message: "未授权：用户信息格式错误",
			Data: types.GetOfflineMessagesData{
				Messages: []types.MessageInfo{},
			},
		}, nil
	}

	// 调用 RPC 服务获取离线消息
	rpcResp, err := l.svcCtx.ChatRpc.GetOfflineMessages(l.ctx, &chat.GetOfflineMessagesReq{
		UserId:    userId,
		AfterTime: req.AfterTime,
	})
	if err != nil {
		l.Errorf("调用 RPC 获取离线消息失败: %v", err)
		return &types.GetOfflineMessagesResp{
			Code:    500,
			Message: fmt.Sprintf("获取离线消息失败: %v", err),
			Data: types.GetOfflineMessagesData{
				Messages: []types.MessageInfo{},
			},
		}, nil
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

	return &types.GetOfflineMessagesResp{
		Code:    0,
		Message: "success",
		Data: types.GetOfflineMessagesData{
			Messages: messages,
		},
	}, nil
}
