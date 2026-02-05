package constants

import "time"

// Redis Key 前缀规范
// 格式: {业务}:{模块}:{具体标识}
// 示例: user:token:access:123, activity:lock:register:456

const (
	// ============ 通用缓存 Key 前缀 ============

	// CacheItemPrefix 示例资源缓存前缀
	CacheItemPrefix = "demo:item:"

	// ============ 用户服务 Redis Key ============

	// CacheUserPrefix 用户信息缓存前缀
	CacheUserPrefix = "user:info:"
	// CacheTokenPrefix Token 缓存前缀
	CacheTokenPrefix = "user:token:"
	// CacheSmsCodePrefix 短信验证码前缀
	CacheSmsCodePrefix = "user:sms:"

	// ============ 活动服务 Redis Key ============

	// CacheActivityPrefix 活动详情缓存前缀
	CacheActivityPrefix = "activity:detail:"
	// CacheActivityListPrefix 活动列表缓存前缀
	CacheActivityListPrefix = "activity:list:"
	// LockRegistrationPrefix 报名分布式锁前缀
	LockRegistrationPrefix = "activity:lock:register:"
	// CacheStockPrefix 库存缓存前缀
	CacheStockPrefix = "activity:stock:"

	// ============ 聊天服务 Redis Key ============

	// CacheUnreadPrefix 未读消息数前缀
	CacheUnreadPrefix = "chat:unread:"

	// ============ 学生认证服务 Redis Key ============

	// VerifyRateLimitPrefix 认证申请限流Key前缀
	// 格式: verify:rate_limit:{userId}
	VerifyRateLimitPrefix = "verify:rate_limit:"

	// ============ OCR服务 Redis Key ============

	// OcrCircuitBreakerPrefix OCR熔断器Key前缀
	// 格式: ocr:circuit:{provider}
	OcrCircuitBreakerPrefix = "ocr:circuit:"

	// OcrCircuitFailuresPrefix OCR失败计数Key前缀
	// 格式: ocr:circuit:{provider}:failures
	OcrCircuitFailuresPrefix = "ocr:circuit:failures:"
)

// ============ 缓存过期时间 ============

const (
	// CacheExpireDefault 默认缓存过期时间
	CacheExpireDefault = 30 * time.Minute
	// CacheExpireShort 短期缓存（热点数据）
	CacheExpireShort = 5 * time.Minute
	// CacheExpireLong 长期缓存（变化少的数据）
	CacheExpireLong = 24 * time.Hour
	// LockExpireDefault 分布式锁默认过期时间
	LockExpireDefault = 10 * time.Second
)

// ============ Key 生成辅助函数 ============
