/**
 * @projectName: CampusHub
 * @package: main
 * @className: UserRPC
 * @author: lijunqi
 * @description: 用户RPC服务入口，提供信用分服务和学生认证服务
 * @date: 2026-01-30
 * @version: 1.0
 */

package main

import (
	"flag"
	"fmt"

	"activity-platform/app/user/rpc/internal/config"
	creditserviceserver "activity-platform/app/user/rpc/internal/server/creditservice"
	verifyserviceserver "activity-platform/app/user/rpc/internal/server/verifyservice"
	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"
	"activity-platform/common/interceptor/rpcserver"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// 配置文件路径
var configFile = flag.String("f", "etc/user.yaml", "the config file")

func main() {
	flag.Parse()

	// 加载配置
	var c config.Config
	conf.MustLoad(*configFile, &c)

	// 创建服务上下文（依赖注入）
	ctx := svc.NewServiceContext(c)

	// 创建 gRPC Server
	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		// 注册 CreditService（信用分服务）
		pb.RegisterCreditServiceServer(grpcServer, creditserviceserver.NewCreditServiceServer(ctx))

		// 注册 VerifyService（学生认证服务）
		pb.RegisterVerifyServiceServer(grpcServer, verifyserviceserver.NewVerifyServiceServer(ctx))

		// 开发环境开启 gRPC Reflection（便于 grpcurl 调试）
		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	// 注册错误拦截器：将 BizError 转换为 gRPC Status
	s.AddUnaryInterceptors(rpcserver.ErrorInterceptor)

	fmt.Printf("Starting user rpc server at %s...\n", c.ListenOn)
	s.Start()
}
