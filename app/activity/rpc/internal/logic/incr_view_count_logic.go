package logic

import (
	"context"
	"errors"
	"fmt"

	"activity-platform/app/activity/model"
	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/activity/rpc/internal/svc"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type IncrViewCountLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewIncrViewCountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *IncrViewCountLogic {
	return &IncrViewCountLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// ==================== 浏览量接口 ====================

// IncrViewCount 增加活动浏览量
//
// 业务逻辑：
//  1. 参数校验（活动 ID 必须大于 0）
//  2. 防刷检查（同一用户/IP 对同一活动 1 小时内只计算一次）
//  3. 查询活动是否存在
//  4. 原子更新浏览量（使用 gorm.Expr）
//  5. 返回最新浏览量
//
// 防刷策略：
//   - 使用 Redis SETEX 实现，key 格式: activity:view:{activity_id}:{user_id|ip}
//   - TTL: 1 小时
//   - 登录用户使用 user_id，未登录用户使用 client_ip
//
// 设计说明：
//   - 浏览量只增不减
//   - 使用原子操作避免并发问题
//   - 防刷失败不影响正常返回（降级处理）
func (l *IncrViewCountLogic) IncrViewCount(in *activity.IncrViewCountReq) (*activity.IncrViewCountResp, error) {
	// 1. 参数校验
	if in.GetId() <= 0 {
		return nil, errorx.ErrInvalidParams("活动ID无效")
	}

	activityID := uint64(in.GetId())

	// 2. 查询活动是否存在
	activityData, err := l.svcCtx.ActivityModel.FindByID(l.ctx, activityID)
	if err != nil {
		if errors.Is(err, model.ErrActivityNotFound) {
			return nil, errorx.New(errorx.CodeActivityNotFound)
		}
		l.Errorf("查询活动失败: id=%d, err=%v", activityID, err)
		return nil, errorx.ErrDBError(err)
	}

	// 3. 只对公开状态的活动计数
	if !activityData.IsPublic() {
		// 非公开状态，直接返回当前浏览量，不报错
		return &activity.IncrViewCountResp{
			ViewCount: int64(activityData.ViewCount),
		}, nil
	}

	// 4. 防刷检查
	viewerKey := l.buildViewerKey(in)
	if viewerKey == "" {
		// 无法识别访问者，直接返回当前浏览量（不计数）
		l.Infof("[WARNING] 无法识别访问者: activity_id=%d", activityID)
		return &activity.IncrViewCountResp{
			ViewCount: int64(activityData.ViewCount),
		}, nil
	}

	// 检查是否已浏览过
	redisKey := fmt.Sprintf("activity:view:%d:%s", activityID, viewerKey)
	exists, err := l.svcCtx.Redis.ExistsCtx(l.ctx, redisKey)
	if err != nil {
		// Redis 错误时降级处理：允许计数
		l.Infof("[WARNING] Redis 防刷检查失败: key=%s, err=%v", redisKey, err)
	} else if exists {
		// 已浏览过，不重复计数
		l.Debugf("重复浏览，跳过计数: activity_id=%d, viewer=%s", activityID, viewerKey)
		return &activity.IncrViewCountResp{
			ViewCount: int64(activityData.ViewCount),
		}, nil
	}

	// 5. 设置防刷标记（1小时过期）
	err = l.svcCtx.Redis.SetexCtx(l.ctx, redisKey, "1", 3600)
	if err != nil {
		// Redis 设置失败，降级处理：仍然计数
		l.Infof("[WARNING] Redis 设置防刷标记失败: key=%s, err=%v", redisKey, err)
	}

	// 6. 原子更新浏览量
	err = l.svcCtx.ActivityModel.IncrViewCount(l.ctx, activityID, 1)
	if err != nil {
		l.Errorf("更新浏览量失败: activity_id=%d, err=%v", activityID, err)
		// 更新失败，返回当前浏览量
		return &activity.IncrViewCountResp{
			ViewCount: int64(activityData.ViewCount),
		}, nil
	}

	// 7. 返回更新后的浏览量（+1）
	newViewCount := int64(activityData.ViewCount) + 1
	l.Infof("浏览量更新成功: activity_id=%d, viewer=%s, new_count=%d",
		activityID, viewerKey, newViewCount)

	return &activity.IncrViewCountResp{
		ViewCount: newViewCount,
	}, nil
}

// buildViewerKey 构建访问者标识
//
// 优先级：
//  1. 登录用户使用 user_id
//  2. 未登录用户使用 client_ip
//
// 返回空字符串表示无法识别访问者
func (l *IncrViewCountLogic) buildViewerKey(in *activity.IncrViewCountReq) string {
	// 优先使用 user_id
	if in.GetUserId() > 0 {
		return fmt.Sprintf("u_%d", in.GetUserId())
	}

	// 使用 client_ip
	if in.GetClientIp() != "" {
		return fmt.Sprintf("ip_%s", in.GetClientIp())
	}

	return ""
}
