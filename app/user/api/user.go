// ============================================================================
// 用户服务 API 入口
// ============================================================================
//
// 负责人：杨春路（B同学）
//
// 说明：
//   user-api 是用户服务的 HTTP 接口层，负责：
//   - 用户注册、登录（签发 JWT Token）
//   - 用户信息查询和更新
//
// 启动命令：
//   go run user.go -f etc/user-api.yaml
//
// 代码生成：
//   cd app/user/api
//   goctl api go -api desc/user.api -dir . -style go_zero
//
// ============================================================================

package main

import (
	"flag"
	"fmt"

	"activity-platform/common/response"

	// TODO(杨春路): goctl 生成代码后取消注释
	// "activity-platform/app/user/api/internal/config"
	// "activity-platform/app/user/api/internal/handler"
	// "activity-platform/app/user/api/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/user-api.yaml", "配置文件路径")

func main() {
	flag.Parse()

	// ============================================================================
	// 重要：设置全局错误处理器
	// ============================================================================
	// 这一步必须在 server.Start() 之前执行
	// 作用：让 goctl 生成的 handler 中的 httpx.ErrorCtx 使用统一的响应格式
	//
	// 不设置时的响应格式：{"error": "用户不存在"}
	// 设置后的响应格式：  {"code": 2001, "message": "用户不存在"}
	//
	response.SetupGlobalErrorHandler()

	// 可选：如果想让 httpx.OkJsonCtx 也使用统一格式，取消下面的注释
	// response.SetupGlobalOkHandler()
	// ============================================================================

	// TODO(杨春路): goctl 生成代码后，取消下方注释，删除临时代码
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
	// fmt.Printf("Starting user-api server at %s:%d...\n", c.Host, c.Port)
	// server.Start()

	// ============ 临时代码（goctl 生成后删除）============
	var c struct {
		rest.RestConf
	}
	conf.MustLoad(*configFile, &c)
	fmt.Printf("user-api 骨架已就绪，等待 goctl 生成代码...\n")
	fmt.Printf("请执行: cd app/user/api && goctl api go -api desc/user.api -dir . -style go_zero\n")
	// ============ 临时代码结束 ============
}
