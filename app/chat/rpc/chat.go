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
	"flag"
	"fmt"

	"activity-platform/app/chat/rpc/chat"
	"activity-platform/app/chat/rpc/internal/config"
	"activity-platform/app/chat/rpc/internal/server"
	"activity-platform/app/chat/rpc/internal/svc"
	"activity-platform/common/interceptor/rpcserver"

	"github.com/zeromicro/go-zero/core/conf"
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

	fmt.Printf("Starting chat rpc server at %s...\n", c.ListenOn)
	s.Start()
}
