/**
 * @projectName: CampusHub
 * @package: main
 * @className: ChatRPC
 * @author: E同学
 * @description: 聊天RPC服务入口，提供群聊管理、消息存储和系统通知服务
 * @date: 2026-02-05
 * @version: 1.0
 */

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"activity-platform/app/chat/mq/consumer"
	"activity-platform/app/chat/rpc/chat"
	"activity-platform/app/chat/rpc/internal/config"
	"activity-platform/app/chat/rpc/internal/server"
	"activity-platform/app/chat/rpc/internal/svc"
	"activity-platform/common/interceptor/rpcserver"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// 配置文件路径
var configFile = flag.String("f", "etc/chat.yaml", "the config file")

func main() {
	flag.Parse()

	// 加载配置
	var c config.Config
	conf.MustLoad(*configFile, &c)

	// 创建服务上下文（依赖注入）
	ctx := svc.NewServiceContext(c)
	defer ctx.MsgClient.Close()

	// 创建 gRPC Server
	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		// 注册 ChatService（聊天服务）
		chat.RegisterChatServiceServer(grpcServer, server.NewChatServiceServer(ctx))

		// 开发环境开启 gRPC Reflection（便于 grpcurl 调试）
		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	// 注册错误拦截器：将 BizError 转换为 gRPC Status
	s.AddUnaryInterceptors(rpcserver.ErrorInterceptor)

	// 异步启动 MQ 消费者
	startMQConsumer(ctx)

	// 监听系统信号，优雅关闭
	go handleShutdown(s, ctx)

	fmt.Printf("Starting chat rpc server at %s...\n", c.ListenOn)
	s.Start()
}

// startMQConsumer 启动 MQ 消费者（异步）
func startMQConsumer(svcCtx *svc.ServiceContext) {
	// 创建本地 Chat RPC Server 实例（用于 MQ 消费者调用）
	chatRpcServer := server.NewChatServiceServer(svcCtx)

	// 注册消费者
	registerConsumers(svcCtx, chatRpcServer)

	// 在 goroutine 中启动消息路由
	go func() {
		mqCtx := context.Background()
		logx.Info("MQ 消费者服务启动中...")

		if err := svcCtx.MsgClient.Run(mqCtx); err != nil {
			logx.Errorf("MQ 消息路由停止: %v", err)
		}
	}()

	// 等待 Router 启动
	go func() {
		<-svcCtx.MsgClient.Running()
		logx.Info("MQ 消费者服务已启动")
	}()
}

// registerConsumers 注册所有消费者
func registerConsumers(svcCtx *svc.ServiceContext, chatRpcServer chat.ChatServiceServer) {
	// 将 Server 转换为 Client 接口（通过类型适配器）
	chatRpcClient := &localChatServiceClient{server: chatRpcServer}

	// 1. 活动创建事件消费者
	activityCreatedConsumer := consumer.NewActivityCreatedConsumer(chatRpcClient)
	activityCreatedConsumer.Subscribe(svcCtx.MsgClient)

	// 2. 用户报名成功事件消费者
	memberJoinedConsumer := consumer.NewActivityMemberJoinedConsumer(chatRpcClient)
	memberJoinedConsumer.Subscribe(svcCtx.MsgClient)

	// 3. 用户取消报名事件消费者
	memberLeftConsumer := consumer.NewActivityMemberLeftConsumer(chatRpcClient)
	memberLeftConsumer.Subscribe(svcCtx.MsgClient)

	logx.Info("✅ 已注册 3 个 MQ 消费者:")
	logx.Info("  - activity.created -> chat-auto-create-group")
	logx.Info("  - activity.member.joined -> chat-auto-add-member")
	logx.Info("  - activity.member.left -> chat-auto-remove-member")
}

// localChatServiceClient 本地 RPC 调用适配器（避免网络调用）
type localChatServiceClient struct {
	server chat.ChatServiceServer
}

func (c *localChatServiceClient) CreateGroup(ctx context.Context, req *chat.CreateGroupReq, opts ...grpc.CallOption) (*chat.CreateGroupResp, error) {
	return c.server.CreateGroup(ctx, req)
}

func (c *localChatServiceClient) AddGroupMember(ctx context.Context, req *chat.AddGroupMemberReq, opts ...grpc.CallOption) (*chat.AddGroupMemberResp, error) {
	return c.server.AddGroupMember(ctx, req)
}

func (c *localChatServiceClient) RemoveGroupMember(ctx context.Context, req *chat.RemoveGroupMemberReq, opts ...grpc.CallOption) (*chat.RemoveGroupMemberResp, error) {
	return c.server.RemoveGroupMember(ctx, req)
}

func (c *localChatServiceClient) DisbandGroup(ctx context.Context, req *chat.DisbandGroupReq, opts ...grpc.CallOption) (*chat.DisbandGroupResp, error) {
	return c.server.DisbandGroup(ctx, req)
}

func (c *localChatServiceClient) GetGroupInfo(ctx context.Context, req *chat.GetGroupInfoReq, opts ...grpc.CallOption) (*chat.GetGroupInfoResp, error) {
	return c.server.GetGroupInfo(ctx, req)
}

func (c *localChatServiceClient) GetGroupMembers(ctx context.Context, req *chat.GetGroupMembersReq, opts ...grpc.CallOption) (*chat.GetGroupMembersResp, error) {
	return c.server.GetGroupMembers(ctx, req)
}

func (c *localChatServiceClient) GetUserGroups(ctx context.Context, req *chat.GetUserGroupsReq, opts ...grpc.CallOption) (*chat.GetUserGroupsResp, error) {
	return c.server.GetUserGroups(ctx, req)
}

func (c *localChatServiceClient) GetGroupByActivityId(ctx context.Context, req *chat.GetGroupByActivityIdReq, opts ...grpc.CallOption) (*chat.GetGroupByActivityIdResp, error) {
	return c.server.GetGroupByActivityId(ctx, req)
}

func (c *localChatServiceClient) SaveMessage(ctx context.Context, req *chat.SaveMessageReq, opts ...grpc.CallOption) (*chat.SaveMessageResp, error) {
	return c.server.SaveMessage(ctx, req)
}

func (c *localChatServiceClient) GetMessageHistory(ctx context.Context, req *chat.GetMessageHistoryReq, opts ...grpc.CallOption) (*chat.GetMessageHistoryResp, error) {
	return c.server.GetMessageHistory(ctx, req)
}

func (c *localChatServiceClient) GetOfflineMessages(ctx context.Context, req *chat.GetOfflineMessagesReq, opts ...grpc.CallOption) (*chat.GetOfflineMessagesResp, error) {
	return c.server.GetOfflineMessages(ctx, req)
}

func (c *localChatServiceClient) CreateNotification(ctx context.Context, req *chat.CreateNotificationReq, opts ...grpc.CallOption) (*chat.CreateNotificationResp, error) {
	return c.server.CreateNotification(ctx, req)
}

func (c *localChatServiceClient) GetNotifications(ctx context.Context, req *chat.GetNotificationsReq, opts ...grpc.CallOption) (*chat.GetNotificationsResp, error) {
	return c.server.GetNotifications(ctx, req)
}

func (c *localChatServiceClient) MarkNotificationRead(ctx context.Context, req *chat.MarkNotificationReadReq, opts ...grpc.CallOption) (*chat.MarkNotificationReadResp, error) {
	return c.server.MarkNotificationRead(ctx, req)
}

func (c *localChatServiceClient) GetUnreadCount(ctx context.Context, req *chat.GetUnreadCountReq, opts ...grpc.CallOption) (*chat.GetUnreadCountResp, error) {
	return c.server.GetUnreadCount(ctx, req)
}

func (c *localChatServiceClient) MarkAllRead(ctx context.Context, req *chat.MarkAllReadReq, opts ...grpc.CallOption) (*chat.MarkAllReadResp, error) {
	return c.server.MarkAllRead(ctx, req)
}

// handleShutdown 处理优雅关闭
func handleShutdown(s *zrpc.RpcServer, svcCtx *svc.ServiceContext) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	logx.Info("收到关闭信号，正在优雅关闭...")

	// 停止 RPC 服务
	s.Stop()

	// 关闭消息客户端
	if err := svcCtx.MsgClient.Close(); err != nil {
		logx.Errorf("关闭消息客户端失败: %v", err)
	}

	logx.Info("Chat RPC 和 MQ 服务已关闭")
	os.Exit(0)
}
