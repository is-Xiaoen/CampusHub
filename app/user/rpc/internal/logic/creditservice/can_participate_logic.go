/**
 * @projectName: CampusHub
 * @package: creditservicelogic
 * @className: CanParticipateLogic
 * @author: lijunqi
 * @description: 校验报名资格逻辑层
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
func (l *CanParticipateLogic) CanParticipate(in *pb.CanParticipateReq) (*pb.CanParticipateResp, error) {
	// 1. 参数校验
	if in.UserId <= 0 {
		l.Errorf("CanParticipate 参数错误: userId=%d", in.UserId)
		return nil, errorx.ErrInvalidParams("用户ID无效")
	}

	// 2. 查询信用记录
	credit, err := l.svcCtx.UserCreditModel.FindByUserID(l.ctx, in.UserId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			l.Infof("CanParticipate 信用记录不存在: userId=%d", in.UserId)
			return nil, errorx.ErrCreditNotFound()
		}
		l.Errorf("CanParticipate 查询信用记录失败: userId=%d, err=%v", in.UserId, err)
		return nil, errorx.ErrDBError(err)
	}

	// 3. 根据信用分判断是否允许报名
	score := credit.Score
	level := credit.Level

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
		// 检查今日是否已报名
		todayCount, err := l.getTodayParticipateCount(in.UserId)
		if err != nil {
			l.Errorf("CanParticipate 获取今日报名次数失败: userId=%d, err=%v", in.UserId, err)
			return nil, errorx.ErrCacheError(err)
		}

		if todayCount >= constants.RiskUserDailyParticipateLimit {
			l.Infof("CanParticipate 风险用户今日限额已用完: userId=%d, score=%d, todayCount=%d",
				in.UserId, score, todayCount)
			return &pb.CanParticipateResp{
				Allowed: false,
				Reason:  fmt.Sprintf("信用分处于风险区间(%d分)，每日仅限报名%d次，今日已用完", score, constants.RiskUserDailyParticipateLimit),
				Score:   int64(score),
				Level:   int32(level),
			}, nil
		}

		l.Infof("CanParticipate 风险用户允许报名: userId=%d, score=%d, todayCount=%d",
			in.UserId, score, todayCount)
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

// getTodayParticipateCount 获取风险用户今日报名次数
// 使用 Redis 记录每日报名计数
func (l *CanParticipateLogic) getTodayParticipateCount(userID int64) (int, error) {
	// Redis Key 格式: risk:participate:daily:{userId}:{date}
	today := time.Now().Format("2006-01-02")
	key := fmt.Sprintf("%s%d:%s", constants.CacheRiskUserDailyCountPrefix, userID, today)

	// 从 Redis 获取计数
	count, err := l.svcCtx.Redis.Get(l.ctx, key).Int()
	if err != nil {
		// Key 不存在，返回0
		if err.Error() == "redis: nil" {
			return 0, nil
		}
		return 0, err
	}

	return count, nil
}
