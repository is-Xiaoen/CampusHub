// ============================================================================
// 用户服务 API 入口
// ============================================================================
//
// 负责人：杨春路（B同学）
//
// 说明：
//   user-api 是用户服务的 HTTP 接口层，负责：
//   - 信用分查询
//   - 学生认证管理
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

	"activity-platform/app/user/api/internal/config"
	"activity-platform/app/user/api/internal/handler"
	"activity-platform/app/user/api/internal/svc"
	"activity-platform/common/response"

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

	// 加载配置
	var c config.Config
	conf.MustLoad(*configFile, &c)

	// 创建 HTTP 服务器
	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	// 创建服务上下文（初始化 RPC 客户端等依赖）
	ctx := svc.NewServiceContext(c)

	// 注册路由
	handler.RegisterHandlers(server, ctx)

	fmt.Printf("Starting user-api server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
