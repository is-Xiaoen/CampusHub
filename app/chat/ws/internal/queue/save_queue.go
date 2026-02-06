package queue

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"activity-platform/app/chat/rpc/chat"
	"activity-platform/app/chat/rpc/chatservice"
	"github.com/zeromicro/go-zero/core/logx"
)

// SaveMessageTask 保存消息任务
type SaveMessageTask struct {
	MessageID string `json:"message_id"`
	GroupID   string `json:"group_id"`
	SenderID  string `json:"sender_id"`
	MsgType   int32  `json:"msg_type"`
	Content   string `json:"content"`
	ImageURL  string `json:"image_url"`
	Retry     int    `json:"retry"` // 重试次数
}

// SaveQueue 消息保存队列
type SaveQueue struct {
	queue          chan *SaveMessageTask
	chatRpc        chatservice.ChatService
	deadLetterChan chan *SaveMessageTask
	wg             sync.WaitGroup
	ctx            context.Context
	cancel         context.CancelFunc
}

var (
	ErrQueueFull = errors.New("save queue is full")
)

// NewSaveQueue 创建保存队列
func NewSaveQueue(chatRpc chatservice.ChatService, workerCount int) *SaveQueue {
	ctx, cancel := context.WithCancel(context.Background())
	sq := &SaveQueue{
		queue:          make(chan *SaveMessageTask, 10000), // 缓冲 10000 条消息
		chatRpc:        chatRpc,
		deadLetterChan: make(chan *SaveMessageTask, 1000),
		ctx:            ctx,
		cancel:         cancel,
	}

	// 启动工作协程
	for i := 0; i < workerCount; i++ {
		sq.wg.Add(1)
		go sq.worker(i)
	}

	// 启动死信队列处理协程
	sq.wg.Add(1)
	go sq.deadLetterHandler()

	logx.Infof("SaveQueue 启动成功，工作协程数：%d", workerCount)
	return sq
}

// Push 推送消息到队列
func (sq *SaveQueue) Push(task *SaveMessageTask) error {
	select {
	case sq.queue <- task:
		return nil
	case <-time.After(time.Second):
		logx.Errorf("SaveQueue 队列已满，消息推送超时: message_id=%s", task.MessageID)
		return ErrQueueFull
	}
}

// worker 工作协程
func (sq *SaveQueue) worker(id int) {
	defer sq.wg.Done()

	logx.Infof("SaveQueue worker %d 启动", id)

	for {
		select {
		case <-sq.ctx.Done():
			logx.Infof("SaveQueue worker %d 停止", id)
			return
		case task := <-sq.queue:
			sq.processTask(task)
		}
	}
}

// processTask 处理任务
func (sq *SaveQueue) processTask(task *SaveMessageTask) {
	startTime := time.Now()

	// 调用 RPC 保存消息
	resp, err := sq.chatRpc.SaveMessage(sq.ctx, &chat.SaveMessageReq{
		MessageId: task.MessageID,
		GroupId:   task.GroupID,
		SenderId:  task.SenderID,
		MsgType:   task.MsgType,
		Content:   task.Content,
		ImageUrl:  task.ImageURL,
	})

	duration := time.Since(startTime)

	if err != nil || !resp.Success {
		logx.Errorf("保存消息失败 (重试 %d/3): message_id=%s, error=%v, duration=%v",
			task.Retry, task.MessageID, err, duration)

		// 重试逻辑
		if task.Retry < 3 {
			task.Retry++
			// 指数退避：1s, 2s, 4s
			backoff := time.Duration(1<<task.Retry) * time.Second
			time.Sleep(backoff)
			sq.queue <- task // 重新入队
		} else {
			// 超过最大重试次数，推送到死信队列
			logx.Errorf("消息保存失败，推送到死信队列: message_id=%s", task.MessageID)
			sq.deadLetterChan <- task
		}
		return
	}

	logx.Infof("消息保存成功: message_id=%s, duration=%v", task.MessageID, duration)
}

// deadLetterHandler 死信队列处理
func (sq *SaveQueue) deadLetterHandler() {
	defer sq.wg.Done()

	for {
		select {
		case <-sq.ctx.Done():
			return
		case task := <-sq.deadLetterChan:
			// 记录到日志文件
			data, _ := json.Marshal(task)
			logx.Errorf("【死信队列】消息保存失败: %s", string(data))

			// TODO: 发送告警通知（钉钉、邮件等）
			// TODO: 写入专门的失败消息表，供人工处理
			// 示例：
			// alertService.SendAlert("消息保存失败", string(data))
			// failedMessageRepo.Insert(task)
		}
	}
}

// Stop 停止队列
func (sq *SaveQueue) Stop() {
	logx.Info("SaveQueue 正在停止...")
	sq.cancel()

	// 等待所有任务处理完成（最多等待 30 秒）
	done := make(chan struct{})
	go func() {
		sq.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logx.Info("SaveQueue 所有任务已处理完成")
	case <-time.After(30 * time.Second):
		logx.Error("SaveQueue 等待超时，强制关闭")
	}

	close(sq.queue)
	close(sq.deadLetterChan)
	logx.Info("SaveQueue 已停止")
}

// GetQueueLength 获取队列长度（用于监控）
func (sq *SaveQueue) GetQueueLength() int {
	return len(sq.queue)
}

// GetDeadLetterLength 获取死信队列长度（用于监控）
func (sq *SaveQueue) GetDeadLetterLength() int {
	return len(sq.deadLetterChan)
}
