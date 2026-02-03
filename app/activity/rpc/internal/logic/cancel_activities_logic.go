package logic

import (
	"context"
	"errors"
	"time"

	"activity-platform/app/activity/model"
	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/activity/rpc/internal/svc"
	"activity-platform/common/ctxdata"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type CancelActivitiesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCancelActivitiesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CancelActivitiesLogic {
	return &CancelActivitiesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// CancelActivities 取消报名活动
func (l *CancelActivitiesLogic) CancelActivities(in *activity.CancelActivityRequest) (*activity.CancelActivityResponse, error) {
	// 1) 参数与登录校验
	activityID := in.GetActivityId()
	userID := ctxdata.GetUserIDFromCtx(l.ctx)
	if activityID <= 0 || userID <= 0 {
		return &activity.CancelActivityResponse{Result: "fail"}, nil
	}

	// 2) 活动有效性与状态校验
	activityData, err := l.svcCtx.ActivityModel.FindByID(l.ctx, uint64(activityID))
	if err != nil {
		if errors.Is(err, model.ErrActivityNotFound) {
			return &activity.CancelActivityResponse{Result: "fail"}, nil
		}
		l.Errorf("活动查询失败: activityId=%d, err=%v", activityID, err)
		return &activity.CancelActivityResponse{Result: "fail"}, nil
	}
	if activityData.Status != model.StatusPublished && activityData.Status != model.StatusOngoing {
		return &activity.CancelActivityResponse{Result: "fail"}, nil
	}

	// 3) 事务内处理：校验报名状态 -> 作废票券 -> 回退名额
	now := time.Now().Unix()
	errAlreadyCanceled := errors.New("registration already canceled")
	errRegistrationInvalid := errors.New("registration status invalid")
	errTicketUsed := errors.New("ticket already used")
	errCountUpdate := errors.New("participant count update failed")

	err = l.svcCtx.DB.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
		// 3.1 查询报名记录并校验状态
		reg, err := l.svcCtx.ActivityRegistrationModel.FindByActivityUserTx(
			l.ctx, tx, uint64(activityID), uint64(userID),
		)
		if err != nil {
			return err
		}
		switch reg.Status {
		case model.RegistrationStatusCanceled:
			return errAlreadyCanceled
		case model.RegistrationStatusSuccess:
		default:
			return errRegistrationInvalid
		}

		// 3.2 查询关联票券，已核销则禁止取消
		ticket, err := l.svcCtx.ActivityTicketModel.FindByRegistrationIDTx(l.ctx, tx, reg.ID)
		if err != nil && !errors.Is(err, model.ErrTicketNotFound) {
			return err
		}
		if err == nil && ticket.Status == model.TicketStatusUsed {
			return errTicketUsed
		}

		// 3.3 更新报名记录为取消
		result := tx.Model(&model.ActivityRegistration{}).
			Where("id = ? AND status = ?", reg.ID, model.RegistrationStatusSuccess).
			Updates(map[string]interface{}{
				"status":      model.RegistrationStatusCanceled,
				"cancel_time": now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errRegistrationInvalid
		}

		// 3.4 作废票券（如存在）
		if err == nil {
			if err := tx.Model(&model.ActivityTicket{}).
				Where("id = ? AND status <> ?", ticket.ID, model.TicketStatusUsed).
				Update("status", model.TicketStatusVoid).Error; err != nil {
				return err
			}
		}

		// 3.5 回退活动报名人数
		result = tx.Model(&model.Activity{}).
			Where("id = ? AND current_participants > 0", activityID).
			Update("current_participants", gorm.Expr("current_participants - 1"))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errCountUpdate
		}

		return nil
	})
	if err != nil {
		// 4) 幂等与业务错误处理
		if errors.Is(err, errAlreadyCanceled) {
			return &activity.CancelActivityResponse{Result: "success"}, nil
		}
		if errors.Is(err, model.ErrRegistrationNotFound) ||
			errors.Is(err, errRegistrationInvalid) ||
			errors.Is(err, errTicketUsed) {
			return &activity.CancelActivityResponse{Result: "fail"}, nil
		}
		l.Errorf("取消报名失败: activityId=%d, userId=%d, err=%v", activityID, userID, err)
		return &activity.CancelActivityResponse{Result: "fail"}, nil
	}

	// 5) 成功返回
	return &activity.CancelActivityResponse{Result: "success"}, nil
}
