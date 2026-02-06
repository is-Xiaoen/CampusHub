/**
 * @projectName: CampusHub
 * @package: main
 * @className: UserMQ
 * @author: lijunqi
 * @description: 用户MQ消费者服务入口（基于 Watermill Redis Stream）
 * @date: 2026-01-30
 * @version: 1.0
 */

package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"activity-platform/app/user/mq/internal/config"
	"activity-platform/app/user/mq/internal/handler"
	"activity-platform/app/user/mq/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
)

// 配置文件路径
var configFile = flag.String("f", "etc/user-mq.yaml", "the config file")

func main() {
	flag.Parse()

	// 加载配置
	var c config.Config
	conf.MustLoad(*configFile, &c)

	// 设置日志
	logx.MustSetup(c.Log)

	// 创建服务上下文（包含 DB、Redis、MsgClient）
	svcCtx, err := svc.NewServiceContext(c)
	if err != nil {
		logx.Errorf("创建服务上下文失败: %v", err)
		os.Exit(1)
	}

	// 创建消息处理器
	handlers := handler.NewHandlers(svcCtx)

	// ================================================================
	// 注册消息订阅（使用 Watermill）
	// ================================================================

	// 订阅信用事件主题
	svcCtx.MsgClient.Subscribe(
		c.Messaging.Topic,      // 订阅的主题（如 "credit:events"）
		"credit-event-handler", // 处理器名称
		handlers.WatermillHandler(),
	)

	logx.Infof("User MQ 服务启动中...")
	logx.Infof("订阅主题: %s, 消费者组: %s", c.Messaging.Topic, c.Messaging.ConsumerGroup)

	// ================================================================
	// 启动消息路由（阻塞运行）
	// ================================================================

	// 创建可取消的上下文
	ctx, cancel := context.WithCancel(context.Background())

	// 监听退出信号
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		logx.Info("收到退出信号，正在停止 User MQ 服务...")
		cancel()
	}()

	// 等待 Router 启动
	go func() {
		<-svcCtx.MsgClient.Running()
		logx.Info("User MQ 服务启动成功")
	}()

	// 启动 Router（阻塞）
	if err := svcCtx.MsgClient.Run(ctx); err != nil {
		logx.Errorf("消息路由运行失败: %v", err)
	}

	// 关闭客户端
	if err := svcCtx.MsgClient.Close(); err != nil {
		logx.Errorf("关闭消息客户端失败: %v", err)
	}

	logx.Info("User MQ 服务已关闭")
}
