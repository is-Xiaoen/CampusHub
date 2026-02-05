package dtm

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dtm-labs/client/dtmcli"
	"github.com/dtm-labs/client/dtmgrpc"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/protobuf/proto"
)

// Client DTM 客户端封装
// 提供分布式事务的发起、健康检查、优雅关闭等功能
//
// 并发安全设计：
// - healthy 使用 atomic.Bool 保证原子读写
// - stopChan 用于通知 healthCheck goroutine 退出
// - wg 用于等待 goroutine 完全退出
type Client struct {
	server     string        // DTM Server gRPC 地址（如 localhost:36790）
	httpServer string        // DTM Server HTTP 地址（如 localhost:36789）
	timeout    time.Duration // 事务超时时间
	httpClient *http.Client  // HTTP 客户端（健康检查用）
	healthy    atomic.Bool   // 健康状态（原子操作）
	stopChan   chan struct{} // 停止信号
	wg         sync.WaitGroup
}

// Config DTM 客户端配置
type Config struct {
	Enabled        bool          // 是否启用 DTM
	Server         string        // DTM gRPC 地址
	HTTPServer     string        // DTM HTTP 地址
	Timeout        time.Duration // 事务超时时间
	ActivityRpcURL string        // Activity 服务 gRPC 地址
	UserRpcURL     string        // User 服务 gRPC 地址
}

// NewClient 创建 DTM 客户端
//
// 初始化流程：
// 1. 设置默认配置
// 2. 启动健康检查 goroutine
// 3. 返回客户端实例
func NewClient(cfg Config) *Client {
	// 设置默认值
	if cfg.Timeout == 0 {
		cfg.Timeout = 120 * time.Second
	}

	client := &Client{
		server:     cfg.Server,
		httpServer: cfg.HTTPServer,
		timeout:    cfg.Timeout,
		httpClient: &http.Client{Timeout: 3 * time.Second},
		stopChan:   make(chan struct{}),
	}

	// 初始状态设为健康（乐观策略）
	client.healthy.Store(true)

	// 启动健康检查 goroutine
	client.wg.Add(1)
	go client.healthCheckLoop()

	logx.Infof("[DTM] 客户端初始化完成: server=%s, httpServer=%s, timeout=%v",
		cfg.Server, cfg.HTTPServer, cfg.Timeout)

	return client
}

// IsHealthy 检查 DTM Server 是否健康
// 使用 atomic.Bool 保证并发安全
func (c *Client) IsHealthy() bool {
	return c.healthy.Load()
}

// GetServer 获取 DTM Server gRPC 地址
func (c *Client) GetServer() string {
	return c.server
}

// GetTimeout 获取事务超时时间
func (c *Client) GetTimeout() time.Duration {
	return c.timeout
}

// Close 关闭客户端
// 1. 发送停止信号
// 2. 等待 healthCheck goroutine 退出
func (c *Client) Close() {
	close(c.stopChan)
	c.wg.Wait()
	logx.Info("[DTM] 客户端已关闭")
}

// healthCheckLoop 健康检查循环
// 每 10 秒检查一次 DTM Server 状态
func (c *Client) healthCheckLoop() {
	defer c.wg.Done()

	// 启动时立即检查一次
	c.doHealthCheck()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopChan:
			logx.Info("[DTM] 健康检查已停止")
			return
		case <-ticker.C:
			c.doHealthCheck()
		}
	}
}

// doHealthCheck 执行健康检查
func (c *Client) doHealthCheck() {
	newHealthy := c.checkHealth()
	oldHealthy := c.healthy.Load()

	// 状态变化时记录日志
	if oldHealthy != newHealthy {
		if newHealthy {
			logx.Infof("[DTM] 服务恢复健康: %s", c.httpServer)
		} else {
			logx.Errorf("[DTM] 服务不健康: %s", c.httpServer)
		}
	}

	c.healthy.Store(newHealthy)
}

// checkHealth 检查 DTM Server 健康状态
func (c *Client) checkHealth() bool {
	url := fmt.Sprintf("http://%s/api/ping", c.httpServer)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// 读取并丢弃 body，避免连接泄漏
	_, _ = io.Copy(io.Discard, resp.Body)

	return resp.StatusCode == http.StatusOK
}

// ============================================================
// SAGA 事务相关方法
// ============================================================

// CreateActivitySagaReq 创建活动 SAGA 请求参数
type CreateActivitySagaReq struct {
	// Activity 服务参数
	ActivityRpcURL string        // Activity RPC 地址（如 localhost:9002）
	ActivityReq    proto.Message // CreateActivityActionReq

	// User 服务参数（可选，无标签时为空）
	UserRpcURL string        // User RPC 地址（如 localhost:9001）
	TagReq     proto.Message // TagUsageCountReq（可为 nil）
	HasTags    bool          // 是否有标签
}

// CreateActivitySaga 创建活动 SAGA 事务
//
// SAGA 流程：
//
//	Branch 1: CreateActivityAction -> CreateActivityCompensate
//	Branch 2: IncrTagUsageCount -> DecrTagUsageCount (仅当有标签时)
//
// 返回值：
//   - gid: 全局事务 ID（用于追踪）
//   - error: 错误信息
func (c *Client) CreateActivitySaga(ctx context.Context, req CreateActivitySagaReq) (string, error) {
	// 生成全局事务 ID
	gid := dtmcli.MustGenGid(c.server)

	// 构建 Activity 服务的分支 URL
	activityActionURL := fmt.Sprintf("%s/activity.ActivityBranchService/CreateActivityAction", req.ActivityRpcURL)
	activityCompensateURL := fmt.Sprintf("%s/activity.ActivityBranchService/CreateActivityCompensate", req.ActivityRpcURL)

	// 创建 SAGA 事务
	saga := dtmgrpc.NewSagaGrpc(c.server, gid).
		Add(activityActionURL, activityCompensateURL, req.ActivityReq)

	// 如果有标签，添加标签计数分支
	if req.HasTags && req.TagReq != nil {
		userIncrURL := fmt.Sprintf("%s/user.TagBranchService/IncrTagUsageCount", req.UserRpcURL)
		userDecrURL := fmt.Sprintf("%s/user.TagBranchService/DecrTagUsageCount", req.UserRpcURL)
		saga = saga.Add(userIncrURL, userDecrURL, req.TagReq)
	}

	// 设置事务选项
	saga.WaitResult = true                          // 等待事务完成
	saga.TimeoutToFail = int64(c.timeout.Seconds()) // 超时后失败
	saga.RetryInterval = 10                         // 重试间隔 10 秒
	saga.BranchHeaders = map[string]string{         // 传递追踪信息
		"x-trace-id": getTraceID(ctx),
	}

	// 提交事务
	logx.Infof("[DTM] 开始创建活动 SAGA 事务: gid=%s", gid)
	if err := saga.Submit(); err != nil {
		logx.Errorf("[DTM] SAGA 事务提交失败: gid=%s, err=%v", gid, err)
		return "", fmt.Errorf("DTM 事务提交失败: %w", err)
	}

	logx.Infof("[DTM] SAGA 事务完成: gid=%s", gid)
	return gid, nil
}

// DeleteActivitySagaReq 删除活动 SAGA 请求参数
type DeleteActivitySagaReq struct {
	ActivityRpcURL string        // Activity RPC 地址
	ActivityReq    proto.Message // DeleteActivityActionReq
	UserRpcURL     string        // User RPC 地址
	TagReq         proto.Message // TagUsageCountReq（用于减少计数）
	HasTags        bool          // 是否有标签
}

// DeleteActivitySaga 删除活动 SAGA 事务
//
// SAGA 流程：
//
//	Branch 1: DeleteActivityAction -> DeleteActivityCompensate
//	Branch 2: DecrTagUsageCount -> IncrTagUsageCount (仅当有标签时)
func (c *Client) DeleteActivitySaga(ctx context.Context, req DeleteActivitySagaReq) (string, error) {
	gid := dtmcli.MustGenGid(c.server)

	activityActionURL := fmt.Sprintf("%s/activity.ActivityBranchService/DeleteActivityAction", req.ActivityRpcURL)
	activityCompensateURL := fmt.Sprintf("%s/activity.ActivityBranchService/DeleteActivityCompensate", req.ActivityRpcURL)

	saga := dtmgrpc.NewSagaGrpc(c.server, gid).
		Add(activityActionURL, activityCompensateURL, req.ActivityReq)

	if req.HasTags && req.TagReq != nil {
		// 删除时：正向是减少计数，补偿是增加计数
		userDecrURL := fmt.Sprintf("%s/user.TagBranchService/DecrTagUsageCount", req.UserRpcURL)
		userIncrURL := fmt.Sprintf("%s/user.TagBranchService/IncrTagUsageCount", req.UserRpcURL)
		saga = saga.Add(userDecrURL, userIncrURL, req.TagReq)
	}

	saga.WaitResult = true
	saga.TimeoutToFail = int64(c.timeout.Seconds())
	saga.RetryInterval = 10
	saga.BranchHeaders = map[string]string{
		"x-trace-id": getTraceID(ctx),
	}

	logx.Infof("[DTM] 开始删除活动 SAGA 事务: gid=%s", gid)
	if err := saga.Submit(); err != nil {
		logx.Errorf("[DTM] SAGA 事务提交失败: gid=%s, err=%v", gid, err)
		return "", fmt.Errorf("DTM 事务提交失败: %w", err)
	}

	logx.Infof("[DTM] SAGA 事务完成: gid=%s", gid)
	return gid, nil
}

// getTraceID 从 context 获取追踪 ID
func getTraceID(ctx context.Context) string {
	if traceID, ok := ctx.Value("trace_id").(string); ok {
		return traceID
	}
	return ""
}
