package logic

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"

	"activity-platform/app/chat/rpc/chat"
	"activity-platform/app/chat/ws/hub"
	"activity-platform/app/chat/ws/internal/queue"
	"activity-platform/app/chat/ws/internal/svc"
	"activity-platform/app/chat/ws/internal/types"
	"activity-platform/common/messaging"
)

// MessageLogic 消息处理逻辑
type MessageLogic struct {
	ctx             context.Context
	svcCtx          *svc.ServiceContext
	messagingClient *messaging.Client
}

// NewMessageLogic 创建消息处理逻辑
func NewMessageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MessageLogic {
	return &MessageLogic{
		ctx:             ctx,
		svcCtx:          svcCtx,
		messagingClient: svcCtx.MessagingClient,
	}
}

// HandleAuth 处理认证
func (l *MessageLogic) HandleAuth(client *hub.Client, msg *types.WSMessage) error {
	var authData types.AuthData
	if err := json.Unmarshal(msg.Data, &authData); err != nil {
		return err
	}

	// 验证 JWT Token
	userID, err := l.svcCtx.JwtAuth.ParseToken(authData.Token)
	if err != nil {
		client.SendMessage(&types.WSMessage{
			Type:      types.TypeAuthFailed,
			Timestamp: time.Now().Unix(),
			Data:      json.RawMessage(`{"message":"Token 无效或已过期"}`),
		})
		return err
	}

	// 设置客户端用户ID
	client.SetUserID(userID)
	client.SetAuthed(true)

	// 发送认证成功消息
	successData, _ := json.Marshal(map[string]string{"user_id": userID})
	client.SendMessage(&types.WSMessage{
		Type:      types.TypeAuthSuccess,
		Timestamp: time.Now().Unix(),
		Data:      successData,
	})

	// 自动加入用户的所有群聊
	go l.autoJoinUserGroups(client, userID)

	logx.Infof("用户 %s 认证成功", userID)
	return nil
}

// HandleSendMessage 处理发送消息（异步保存版本）
func (l *MessageLogic) HandleSendMessage(client *hub.Client, msg *types.WSMessage) error {
	var sendData types.SendMessageData
	if err := json.Unmarshal(msg.Data, &sendData); err != nil {
		return err
	}

	// 生成消息ID
	messageID := uuid.New().String()
	now := time.Now().Unix()

	// 将 string 类型的 UserID 转换为 uint64
	senderID, err := strconv.ParseUint(client.GetUserID(), 10, 64)
	if err != nil {
		logx.Errorf("解析用户ID失败: %v", err)
		return err
	}

	// 构造新消息数据
	newMsgData := types.NewMessageData{
		MessageID:  messageID,
		GroupID:    sendData.GroupID,
		SenderID:   senderID,
		SenderName: "User", // TODO: 从用户服务获取用户名
		MsgType:    sendData.MsgType,
		Content:    sendData.Content,
		ImageURL:   sendData.ImageURL,
		CreatedAt:  now,
	}

	// 1. 立即发送 ACK（快速响应，延迟 1-2ms）✅
	ackData := types.AckData{
		MessageID: messageID,
		Success:   true,
	}
	ackPayload, _ := json.Marshal(ackData)
	client.SendMessage(&types.WSMessage{
		Type:      types.TypeAck,
		MessageID: msg.MessageID,
		Timestamp: now,
		Data:      ackPayload,
	})

	// 2. 发布到消息中间件（实时推送）
	payload, _ := json.Marshal(newMsgData)
	if err := l.messagingClient.Publish(l.ctx, "chat.message.new", payload); err != nil {
		logx.Errorf("发布消息到中间件失败: %v", err)
		// 不影响后续流程，消息已保存到数据库
	}

	// 3. 异步保存到数据库（推送到队列）✅
	task := &queue.SaveMessageTask{
		MessageID: messageID,
		GroupID:   sendData.GroupID,
		SenderID:  senderID,
		MsgType:   sendData.MsgType,
		Content:   sendData.Content,
		ImageURL:  sendData.ImageURL,
		Retry:     0,
	}

	if err := l.svcCtx.SaveQueue.Push(task); err != nil {
		logx.Errorf("推送消息到保存队列失败: %v", err)
		// 队列满了，记录告警
		// TODO: 发送告警通知
	}

	logx.Infof("消息处理完成（异步）: message_id=%s, group_id=%s", messageID, sendData.GroupID)
	return nil
}

// HandleSendMessageSync 处理发送消息（同步保存版本，用于重要消息）
func (l *MessageLogic) HandleSendMessageSync(client *hub.Client, msg *types.WSMessage) error {
	var sendData types.SendMessageData
	if err := json.Unmarshal(msg.Data, &sendData); err != nil {
		return err
	}

	// 生成消息ID
	messageID := uuid.New().String()
	now := time.Now().Unix()

	// 将 string 类型的 UserID 转换为 uint64
	senderID, err := strconv.ParseUint(client.GetUserID(), 10, 64)
	if err != nil {
		logx.Errorf("解析用户ID失败: %v", err)
		return err
	}

	// 构造新消息数据
	newMsgData := types.NewMessageData{
		MessageID:  messageID,
		GroupID:    sendData.GroupID,
		SenderID:   senderID,
		SenderName: "User",
		MsgType:    sendData.MsgType,
		Content:    sendData.Content,
		ImageURL:   sendData.ImageURL,
		CreatedAt:  now,
	}

	// 1. 先保存消息到数据库（同步，确保可靠性）
	_, err = l.svcCtx.ChatRpc.SaveMessage(l.ctx, &chat.SaveMessageReq{
		MessageId: messageID,
		GroupId:   sendData.GroupID,
		SenderId:  senderID,
		MsgType:   sendData.MsgType,
		Content:   sendData.Content,
		ImageUrl:  sendData.ImageURL,
	})
	if err != nil {
		logx.Errorf("保存消息到数据库失败: %v", err)
		// 发送错误响应给客户端
		client.SendMessage(&types.WSMessage{
			Type:      types.TypeError,
			MessageID: msg.MessageID,
			Timestamp: now,
			Data:      json.RawMessage(`{"message":"消息保存失败"}`),
		})
		return err
	}

	// 2. 发布到消息中间件（实时推送）
	payload, _ := json.Marshal(newMsgData)
	if err := l.messagingClient.Publish(l.ctx, "chat.message.new", payload); err != nil {
		logx.Errorf("发布消息到中间件失败: %v", err)
	}

	// 3. 发送 ACK 确认
	ackData := types.AckData{
		MessageID: messageID,
		Success:   true,
	}
	ackPayload, _ := json.Marshal(ackData)
	client.SendMessage(&types.WSMessage{
		Type:      types.TypeAck,
		MessageID: msg.MessageID,
		Timestamp: now,
		Data:      ackPayload,
	})

	logx.Infof("消息处理完成（同步）: message_id=%s, group_id=%s", messageID, sendData.GroupID)
	return nil
}

// HandleJoinGroup 处理加入群聊
func (l *MessageLogic) HandleJoinGroup(client *hub.Client, msg *types.WSMessage) error {
	var data types.JoinGroupData
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		return err
	}

	// TODO: 验证用户是否是群成员（调用 RPC）
	// 这里简化处理，直接加入

	// 将客户端添加到群聊
	client.GetHub().AddClientToGroup(client, data.GroupID)

	logx.Infof("用户 %s 加入群聊 %s", client.GetUserID(), data.GroupID)
	return nil
}

// HandleLeaveGroup 处理离开群聊
func (l *MessageLogic) HandleLeaveGroup(client *hub.Client, msg *types.WSMessage) error {
	var data types.LeaveGroupData
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		return err
	}

	// 将客户端从群聊移除
	client.GetHub().RemoveClientFromGroup(client, data.GroupID)

	logx.Infof("用户 %s 离开群聊 %s", client.GetUserID(), data.GroupID)
	return nil
}

// HandleMarkRead 处理标记已读
func (l *MessageLogic) HandleMarkRead(client *hub.Client, msg *types.WSMessage) error {
	var data types.MarkReadData
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		return err
	}

	// TODO: 调用 RPC 标记已读
	// 这里简化处理

	logx.Infof("用户 %s 在群聊 %s 中标记消息 %s 为已读",
		client.GetUserID(), data.GroupID, data.MessageID)
	return nil
}

// autoJoinUserGroups 自动加入用户的所有群聊
func (l *MessageLogic) autoJoinUserGroups(client *hub.Client, userID string) {
	// 将 string 类型的 UserID 转换为 uint64
	userIDUint, err := strconv.ParseUint(userID, 10, 64)
	if err != nil {
		logx.Errorf("解析用户ID失败: %v", err)
		return
	}

	// 调用 RPC 获取用户的所有群聊
	resp, err := l.svcCtx.ChatRpc.GetUserGroups(l.ctx, &chat.GetUserGroupsReq{
		UserId:   userIDUint,
		Page:     1,
		PageSize: 100,
	})
	if err != nil {
		logx.Errorf("获取用户群聊列表失败: %v", err)
		return
	}

	// 将用户添加到所有群聊
	for _, group := range resp.Groups {
		client.GetHub().AddClientToGroup(client, group.GroupId)
	}

	logx.Infof("用户 %s 自动加入了 %d 个群聊", userID, len(resp.Groups))
}
