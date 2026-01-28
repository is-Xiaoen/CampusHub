package middleware

import (
	"net/http"
	"sync"
	"time"

	"activity-platform/common/response"
)

// RateLimiter 令牌桶限流器
// 面试亮点：令牌桶 vs 漏桶算法
// 令牌桶：允许突发流量，适合大多数场景
// 漏桶：严格匀速，适合需要匀速处理的场景
type RateLimiter struct {
	rate       float64    // 每秒生成令牌数
	burst      int        // 桶容量（最大令牌数）
	tokens     float64    // 当前令牌数
	lastUpdate time.Time  // 上次更新时间
	mu         sync.Mutex
}

// NewRateLimiter 创建限流器
// rate: 每秒允许的请求数
// burst: 突发容量
func NewRateLimiter(rate float64, burst int) *RateLimiter {
	return &RateLimiter{
		rate:       rate,
		burst:      burst,
		tokens:     float64(burst),
		lastUpdate: time.Now(),
	}
}

// Allow 判断是否允许请求
func (r *RateLimiter) Allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(r.lastUpdate).Seconds()
	r.lastUpdate = now

	// 添加新令牌
	r.tokens += elapsed * r.rate
	if r.tokens > float64(r.burst) {
		r.tokens = float64(r.burst)
	}

	// 消耗令牌
	if r.tokens >= 1 {
		r.tokens--
		return true
	}
	return false
}

// IPRateLimiter 基于IP的限流器
type IPRateLimiter struct {
	limiters map[string]*RateLimiter
	mu       sync.RWMutex
	rate     float64
	burst    int
}

// NewIPRateLimiter 创建IP限流器
func NewIPRateLimiter(rate float64, burst int) *IPRateLimiter {
	return &IPRateLimiter{
		limiters: make(map[string]*RateLimiter),
		rate:     rate,
		burst:    burst,
	}
}

// GetLimiter 获取指定IP的限流器
func (i *IPRateLimiter) GetLimiter(ip string) *RateLimiter {
	i.mu.RLock()
	limiter, exists := i.limiters[ip]
	i.mu.RUnlock()

	if exists {
		return limiter
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	// 双重检查
	if limiter, exists = i.limiters[ip]; exists {
		return limiter
	}

	limiter = NewRateLimiter(i.rate, i.burst)
	i.limiters[ip] = limiter
	return limiter
}

// RateLimitMiddleware 限流中间件
type RateLimitMiddleware struct {
	ipLimiter     *IPRateLimiter
	globalLimiter *RateLimiter
}

// NewRateLimitMiddleware 创建限流中间件
// globalRate: 全局限流（每秒请求数）
// globalBurst: 全局突发容量
// ipRate: 单IP限流（每秒请求数）
// ipBurst: 单IP突发容量
func NewRateLimitMiddleware(globalRate float64, globalBurst int, ipRate float64, ipBurst int) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		ipLimiter:     NewIPRateLimiter(ipRate, ipBurst),
		globalLimiter: NewRateLimiter(globalRate, globalBurst),
	}
}

// Handle 中间件处理函数
func (m *RateLimitMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 全局限流
		if !m.globalLimiter.Allow() {
			response.Error(w, 429, "服务繁忙，请稍后重试")
			return
		}

		// IP限流
		ip := getClientIP(r)
		if !m.ipLimiter.GetLimiter(ip).Allow() {
			response.Error(w, 429, "请求过于频繁，请稍后重试")
			return
		}

		next(w, r)
	}
}

// getClientIP 获取客户端IP
func getClientIP(r *http.Request) string {
	// 优先从X-Forwarded-For获取
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		return xff
	}
	// 其次从X-Real-IP获取
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}
	// 最后从RemoteAddr获取
	return r.RemoteAddr
}
