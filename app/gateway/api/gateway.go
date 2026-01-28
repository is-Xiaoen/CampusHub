// ============================================================================
// Gateway API 网关服务入口
// ============================================================================
//
// 项目：CampusHub - 校园活动平台
// 模块：API 网关（BFF 层）
//
// 功能说明：
//   - HTTP 路由注册与请求分发
//   - 中间件加载（鉴权、限流、CORS、请求追踪）
//   - RPC 客户端初始化与服务发现
//   - 统一错误处理与响应格式化
//   - 优雅关闭
//
// 架构位置：
//   客户端 -> Gateway API (8080) -> RPC Services (9001-9003)
//
// 启动命令：
//   go run gateway.go -f etc/gateway.yaml
//
// ============================================================================

package main

import (
	"flag"
	"fmt"
	"net/http"

	"activity-platform/app/gateway/api/internal/config"
	"activity-platform/app/gateway/api/internal/handler"
	"activity-platform/app/gateway/api/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/gateway.yaml", "配置文件路径")

func main() {
	flag.Parse()

	// ==================== 1. 加载配置 ====================
	var c config.Config
	conf.MustLoad(*configFile, &c)

	// ==================== 2. 创建 REST 服务器 ====================
	server := rest.MustNewServer(c.RestConf, rest.WithNotFoundHandler(notFoundHandler()))

	defer server.Stop()

	// ==================== 3. 初始化服务上下文 ====================
	ctx := svc.NewServiceContext(c)

	// ==================== 4. 注册路由和中间件 ====================
	handler.RegisterHandlers(server, ctx)

	// ==================== 5. 启动服务 ====================
	fmt.Printf("Starting Gateway API server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}

// notFoundHandler 404 处理
func notFoundHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"code":1004,"message":"接口不存在"}`))
	}
}
