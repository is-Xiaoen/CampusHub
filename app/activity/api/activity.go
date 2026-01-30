// ============================================================================
// 活动服务 API 入口
// ============================================================================
//
// 负责人：马肖阳（C/D同学）
//
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
// ============================================================================

package main

import (
	"flag"
	"fmt"

	"activity-platform/common/response"

	// TODO(马肖阳): goctl 生成代码后取消注释
	// "activity-platform/app/activity/api/internal/config"
	// "activity-platform/app/activity/api/internal/handler"
	// "activity-platform/app/activity/api/internal/svc"

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

	// TODO(马肖阳): goctl 生成代码后，取消下方注释，删除临时代码
	//
	// var c config.Config
	// conf.MustLoad(*configFile, &c)
	//
	// server := rest.MustNewServer(c.RestConf)
	// defer server.Stop()
	//
	// ctx := svc.NewServiceContext(c)
	// handler.RegisterHandlers(server, ctx)
	//
	// fmt.Printf("Starting activity-api server at %s:%d...\n", c.Host, c.Port)
	// server.Start()

	// ============ 临时代码（goctl 生成后删除）============
	var c struct {
		rest.RestConf
	}
	conf.MustLoad(*configFile, &c)
	fmt.Printf("activity-api 骨架已就绪，等待 goctl 生成代码...\n")
	fmt.Printf("请执行: cd app/activity/api && goctl api go -api desc/activity.api -dir . -style go_zero\n")
	// ============ 临时代码结束 ============
}
