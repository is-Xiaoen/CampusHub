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

// cancellableStatuses 可取消的活动状态集合
// 设计说明：
// - 草稿/待审核/已发布/进行中 可以取消
// - 已结束(4)：保留历史记录，不允许取消
// - 已拒绝(5)：应该重新编辑提交，不需要取消
// - 已取消(6)：已经是取消状态
var cancellableStatuses = map[int8]bool{
	model.StatusDraft:     true,
	model.StatusPending:   true,
	model.StatusPublished: true,
	model.StatusOngoing:   true,
}

type CancelActivityLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCancelActivityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CancelActivityLogic {
	return &CancelActivityLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// CancelActivity 取消活动
// 权限规则：
//   - 管理员(is_admin=true)：可取消任何活动
//   - 组织者：只能取消自己创建的活动
//
// 状态规则：
//   - 草稿(0)/待审核(1)/已发布(2)/进行中(3)：可取消
//   - 已结束(4)：不可取消（保留历史记录）
//   - 已拒绝(5)：不可取消（应重新编辑）
//   - 已取消(6)：不可取消（已经是取消状态）
func (l *CancelActivityLogic) CancelActivity(in *activity.CancelActivityReq) (*activity.CancelActivityResp, error) {
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

	// 4. 状态校验
	if err := l.checkCancellable(activityData); err != nil {
		return nil, err
	}

	// 5. 事务：更新状态 + 记录日志
	oldStatus := activityData.Status
	err = l.svcCtx.DB.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
		// 5.1 更新状态为已取消（带乐观锁）
		err := l.svcCtx.ActivityModel.UpdateStatus(
			l.ctx, tx,
			uint64(in.Id),
			activityData.Version,
			model.StatusCancelled,
			in.Reason, // 取消原因存入 reject_reason 字段（复用）
		)
		if err != nil {
			return err
		}

		// 5.2 记录状态变更日志
		operatorType := model.OperatorTypeUser
		if in.IsAdmin {
			operatorType = model.OperatorTypeAdmin
		}

		statusLog := &model.ActivityStatusLog{
			ActivityID:   uint64(in.Id),
			FromStatus:   oldStatus,
			ToStatus:     model.StatusCancelled,
			OperatorID:   uint64(in.OperatorId),
			OperatorType: operatorType,
			Reason:       l.buildLogReason(in),
		}
		if err := l.svcCtx.StatusLogModel.Create(l.ctx, tx, statusLog); err != nil {
			l.Errorf("记录状态日志失败: %v", err)
			// 日志记录失败不影响主流程
		}

		return nil
	})

	if err != nil {
		if errors.Is(err, model.ErrActivityConcurrentUpdate) {
			return nil, errorx.New(errorx.CodeActivityConcurrentUpdate)
		}
		l.Errorf("取消活动失败: id=%d, err=%v", in.Id, err)
		return nil, errorx.ErrDBError(err)
	}

	l.Infof("活动取消成功: id=%d, operatorId=%d, isAdmin=%v, fromStatus=%d(%s), reason=%s",
		in.Id, in.OperatorId, in.IsAdmin, oldStatus, model.StatusText[oldStatus], in.Reason)

	return &activity.CancelActivityResp{
		Status: int32(model.StatusCancelled),
	}, nil
}

// validateParams 参数校验
func (l *CancelActivityLogic) validateParams(in *activity.CancelActivityReq) error {
	if in.Id <= 0 {
		return errorx.ErrInvalidParams("活动ID无效")
	}
	if in.OperatorId <= 0 {
		return errorx.ErrInvalidParams("操作者信息缺失")
	}
	// reason 可选，不强制要求
	return nil
}

// checkPermission 权限校验
func (l *CancelActivityLogic) checkPermission(activityData *model.Activity, in *activity.CancelActivityReq) error {
	// 管理员可以取消任何活动
	if in.IsAdmin {
		l.Infof("[管理员取消] activityId=%d, adminId=%d", in.Id, in.OperatorId)
		return nil
	}

	// 非管理员只能取消自己创建的活动
	if activityData.OrganizerID != uint64(in.OperatorId) {
		l.Infof("[权限拒绝] 无权限取消活动: activityId=%d, organizerId=%d, operatorId=%d",
			in.Id, activityData.OrganizerID, in.OperatorId)
		return errorx.New(errorx.CodeActivityPermissionDenied)
	}

	return nil
}

// checkCancellable 检查活动是否可取消
func (l *CancelActivityLogic) checkCancellable(activityData *model.Activity) error {
	status := activityData.Status

	// 检查是否在可取消状态集合中
	if cancellableStatuses[status] {
		return nil
	}

	// 针对不同状态返回特定错误消息
	switch status {
	case model.StatusFinished:
		l.Infof("[状态拒绝] 已结束活动不能取消: id=%d", activityData.ID)
		return errorx.NewWithMessage(errorx.CodeActivityStatusInvalid,
			"已结束的活动不能取消")

	case model.StatusRejected:
		l.Infof("[状态拒绝] 已拒绝活动不能取消: id=%d", activityData.ID)
		return errorx.NewWithMessage(errorx.CodeActivityStatusInvalid,
			"已拒绝的活动不能取消，请重新编辑后提交")

	case model.StatusCancelled:
		l.Infof("[状态拒绝] 活动已经是取消状态: id=%d", activityData.ID)
		return errorx.NewWithMessage(errorx.CodeActivityStatusInvalid,
			"活动已经被取消")

	default:
		l.Errorf("[未知状态] 活动状态异常: id=%d, status=%d", activityData.ID, status)
		return errorx.NewWithMessage(errorx.CodeActivityStatusInvalid, "活动状态异常")
	}
}

// buildLogReason 构建日志记录的原因
func (l *CancelActivityLogic) buildLogReason(in *activity.CancelActivityReq) string {
	if in.Reason != "" {
		return in.Reason
	}
	if in.IsAdmin {
		return "管理员取消活动"
	}
	return "组织者取消活动"
}
