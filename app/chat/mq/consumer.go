package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"activity-platform/app/chat/mq/consumer"
	"activity-platform/app/chat/mq/internal/config"
	"activity-platform/app/chat/mq/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
)

var configFile = flag.String("f", "etc/consumer.yaml", "配置文件路径")

func main() {
	flag.Parse()

	// 加载配置
	var c config.Config
	conf.MustLoad(*configFile, &c)

	// 初始化日志
	logx.MustSetup(logx.LogConf{
		ServiceName: c.Name,
		Mode:        c.Mode,
	})
	defer logx.Close()

	// 创建服务上下文
	svcCtx := svc.NewServiceContext(c)
	defer svcCtx.MsgClient.Close()

	// 注册消费者
	registerConsumers(svcCtx)

	// 启动消息路由
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 监听系统信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// 在 goroutine 中运行消息路由
	go func() {
		logx.Info("消费者服务启动中...")
		if err := svcCtx.MsgClient.Run(ctx); err != nil {
			logx.Errorf("消息路由停止: %v", err)
		}
	}()

	// 等待 Router 启动
	<-svcCtx.MsgClient.Running()
	logx.Info("消费者服务已启动")

	// 等待关闭信号
	<-sigChan
	logx.Info("收到关闭信号，正在优雅关闭...")
	cancel()

	logx.Info("消费者服务已关闭")
}

// registerConsumers 注册所有消费者
func registerConsumers(svcCtx *svc.ServiceContext) {
	// 1. 活动创建事件消费者
	activityCreatedConsumer := consumer.NewActivityCreatedConsumer(svcCtx.ChatRpc)
	activityCreatedConsumer.Subscribe(svcCtx.MsgClient)

	// 2. 用户报名成功事件消费者
	memberJoinedConsumer := consumer.NewActivityMemberJoinedConsumer(svcCtx.ChatRpc)
	memberJoinedConsumer.Subscribe(svcCtx.MsgClient)

	// 3. 用户取消报名事件消费者
	memberLeftConsumer := consumer.NewActivityMemberLeftConsumer(svcCtx.ChatRpc)
	memberLeftConsumer.Subscribe(svcCtx.MsgClient)

	fmt.Println("✅ 已注册 3 个消费者:")
	fmt.Println("  - activity.created -> chat-auto-create-group")
	fmt.Println("  - activity.member.joined -> chat-auto-add-member")
	fmt.Println("  - activity.member.left -> chat-auto-remove-member")
}
