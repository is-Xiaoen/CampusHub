/**
 * @projectName: CampusHub
 * @package: creditservicelogic
 * @className: CanPublishLogic
 * @author: lijunqi
 * @description: 校验发布资格逻辑层（Cache-Aside模式）
 * @date: 2026-01-30
 * @version: 1.0
 */

package creditservicelogic

import (
	"context"
	"fmt"

	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"
	"activity-platform/common/constants"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

// CanPublishLogic 校验发布资格逻辑处理器
type CanPublishLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

// NewCanPublishLogic 创建校验发布资格逻辑实例
func NewCanPublishLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CanPublishLogic {
	return &CanPublishLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// CanPublish 校验是否允许发布活动
// 业务逻辑:
//   - score >= 90: 允许发布（Lv3优秀用户、Lv4社区之星）
//   - score < 90: 禁止发布
//
// 使用 Cache-Aside 模式读取信用分缓存
func (l *CanPublishLogic) CanPublish(in *pb.CanPublishReq) (*pb.CanPublishResp, error) {
	// 1. 参数校验
	if in.UserId <= 0 {
		l.Errorf("CanPublish 参数错误: userId=%d", in.UserId)
		return nil, errorx.ErrInvalidParams("用户ID无效")
	}

	// 2. 使用 Cache-Aside 模式获取信用分
	score, level, err := l.getCreditWithCache(in.UserId)
	if err != nil {
		return nil, err
	}

	// 3. 根据信用分判断是否允许发布
	// 3.1 信用分不足（score < 90）：禁止发布
	if score < constants.CreditThresholdPublish {
		l.Infof("CanPublish 信用分不足禁止发布: userId=%d, score=%d, threshold=%d",
			in.UserId, score, constants.CreditThresholdPublish)
		return &pb.CanPublishResp{
			Allowed: false,
			Reason:  fmt.Sprintf("信用分不足%d分（当前%d分），暂时无法发布活动，请先通过参与活动积累信用", constants.CreditThresholdPublish, score),
			Score:   int64(score),
			Level:   int32(level),
		}, nil
	}

	// 3.2 信用分充足（score >= 90）：允许发布
	l.Infof("CanPublish 允许发布: userId=%d, score=%d, level=%d", in.UserId, score, level)

	return &pb.CanPublishResp{
		Allowed: true,
		Reason:  "",
		Score:   int64(score),
		Level:   int32(level),
	}, nil
}

// getCreditWithCache 使用 Cache-Aside 模式获取信用分
// 流程: 先查缓存 -> 缓存未命中查MySQL -> 回填缓存
func (l *CanPublishLogic) getCreditWithCache(userID int64) (score int, level int8, err error) {
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
