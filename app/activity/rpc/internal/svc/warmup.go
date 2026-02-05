package svc

import (
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// WarmupCache 启动时预热缓存
//
// 预热策略：
//   - 分类列表：几乎不变，启动时加载
//   - 热门活动 Top10：首页热点，启动时加载
//   - 活动详情：数据量大，按需加载（不预热）
//
// 注意事项：
//   - 预热在协程中执行，不阻塞服务启动
//   - 设置超时时间，避免预热过慢影响服务健康检查
//   - 预热失败仅记录日志，不影响服务正常运行
func (s *ServiceContext) WarmupCache() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logx.Info("[CacheWarmup] 开始预热缓存...")

	// 1. 预热分类列表缓存
	if s.CategoryCache != nil {
		if err := s.CategoryCache.Warmup(ctx); err != nil {
			logx.Errorf("[CacheWarmup] 预热分类缓存失败: %v", err)
		} else {
			logx.Info("[CacheWarmup] 预热分类缓存成功")
		}
	}

	// 2. 预热热门活动缓存
	if s.HotCache != nil {
		if err := s.HotCache.Warmup(ctx); err != nil {
			logx.Errorf("[CacheWarmup] 预热热门活动缓存失败: %v", err)
		} else {
			logx.Info("[CacheWarmup] 预热热门活动缓存成功")
		}
	}

	logx.Info("[CacheWarmup] 缓存预热完成")
}

// WarmupCacheAsync 异步预热缓存（不阻塞主流程）
func (s *ServiceContext) WarmupCacheAsync() {
	go s.WarmupCache()
}
