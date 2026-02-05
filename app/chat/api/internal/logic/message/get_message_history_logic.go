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

func (l *GetMessageHistoryLogic) GetMessageHistory(req *types.GetMessageHistoryReq) (resp *types.GetMessageHistoryResp, err error) {
	// 调用 RPC 服务获取消息历史
	rpcResp, err := l.svcCtx.ChatRpc.GetMessageHistory(l.ctx, &chat.GetMessageHistoryReq{
		GroupId:  req.GroupId,
		BeforeId: req.BeforeId,
		Limit:    req.Limit,
	})
	if err != nil {
		l.Errorf("调用 RPC 获取消息历史失败: %v", err)
		return &types.GetMessageHistoryResp{
			Code:    500,
			Message: fmt.Sprintf("获取消息历史失败: %v", err),
			Data: types.GetMessageHistoryData{
				Messages: []types.MessageInfo{},
				HasMore:  false,
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

	return &types.GetMessageHistoryResp{
		Code:    0,
		Message: "success",
		Data: types.GetMessageHistoryData{
			Messages: messages,
			HasMore:  rpcResp.HasMore,
		},
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
	return fmt.Sprintf("%d", timestamp)
}
