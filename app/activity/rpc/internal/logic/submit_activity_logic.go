package logic

import (
	"context"
	"errors"
	"time"

	"activity-platform/app/activity/model"
	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/activity/rpc/internal/svc"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type SubmitActivityLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSubmitActivityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SubmitActivityLogic {
	return &SubmitActivityLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// SubmitActivity 提交活动（草稿 → 已发布）
// MVP 版本：没有后台管理系统，审批自动通过，直接发布
// 状态流转：Draft(0) → Published(2)，跳过 Pending(1)
func (l *SubmitActivityLogic) SubmitActivity(in *activity.SubmitActivityReq) (*activity.SubmitActivityResp, error) {
	// 1. 参数校验
	if in.Id <= 0 {
		return nil, errorx.ErrInvalidParams("活动ID无效")
	}
	if in.OperatorId <= 0 {
		return nil, errorx.ErrInvalidParams("操作者信息缺失")
	}

	// 2. 查询活动
	activityData, err := l.svcCtx.ActivityModel.FindByID(l.ctx, uint64(in.Id))
	if err != nil {
		if errors.Is(err, model.ErrActivityNotFound) {
			return nil, errorx.New(errorx.CodeActivityNotFound)
		}
		l.Errorf("查询活动失败: id=%d, err=%v", in.Id, err)
		return nil, errorx.ErrDBError(err)
	}

	// 3. 权限校验：只有组织者能提交自己的活动
	if activityData.OrganizerID != uint64(in.OperatorId) {
		l.Infof("[权限拒绝] 无权限提交活动: activityId=%d, organizerId=%d, operatorId=%d",
			in.Id, activityData.OrganizerID, in.OperatorId)
		return nil, errorx.New(errorx.CodeActivityPermissionDenied)
	}

	// 4. 状态校验：只有草稿状态可以提交
	// MVP 版本也支持从"已拒绝"状态重新提交（虽然MVP没有拒绝功能，但保留兼容性）
	if activityData.Status != model.StatusDraft && activityData.Status != model.StatusRejected {
		l.Infof("[状态拒绝] 活动状态不允许提交: id=%d, currentStatus=%d",
			in.Id, activityData.Status)
		return nil, errorx.NewWithMessage(errorx.CodeActivityStatusInvalid,
			"只有草稿状态的活动可以提交")
	}

	// 5. 时间重新校验：草稿可能保存很久，提交时需确认时间仍有效
	now := time.Now().Unix()
	if activityData.ActivityEndTime <= now {
		return nil, errorx.NewWithMessage(errorx.CodeActivityTimeInvalid,
			"活动结束时间已过期，请修改后重新提交")
	}
	if activityData.ActivityStartTime <= now {
		return nil, errorx.NewWithMessage(errorx.CodeActivityTimeInvalid,
			"活动开始时间已过期，请修改后重新提交")
	}
	if activityData.RegisterEndTime <= now {
		return nil, errorx.NewWithMessage(errorx.CodeActivityTimeInvalid,
			"报名截止时间已过期，请修改后重新提交")
	}

	// 6. 事务：更新状态 + 记录日志
	err = l.svcCtx.DB.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
		// 6.1 更新状态为已发布（MVP: 跳过待审核）
		oldStatus := activityData.Status
		err := l.svcCtx.ActivityModel.UpdateStatus(
			l.ctx, tx,
			uint64(in.Id),
			activityData.Version,
			model.StatusPublished, // MVP: 直接发布
			"",                    // 无拒绝原因
		)
		if err != nil {
			if errors.Is(err, model.ErrActivityConcurrentUpdate) {
				return err
			}
			return err
		}

		// 6.2 记录状态变更日志
		statusLog := &model.ActivityStatusLog{
			ActivityID:   uint64(in.Id),
			FromStatus:   oldStatus,
			ToStatus:     model.StatusPublished,
			OperatorID:   uint64(in.OperatorId),
			OperatorType: model.OperatorTypeUser,
			Reason:       "MVP自动审批通过",
		}
		if err := l.svcCtx.StatusLogModel.Create(l.ctx, tx, statusLog); err != nil {
			l.Errorf("记录状态日志失败: %v", err)
			// 日志记录失败不影响主流程，但仍记录错误
			// 如果要求严格一致性，可以 return err
		}

		return nil
	})

	if err != nil {
		if errors.Is(err, model.ErrActivityConcurrentUpdate) {
			return nil, errorx.New(errorx.CodeActivityConcurrentUpdate)
		}
		l.Errorf("提交活动失败: id=%d, err=%v", in.Id, err)
		return nil, errorx.ErrDBError(err)
	}

	// 删除缓存（状态变更成功后）
	if l.svcCtx.ActivityCache != nil {
		if err := l.svcCtx.ActivityCache.Invalidate(l.ctx, uint64(in.Id)); err != nil {
			l.Infof("[WARNING] 删除活动缓存失败: id=%d, err=%v", in.Id, err)
		}
	}

	// 异步同步到 ES（发布后需要被搜索到）
	if l.svcCtx.SyncService != nil {
		// 重新查询最新数据用于同步
		updatedActivity, err := l.svcCtx.ActivityModel.FindByID(l.ctx, uint64(in.Id))
		if err == nil {
			l.svcCtx.SyncService.IndexActivityAsync(updatedActivity)
		}
	}

	// 异步发布活动创建事件（草稿提交发布后通知 Chat 创建群聊）
	if l.svcCtx.MsgProducer != nil {
		l.svcCtx.MsgProducer.PublishActivityCreated(
			l.ctx, uint64(in.Id), activityData.OrganizerID, activityData.Title,
		)
	}

	l.Infof("活动提交成功（MVP自动发布）: id=%d, status=%d->%d",
		in.Id, activityData.Status, model.StatusPublished)

	return &activity.SubmitActivityResp{
		Status: int32(model.StatusPublished),
	}, nil
}
