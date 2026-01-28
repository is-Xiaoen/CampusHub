// ============================================================================
// 健康检查与服务信息
// ============================================================================

package handler

import (
	"net/http"
	"runtime"
	"time"

	"activity-platform/app/gateway/api/internal/svc"
	"activity-platform/common/response"
)

var startTime = time.Now()

// HealthHandler 健康检查接口
// GET /health
// 用途：Kubernetes 探针、负载均衡健康检查
func HealthHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response.Success(w, map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now().Format(time.RFC3339),
			"uptime":    time.Since(startTime).String(),
		})
	}
}

// IndexHandler 服务信息接口
// GET /
// 用途：显示服务基本信息
func IndexHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response.Success(w, map[string]interface{}{
			"service":    "CampusHub Gateway API",
			"version":    "1.0.0",
			"go_version": runtime.Version(),
			"endpoints": map[string]string{
				"health":     "GET /health",
				"user":       "/api/v1/user/*",
				"activities": "/api/v1/activities/*",
				"chat":       "/api/v1/chat/*",
			},
		})
	}
}
