package logic

import (
	"context"
	"errors"

	"activity-platform/app/activity/model"
	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/activity/rpc/internal/svc"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type DeleteActivityLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteActivityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteActivityLogic {
	return &DeleteActivityLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// DeleteActivity 删除活动（软删除）
// 权限规则：
//   - 管理员(is_admin=true)：可删除任何活动
//   - 组织者：只能删除自己创建的活动，且需满足状态和报名条件
//
// 状态规则（非管理员）：
//   - 草稿(0)/待审核(1)/已拒绝(5)：可直接删除
//   - 已发布(2)/进行中(3)：无报名时可删除，有报名时不可删除
//   - 已结束(4)/已取消(6)：不可删除（保留历史记录）
func (l *DeleteActivityLogic) DeleteActivity(in *activity.DeleteActivityReq) (*activity.DeleteActivityResp, error) {
	// 1. 参数校验
	if err := l.validateParams(in); err != nil {
		return nil, err
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

	// 3. 权限校验
	if err := l.checkPermission(activityData, in); err != nil {
		return nil, err
	}

	// 4. 状态和报名校验（非管理员需要校验）
	if !in.IsAdmin {
		if err := l.checkDeletable(activityData); err != nil {
			return nil, err
		}
	}

	// 5. 事务：删除活动 + 清理关联数据
	err = l.svcCtx.DB.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
		// 5.1 获取关联的标签ID（用于后续更新统计）
		tagIDs, err := l.svcCtx.ActivityTagModel.FindIDsByActivityID(l.ctx, uint64(in.Id))
		if err != nil {
			l.Errorf("查询活动标签失败: activityId=%d, err=%v", in.Id, err)
			// 标签查询失败不阻塞删除流程
			tagIDs = nil
		}

		// 5.2 删除标签关联
		if err := l.svcCtx.ActivityTagModel.UnbindFromActivity(l.ctx, tx, uint64(in.Id)); err != nil {
			l.Errorf("删除标签关联失败: activityId=%d, err=%v", in.Id, err)
			return err
		}

		// 5.3 软删除活动
		if err := tx.Delete(&model.Activity{}, in.Id).Error; err != nil {
			l.Errorf("删除活动失败: id=%d, err=%v", in.Id, err)
			return err
		}

		// 5.4 更新活动标签统计（减少 activity_count）
		if len(tagIDs) > 0 {
			if err := l.svcCtx.TagStatsModel.BatchDecrActivityCount(l.ctx, tx, tagIDs); err != nil {
				l.Errorf("更新标签统计失败: tagIDs=%v, err=%v", tagIDs, err)
				// 统计更新失败不影响主流程
			}
		}

		return nil
	})

	if err != nil {
		l.Errorf("删除活动事务失败: id=%d, err=%v", in.Id, err)
		return nil, errorx.ErrDBError(err)
	}

	// 删除缓存（删除成功后）
	if l.svcCtx.ActivityCache != nil {
		if err := l.svcCtx.ActivityCache.Invalidate(l.ctx, uint64(in.Id)); err != nil {
			// 缓存删除失败不影响主流程，记录日志即可
			l.Infof("[WARNING] 删除活动缓存失败: id=%d, err=%v", in.Id, err)
		}
	}

	// 异步从 ES 删除文档
	if l.svcCtx.SyncService != nil {
		l.svcCtx.SyncService.DeleteActivityAsync(uint64(in.Id))
	}

	l.Infof("活动删除成功: id=%d, operatorId=%d, isAdmin=%v",
		in.Id, in.OperatorId, in.IsAdmin)

	return &activity.DeleteActivityResp{
		Success: true,
	}, nil
}

// validateParams 参数校验
func (l *DeleteActivityLogic) validateParams(in *activity.DeleteActivityReq) error {
	if in.Id <= 0 {
		return errorx.ErrInvalidParams("活动ID无效")
	}
	if in.OperatorId <= 0 {
		return errorx.ErrInvalidParams("操作者信息缺失")
	}
	return nil
}

// checkPermission 权限校验
func (l *DeleteActivityLogic) checkPermission(activityData *model.Activity, in *activity.DeleteActivityReq) error {
	// 管理员可以删除任何活动
	if in.IsAdmin {
		l.Infof("[管理员删除] activityId=%d, adminId=%d", in.Id, in.OperatorId)
		return nil
	}

	// 非管理员只能删除自己创建的活动
	if activityData.OrganizerID != uint64(in.OperatorId) {
		l.Infof("[权限拒绝] 无权限删除活动: activityId=%d, organizerId=%d, operatorId=%d",
			in.Id, activityData.OrganizerID, in.OperatorId)
		return errorx.New(errorx.CodeActivityPermissionDenied)
	}

	return nil
}

// checkDeletable 检查活动是否可删除（状态和报名校验）
func (l *DeleteActivityLogic) checkDeletable(activityData *model.Activity) error {
	status := activityData.Status

	// 可直接删除的状态：草稿、待审核、已拒绝
	if status == model.StatusDraft || status == model.StatusPending || status == model.StatusRejected {
		return nil
	}

	// 不可删除的状态：已结束、已取消
	if status == model.StatusFinished || status == model.StatusCancelled {
		l.Infof("[状态拒绝] 活动状态不允许删除: id=%d, status=%d(%s)",
			activityData.ID, status, activityData.StatusText())
		return errorx.NewWithMessage(errorx.CodeActivityStatusInvalid,
			"已结束或已取消的活动不能删除")
	}

	// 已发布/进行中状态：检查是否有报名
	if status == model.StatusPublished || status == model.StatusOngoing {
		if activityData.CurrentParticipants > 0 {
			l.Infof("[报名限制] 有报名记录不能删除: id=%d, currentParticipants=%d",
				activityData.ID, activityData.CurrentParticipants)
			return errorx.New(errorx.CodeActivityHasRegistration)
		}
		// 无报名，可以删除
		return nil
	}

	// 未知状态，拒绝删除
	l.Errorf("[未知状态] 活动状态异常: id=%d, status=%d", activityData.ID, status)
	return errorx.NewWithMessage(errorx.CodeActivityStatusInvalid, "活动状态异常")
}
