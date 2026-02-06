package logic

import (
	"context"
	"time"

	"activity-platform/app/chat/model"
	"activity-platform/app/chat/rpc/chat"
	"activity-platform/app/chat/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type SaveMessageLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSaveMessageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SaveMessageLogic {
	return &SaveMessageLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// SaveMessage 保存消息
func (l *SaveMessageLogic) SaveMessage(in *chat.SaveMessageReq) (*chat.SaveMessageResp, error) {
	// 构造消息数据
	message := &model.Message{
		MessageID: in.MessageId,
		GroupID:   in.GroupId,
		SenderID:  in.SenderId,
		MsgType:   int8(in.MsgType),
		Content:   in.Content,
		ImageURL:  in.ImageUrl,
		Status:    1, // 1-正常
		CreatedAt: time.Now(),
	}

	// 插入数据库
	if err := l.svcCtx.MessageModel.Insert(l.ctx, message); err != nil {
		logx.Errorf("保存消息失败: %v", err)
		return &chat.SaveMessageResp{
			Success:   false,
			MessageId: in.MessageId,
		}, err
	}

	logx.Infof("消息保存成功: message_id=%s, group_id=%s, sender_id=%s",
		in.MessageId, in.GroupId, in.SenderId)

	return &chat.SaveMessageResp{
		Success:   true,
		MessageId: in.MessageId,
	}, nil
}
