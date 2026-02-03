package syncer

import (
	"context"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

// ==================== 分布式锁 ====================
//
// 用途：防止多实例重复执行对账任务
//
// 实现原理：
//   - 基于 Redis SETNX + TTL
//   - 自动续期防止任务超时被释放
//   - 支持优雅释放
//
// 企业级设计：
//   - 锁超时自动释放，防止死锁
//   - 看门狗机制，任务执行中自动续期
//   - 只有持有者才能释放锁（通过 token 验证）

// DistributedLock 分布式锁接口
type DistributedLock interface {
	// TryLock 尝试获取锁，返回是否成功
	TryLock(ctx context.Context) (bool, error)
	// Unlock 释放锁
	Unlock(ctx context.Context) error
	// Refresh 刷新锁的过期时间（看门狗调用）
	Refresh(ctx context.Context) error
}

// RedisLock Redis 分布式锁实现
type RedisLock struct {
	rds       *redis.Redis
	key       string        // 锁的 key
	token     string        // 锁的 token（用于安全释放）
	ttl       time.Duration // 锁的过期时间
	stopRenew chan struct{} // 停止续期信号
	renewDone chan struct{} // 续期协程退出信号
}

// NewRedisLock 创建 Redis 分布式锁
//
// 参数：
//   - rds: Redis 客户端
//   - key: 锁的 key（建议格式：lock:service:task）
//   - ttl: 锁的过期时间（建议 30 秒，配合看门狗续期）
func NewRedisLock(rds *redis.Redis, key string, ttl time.Duration) *RedisLock {
	return &RedisLock{
		rds:   rds,
		key:   key,
		token: fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().UnixNano()%1000),
		ttl:   ttl,
	}
}

// TryLock 尝试获取锁
//
// 使用 SETNX + TTL 原子操作
// 成功返回 true，失败返回 false（锁已被其他实例持有）
func (l *RedisLock) TryLock(ctx context.Context) (bool, error) {
	// SET key value NX PX milliseconds
	ok, err := l.rds.SetnxExCtx(ctx, l.key, l.token, int(l.ttl.Seconds()))
	if err != nil {
		return false, fmt.Errorf("获取分布式锁失败: %w", err)
	}

	if ok {
		// 启动看门狗（自动续期）
		l.startWatchdog(ctx)
		logx.Infof("[RedisLock] 获取锁成功: key=%s, ttl=%v", l.key, l.ttl)
	}

	return ok, nil
}

// Unlock 释放锁
//
// 只有锁的持有者（token 匹配）才能释放
// 使用 Lua 脚本保证原子性
func (l *RedisLock) Unlock(ctx context.Context) error {
	// 停止看门狗
	l.stopWatchdog()

	// Lua 脚本：只有 token 匹配才删除
	script := `
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("DEL", KEYS[1])
		else
			return 0
		end
	`

	result, err := l.rds.EvalCtx(ctx, script, []string{l.key}, l.token)
	if err != nil {
		return fmt.Errorf("释放分布式锁失败: %w", err)
	}

	if result.(int64) == 1 {
		logx.Infof("[RedisLock] 释放锁成功: key=%s", l.key)
	} else {
		logx.Infof("[RedisLock] 锁已过期或被其他实例持有: key=%s", l.key)
	}

	return nil
}

// Refresh 刷新锁的过期时间
//
// 看门狗调用，延长锁的持有时间
func (l *RedisLock) Refresh(ctx context.Context) error {
	// Lua 脚本：只有 token 匹配才续期
	script := `
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("PEXPIRE", KEYS[1], ARGV[2])
		else
			return 0
		end
	`

	result, err := l.rds.EvalCtx(ctx, script, []string{l.key}, l.token, int(l.ttl.Milliseconds()))
	if err != nil {
		return fmt.Errorf("续期分布式锁失败: %w", err)
	}

	if result.(int64) == 0 {
		return fmt.Errorf("锁已过期或被其他实例持有")
	}

	return nil
}

// startWatchdog 启动看门狗（自动续期）
//
// 每隔 TTL/3 时间续期一次，确保任务执行期间锁不会过期
func (l *RedisLock) startWatchdog(ctx context.Context) {
	l.stopRenew = make(chan struct{})
	l.renewDone = make(chan struct{})

	renewInterval := l.ttl / 3
	if renewInterval < time.Second {
		renewInterval = time.Second
	}

	go func() {
		defer close(l.renewDone)

		ticker := time.NewTicker(renewInterval)
		defer ticker.Stop()

		for {
			select {
			case <-l.stopRenew:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := l.Refresh(ctx); err != nil {
					logx.Errorf("[RedisLock] 续期失败: key=%s, err=%v", l.key, err)
					return
				}
				logx.Debugf("[RedisLock] 续期成功: key=%s", l.key)
			}
		}
	}()
}

// stopWatchdog 停止看门狗
func (l *RedisLock) stopWatchdog() {
	if l.stopRenew != nil {
		close(l.stopRenew)
		<-l.renewDone // 等待续期协程退出
	}
}

// ==================== 空锁实现（单实例模式） ====================

// NoopLock 空锁实现
//
// 不做任何锁操作，适用于单实例部署
type NoopLock struct{}

// NewNoopLock 创建空锁
func NewNoopLock() *NoopLock {
	return &NoopLock{}
}

// TryLock 总是返回成功
func (l *NoopLock) TryLock(ctx context.Context) (bool, error) {
	return true, nil
}

// Unlock 不做任何操作
func (l *NoopLock) Unlock(ctx context.Context) error {
	return nil
}

// Refresh 不做任何操作
func (l *NoopLock) Refresh(ctx context.Context) error {
	return nil
}

// ==================== 工厂方法 ====================

// LockConfig 锁配置
type LockConfig struct {
	Enabled bool          // 是否启用分布式锁
	Key     string        // 锁的 key
	TTL     time.Duration // 锁的过期时间
}

// DefaultReconcileLockConfig 默认的对账锁配置
func DefaultReconcileLockConfig() LockConfig {
	return LockConfig{
		Enabled: true,
		Key:     "lock:activity:tag:reconcile",
		TTL:     30 * time.Second,
	}
}

// NewDistributedLock 创建分布式锁
//
// 根据配置决定使用 Redis 锁还是空锁
func NewDistributedLock(rds *redis.Redis, config LockConfig) DistributedLock {
	if !config.Enabled || rds == nil {
		return NewNoopLock()
	}
	return NewRedisLock(rds, config.Key, config.TTL)
}
