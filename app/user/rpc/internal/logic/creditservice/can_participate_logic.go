/**
 * @projectName: CampusHub
 * @package: creditservicelogic
 * @className: CanParticipateLogic
 * @author: lijunqi
 * @description: 校验报名资格逻辑层（Cache-Aside模式）
 * @date: 2026-01-30
 * @version: 1.0
 */

package creditservicelogic

import (
	"context"
	"fmt"
	"time"

	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"
	"activity-platform/common/constants"
	"activity-platform/common/errorx"

	"github.com/go-redis/redis/v8"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

// CanParticipateLogic 校验报名资格逻辑处理器
type CanParticipateLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

// NewCanParticipateLogic 创建校验报名资格逻辑实例
func NewCanParticipateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CanParticipateLogic {
	return &CanParticipateLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// CanParticipate 校验是否允许报名
// 业务逻辑:
//   - score < 60: 黑名单，禁止报名
//   - 60 <= score < 70: 风险用户，每日限报名1次
//   - score >= 70: 正常报名
//
// 使用 Cache-Aside 模式读取信用分缓存
func (l *CanParticipateLogic) CanParticipate(in *pb.CanParticipateReq) (*pb.CanParticipateResp, error) {
	// 1. 参数校验
	if in.UserId <= 0 {
		l.Errorf("CanParticipate 参数错误: userId=%d", in.UserId)
		return nil, errorx.ErrInvalidParams("用户ID无效")
	}

	// 2. 使用 Cache-Aside 模式获取信用分
	score, level, err := l.getCreditWithCache(in.UserId)
	if err != nil {
		return nil, err
	}

	// 3. 根据信用分判断是否允许报名
	// 3.1 黑名单用户（score < 60）：禁止报名
	if score < constants.CreditThresholdBlacklist {
		l.Infof("CanParticipate 黑名单用户禁止报名: userId=%d, score=%d", in.UserId, score)
		return &pb.CanParticipateResp{
			Allowed: false,
			Reason:  fmt.Sprintf("信用分过低(%d分)，账户已被限制报名", score),
			Score:   int64(score),
			Level:   int32(level),
		}, nil
	}

	// 3.2 风险用户（60 <= score < 70）：每日限报名1次
	if score < constants.CreditThresholdRisk {
		allowed, reason, err := l.checkRiskUserDailyLimit(in.UserId, score)
		if err != nil {
			return nil, err
		}
		if !allowed {
			return &pb.CanParticipateResp{
				Allowed: false,
				Reason:  reason,
				Score:   int64(score),
				Level:   int32(level),
			}, nil
		}
	}

	// 3.3 正常用户（score >= 70）：允许报名
	l.Infof("CanParticipate 允许报名: userId=%d, score=%d, level=%d", in.UserId, score, level)

	return &pb.CanParticipateResp{
		Allowed: true,
		Reason:  "",
		Score:   int64(score),
		Level:   int32(level),
	}, nil
}

// getCreditWithCache 使用 Cache-Aside 模式获取信用分
// 流程: 先查缓存 -> 缓存未命中查MySQL -> 回填缓存
func (l *CanParticipateLogic) getCreditWithCache(userID int64) (score int, level int8, err error) {
	// 1. 先尝试从缓存获取
	cacheData, exists, cacheErr := l.svcCtx.CreditCache.Get(l.ctx, userID)
	if cacheErr != nil {
		// Redis 错误，记录日志但继续查库（降级处理）
		l.Errorf("getCreditWithCache Redis读取失败: userId=%d, err=%v", userID, cacheErr)
	}
	if exists && cacheData != nil {
		return cacheData.Score, cacheData.Level, nil
	}

	// 2. 缓存未命中，查询 MySQL
	credit, err := l.svcCtx.UserCreditModel.FindByUserID(l.ctx, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			l.Infof("getCreditWithCache 信用记录不存在: userId=%d", userID)
			return 0, 0, errorx.ErrCreditNotFound()
		}
		l.Errorf("getCreditWithCache 查询信用记录失败: userId=%d, err=%v", userID, err)
		return 0, 0, errorx.ErrDBError(err)
	}

	// 3. 回填缓存（异步，不阻塞主流程）
	go func() {
		if setErr := l.svcCtx.CreditCache.Set(l.ctx, userID, credit.Score, credit.Level); setErr != nil {
			l.Errorf("getCreditWithCache 回填缓存失败: userId=%d, err=%v", userID, setErr)
		}
	}()

	l.Infof("getCreditWithCache 缓存未命中，查库成功: userId=%d, score=%d", userID, credit.Score)
	return credit.Score, credit.Level, nil
}

// checkRiskUserDailyLimit 检查风险用户每日报名限制
func (l *CanParticipateLogic) checkRiskUserDailyLimit(
	userID int64,
	score int,
) (allowed bool, reason string, err error) {
	todayCount, err := l.getTodayParticipateCount(userID)
	if err != nil {
		l.Errorf("checkRiskUserDailyLimit 获取今日报名次数失败: userId=%d, err=%v", userID, err)
		return false, "", errorx.ErrCacheError(err)
	}

	if todayCount >= constants.RiskUserDailyParticipateLimit {
		l.Infof("checkRiskUserDailyLimit 风险用户今日限额已用完: userId=%d, score=%d, todayCount=%d",
			userID, score, todayCount)
		return false, fmt.Sprintf("信用分处于风险区间(%d分)，每日仅限报名%d次，今日已用完",
			score, constants.RiskUserDailyParticipateLimit), nil
	}

	l.Infof("checkRiskUserDailyLimit 风险用户允许报名: userId=%d, score=%d, todayCount=%d",
		userID, score, todayCount)
	return true, "", nil
}

// getTodayParticipateCount 获取风险用户今日报名次数
// 使用 Redis INCR 记录每日报名计数
func (l *CanParticipateLogic) getTodayParticipateCount(userID int64) (int, error) {
	today := time.Now().Format("2006-01-02")
	key := fmt.Sprintf("%s%d:%s", constants.CacheRiskDailyCountPrefix, userID, today)

	count, err := l.svcCtx.Redis.Get(l.ctx, key).Int()
	if err != nil {
		if err == redis.Nil {
			return 0, nil
		}
		return 0, err
	}

	return count, nil
}

// IncrementRiskUserDailyCount 风险用户报名成功后，递增每日计数
// 由 Activity 服务报名成功后调用
func (l *CanParticipateLogic) IncrementRiskUserDailyCount(userID int64) error {
	today := time.Now().Format("2006-01-02")
	key := fmt.Sprintf("%s%d:%s", constants.CacheRiskDailyCountPrefix, userID, today)

	// INCR 并设置过期时间（当日剩余秒数 + 1小时buffer）
	pipe := l.svcCtx.Redis.Pipeline()
	pipe.Incr(l.ctx, key)

	// 计算当日剩余秒数
	now := time.Now()
	endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
	ttl := endOfDay.Sub(now) + time.Hour

	pipe.Expire(l.ctx, key, ttl)
	_, err := pipe.Exec(l.ctx)

	return err
}
