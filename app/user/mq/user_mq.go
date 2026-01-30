/**
 * @projectName: CampusHub
 * @package: main
 * @className: UserMQ
 * @author: lijunqi
 * @description: 用户MQ消费者服务入口（基于Redis Stream）
 * @date: 2026-01-30
 * @version: 1.0
 */

package main

import (
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

	// 创建服务上下文
	svcCtx := svc.NewServiceContext(c)

	// 创建消息处理器
	handlers := handler.NewHandlers(svcCtx)

	// ================================================================
	// [待实现] 启动 Redis Stream 消费者
	// ================================================================
	//
	// 需要队友提供 Redis Stream 消费者的实现，接口大致如下：
	//
	// consumer := redisstream.NewConsumer(redisstream.Config{
	//     Redis:    svcCtx.Redis,
	//     Stream:   c.Stream.Key,
	//     Group:    c.Stream.Group,
	//     Consumer: c.Stream.Consumer,
	// })
	//
	// consumer.OnMessage(func(ctx context.Context, msg redisstream.Message) error {
	//     return handlers.Handle(ctx, &handler.Message{
	//         ID:   msg.ID,
	//         Type: msg.Values["type"].(string),
	//         Data: msg.Values["data"].(string),
	//     })
	// })
	//
	// if err := consumer.Start(); err != nil {
	//     logx.Errorf("启动消费者失败: %v", err)
	//     panic(err)
	// }
	// defer consumer.Stop()
	//
	// ================================================================

	logx.Infof("User MQ 服务启动成功")
	logx.Infof("Stream配置: key=%s, group=%s, consumer=%s",
		c.Stream.Key, c.Stream.Group, c.Stream.Consumer)

	// 临时：打印处理器信息
	_ = handlers

	// 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logx.Info("User MQ 服务已关闭")
}
