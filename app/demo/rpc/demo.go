// ============================================================================
// Demo RPC 服务入口
// ============================================================================
//
// 文件说明：
//   这是服务的主入口文件，负责：
//   - 加载配置
//   - 初始化服务上下文
//   - 启动 gRPC 服务器
//   - 注册到 Etcd
//
// 启动命令：
//   go run demo.go                           # 使用默认配置
//   go run demo.go -f etc/demo.yaml          # 指定配置文件
//   go run demo.go -f etc/demo-prod.yaml     # 生产环境配置
//
// ============================================================================

package main

import (
	"flag"
	"fmt"

	"activity-platform/app/demo/rpc/internal/config"
	"activity-platform/app/demo/rpc/internal/server"
	"activity-platform/app/demo/rpc/internal/svc"
	"activity-platform/app/demo/rpc/pb"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// configFile 配置文件路径
// 可通过 -f 参数指定，默认为 etc/demo.yaml
var configFile = flag.String("f", "etc/demo.yaml", "配置文件路径")

func main() {
	flag.Parse()

	// ==================== 1. 加载配置 ====================
	var c config.Config
	conf.MustLoad(*configFile, &c)

	// ==================== 2. 初始化服务上下文 ====================
	// ServiceContext 负责初始化数据库、缓存、Model 等
	ctx := svc.NewServiceContext(c)

	// ==================== 3. 创建 gRPC 服务器 ====================
	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		// 注册服务实现
		pb.RegisterDemoServiceServer(grpcServer, server.NewDemoServiceServer(ctx))

		// 开发环境启用反射（便于使用 grpcurl 测试）
		// 生产环境建议关闭
		if c.Mode == service.DevMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	// ==================== 4. 启动服务 ====================
	fmt.Printf("Starting Demo RPC server at %s...\n", c.ListenOn)
	s.Start()
}
