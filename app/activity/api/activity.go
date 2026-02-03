package main

import (
	"flag"
	"fmt"

	"activity-platform/app/activity/api/internal/config"
	"activity-platform/app/activity/api/internal/handler"
	"activity-platform/app/activity/api/internal/svc"
	"activity-platform/common/response"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/activity-api.yaml", "配置文件路径")

func main() {
	flag.Parse()

	// ============================================================================
	// 重要：设置全局错误处理器（必须在 server.Start() 之前）
	// ============================================================================
	response.SetupGlobalErrorHandler()
	// response.SetupGlobalOkHandler() // 可选：统一成功响应格式
	// ============================================================================

	// 1. 加载配置文件
	var c config.Config
	conf.MustLoad(*configFile, &c)

	// 2. 创建 REST 服务器
	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	// 3. 初始化服务上下文
	ctx := svc.NewServiceContext(c)

	// 4. 注册路由处理器
	handler.RegisterHandlers(server, ctx)

	// 5. 启动服务
	fmt.Printf("Starting activity-api server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}

// 活动服务 API 入口
// 说明：
//   activity-api 是活动服务的 HTTP 接口层，负责：
//   - 活动 CRUD
//   - 报名、签到
//   - 我的活动列表
//
// 启动命令：
//   go run activity.go -f etc/activity-api.yaml
//
// 代码生成：
//   cd app/activity/api
//   goctl api go -api desc/activity.api -dir . -style go_zero
//
