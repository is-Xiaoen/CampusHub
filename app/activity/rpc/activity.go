package main

import (
	"flag"
	"fmt"

	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/activity/rpc/internal/config"
	"activity-platform/app/activity/rpc/internal/cron"
	"activity-platform/app/activity/rpc/internal/server"
	activitybranchserver "activity-platform/app/activity/rpc/internal/server/activitybranchservice"
	"activity-platform/app/activity/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// ci: verify build pipeline
var configFile = flag.String("f", "etc/activity.yaml", "配置文件路径")

func main() {
	flag.Parse()

	// 1. 加载配置
	var c config.Config
	conf.MustLoad(*configFile, &c)

	// 2. 初始化 ServiceContext
	ctx := svc.NewServiceContext(c)

	// 3. 缓存预热（异步执行，不阻塞启动）
	ctx.WarmupCacheAsync()

	// 4. 启动状态自动流转定时任务
	statusCron := cron.NewStatusCron(
		ctx.Redis,
		ctx.DB,
		ctx.ActivityModel,
		ctx.StatusLogModel,
		ctx.MsgProducer,
	)
	statusCron.Start()
	defer statusCron.Stop()

	// 5. DTM 客户端关闭（如果启用）
	if ctx.DTMClient != nil {
		defer ctx.DTMClient.Close()
	}

	// 5.5 消息发布器关闭（如果启用）
	if ctx.MsgProducer != nil {
		defer ctx.MsgProducer.Close()
	}

	// 6. 创建 RPC 服务
	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		// 注册 ActivityService（外部接口）- 使用根目录 logic 的实现
		activity.RegisterActivityServiceServer(grpcServer, server.NewActivityServiceServer(ctx))

		// 注册 ActivityBranchService（DTM 分支操作接口）
		activity.RegisterActivityBranchServiceServer(grpcServer, activitybranchserver.NewActivityBranchServiceServer(ctx))

		// 开发环境启用 gRPC 反射（便于调试）
		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	fmt.Printf("Starting activity rpc server at %s...\n", c.ListenOn)
	logx.Infof("活动服务 RPC 启动: %s", c.ListenOn)
	s.Start()
}

// 活动服务 RPC 入口
// 说明：
//   activity-rpc 是活动服务的 gRPC 服务层，负责：
//   - 活动 CRUD + 状态机
//   - 活动列表/详情/搜索
//   - 分类标签管理
//   - 跨服务调用接口（供 User/Chat 服务调用）
//
// 启动命令：
//   go run activity.go -f etc/activity.yaml
//
// 代码生成：
//   cd app/activity/rpc
//   goctl rpc protoc activity.proto --go_out=. --go-grpc_out=. --zrpc_out=. --style go_zero
