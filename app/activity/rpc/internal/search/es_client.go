package search

import (
	"context"
	"fmt"
	"time"

	"github.com/olivere/elastic/v7"
	"github.com/zeromicro/go-zero/core/logx"
)

// ==================== ES 配置 ====================

// ESConfig ES 配置
type ESConfig struct {
	Enabled       bool     `json:",default=false"`                   // 是否启用 ES
	Hosts         []string `json:",default=[http://localhost:9200]"` // ES 地址
	Username      string   `json:",optional"`                        // 认证用户名
	Password      string   `json:",optional"`                        // 认证密码
	IndexName     string   `json:",default=activities"`              // 索引名
	MaxRetries    int      `json:",default=3"`                       // 最大重试次数
	HealthTimeout int      `json:",default=5"`                       // 健康检查超时（秒）
}

// ==================== ES 客户端 ====================

// ESClient ES 客户端封装
type ESClient struct {
	client    *elastic.Client
	indexName string
	config    ESConfig
}

// NewESClient 创建 ES 客户端
//
// 初始化流程：
// 1. 配置连接选项（地址、认证、重试策略）
// 2. 创建客户端连接
// 3. 执行健康检查（Ping）
// 4. 确保索引存在
func NewESClient(cfg ESConfig) (*ESClient, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	// 1. 配置选项
	options := []elastic.ClientOptionFunc{
		elastic.SetURL(cfg.Hosts...),
		elastic.SetSniff(false), // 单节点关闭嗅探（重要！Docker 环境必须关闭）
		elastic.SetHealthcheck(true),
		elastic.SetHealthcheckTimeout(time.Duration(cfg.HealthTimeout) * time.Second),
		// 指数退避重试策略：100ms → 200ms → 400ms → ...（最大 5s）
		elastic.SetRetrier(elastic.NewBackoffRetrier(
			elastic.NewExponentialBackoff(100*time.Millisecond, 5*time.Second),
		)),
	}

	// 2. 认证配置（如果有）
	if cfg.Username != "" && cfg.Password != "" {
		options = append(options, elastic.SetBasicAuth(cfg.Username, cfg.Password))
	}

	// 3. 创建客户端
	client, err := elastic.NewClient(options...)
	if err != nil {
		return nil, fmt.Errorf("创建 ES 客户端失败: %w", err)
	}

	// 4. 健康检查
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.HealthTimeout)*time.Second)
	defer cancel()

	info, code, err := client.Ping(cfg.Hosts[0]).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("ES 连接失败: %w", err)
	}
	logx.Infof("[ESClient] 连接成功: version=%s, status_code=%d", info.Version.Number, code)

	esClient := &ESClient{
		client:    client,
		indexName: cfg.IndexName,
		config:    cfg,
	}

	// 5. 确保索引存在
	if err := esClient.ensureIndex(ctx); err != nil {
		logx.Errorf("[ESClient] 创建索引失败: %v", err)
		// 索引创建失败不阻塞启动，后续可手动创建
	}

	return esClient, nil
}

// ensureIndex 确保索引存在
func (c *ESClient) ensureIndex(ctx context.Context) error {
	exists, err := c.client.IndexExists(c.indexName).Do(ctx)
	if err != nil {
		return fmt.Errorf("检查索引存在失败: %w", err)
	}

	if !exists {
		// 创建索引
		createIndex, err := c.client.CreateIndex(c.indexName).
			BodyString(IndexMapping).
			Do(ctx)
		if err != nil {
			return fmt.Errorf("创建索引失败: %w", err)
		}
		if !createIndex.Acknowledged {
			return fmt.Errorf("创建索引未被确认")
		}
		logx.Infof("[ESClient] 索引 %s 创建成功", c.indexName)
	} else {
		logx.Infof("[ESClient] 索引 %s 已存在", c.indexName)
	}

	return nil
}

// Client 获取原始客户端（用于高级操作）
func (c *ESClient) Client() *elastic.Client {
	return c.client
}

// IndexName 获取索引名
func (c *ESClient) IndexName() string {
	return c.indexName
}

// Close 关闭客户端
func (c *ESClient) Close() {
	if c.client != nil {
		c.client.Stop()
		logx.Info("[ESClient] 客户端已关闭")
	}
}

// HealthCheck ES 健康检查
func (c *ESClient) HealthCheck(ctx context.Context) error {
	health, err := c.client.ClusterHealth().Do(ctx)
	if err != nil {
		return fmt.Errorf("健康检查失败: %w", err)
	}

	// 检查集群状态
	if health.Status == "red" {
		return fmt.Errorf("ES 集群状态异常: %s", health.Status)
	}

	logx.Debugf("[ESClient] 健康检查通过: status=%s, nodes=%d", health.Status, health.NumberOfNodes)
	return nil
}

// ==================== 熔断器封装 ====================

// ESClientWithBreaker 带熔断器的 ES 客户端
//
// 熔断器作用：
// - 当 ES 连续失败时，自动熔断，快速返回降级响应
// - 避免大量请求堆积，导致服务雪崩
// - 一段时间后自动尝试恢复
type ESClientWithBreaker struct {
	*ESClient
	enabled bool // ES 是否启用
}

// NewESClientWithBreaker 创建带熔断器的 ES 客户端
func NewESClientWithBreaker(cfg ESConfig) (*ESClientWithBreaker, error) {
	if !cfg.Enabled {
		logx.Info("[ESClient] ES 未启用，搜索将使用 MySQL")
		return &ESClientWithBreaker{
			ESClient: nil,
			enabled:  false,
		}, nil
	}

	client, err := NewESClient(cfg)
	if err != nil {
		return nil, err
	}

	return &ESClientWithBreaker{
		ESClient: client,
		enabled:  true,
	}, nil
}

// IsEnabled 检查 ES 是否启用
func (c *ESClientWithBreaker) IsEnabled() bool {
	return c.enabled && c.ESClient != nil
}

// IsAvailable 检查 ES 是否可用（简化版，实际可接入 go-zero breaker）
//
// 当前实现：直接检查客户端是否存在
// 生产环境建议：接入 go-zero 内置熔断器
func (c *ESClientWithBreaker) IsAvailable() bool {
	if !c.IsEnabled() {
		return false
	}

	// 简单健康检查（可选：缓存检查结果，避免频繁检查）
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := c.HealthCheck(ctx)
	return err == nil
}

// SearchWithFallback 带降级的搜索
//
// 优先使用 ES 搜索，ES 不可用时返回错误让上层降级到 MySQL
func (c *ESClientWithBreaker) SearchWithFallback(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
	if !c.IsEnabled() {
		return nil, ErrESNotEnabled
	}

	return c.ESClient.Search(ctx, req)
}
