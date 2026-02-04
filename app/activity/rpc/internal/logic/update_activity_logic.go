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

type UpdateActivityLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateActivityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateActivityLogic {
	return &UpdateActivityLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// UpdateActivity 更新活动
// 核心设计要点：
// 1. 乐观锁防止并发更新冲突
// 2. 权限校验：只有组织者能修改自己的活动
// 3. 状态校验：不同状态允许修改的字段不同
// 4. 事务保证：更新活动和标签在同一事务中
func (l *UpdateActivityLogic) UpdateActivity(in *activity.UpdateActivityReq) (*activity.UpdateActivityResp, error) {
	// 1. 基础参数校验
	if err := l.validateBasicParams(in); err != nil {
		return nil, err
	}

	// 2. 查询活动信息
	activityData, err := l.svcCtx.ActivityModel.FindByID(l.ctx, uint64(in.Id))
	if err != nil {
		if errors.Is(err, model.ErrActivityNotFound) {
			return nil, errorx.New(errorx.CodeActivityNotFound)
		}
		l.Errorf("查询活动失败: id=%d, err=%v", in.Id, err)
		return nil, errorx.ErrDBError(err)
	}

	// 3. 权限校验：只有组织者能修改自己的活动
	if activityData.OrganizerID != uint64(in.OperatorId) {
		l.Infof("[权限拒绝] 无权限修改活动: activityId=%d, organizerId=%d, operatorId=%d",
			in.Id, activityData.OrganizerID, in.OperatorId)
		return nil, errorx.New(errorx.CodeActivityPermissionDenied)
	}

	// 4. 版本号校验（乐观锁前置检查，避免不必要的处理）
	if activityData.Version != uint32(in.Version) {
		l.Infof("[版本冲突] 版本号不匹配: id=%d, currentVersion=%d, requestVersion=%d",
			in.Id, activityData.Version, in.Version)
		return nil, errorx.New(errorx.CodeActivityConcurrentUpdate)
	}

	// 5. 状态校验 + 构建更新字段
	updates, newStatus, err := l.buildUpdates(in, activityData)
	if err != nil {
		return nil, err
	}

	// 如果没有任何字段需要更新（只传了 id 和 version）
	if len(updates) == 0 && !in.UpdateTags {
		return &activity.UpdateActivityResp{
			Status:     int32(activityData.Status),
			NewVersion: int32(activityData.Version),
		}, nil
	}

	// 6. 事务：更新活动 + 更新标签
	var finalVersion uint32
	err = l.svcCtx.DB.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
		// 6.1 更新活动（带乐观锁）
		if len(updates) > 0 {
			updates["version"] = gorm.Expr("version + 1")
			if newStatus != activityData.Status {
				updates["status"] = newStatus
			}

			result := tx.Model(&model.Activity{}).
				Where("id = ? AND version = ?", in.Id, in.Version).
				Updates(updates)

			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return model.ErrActivityConcurrentUpdate
			}
			finalVersion = uint32(in.Version) + 1
		} else {
			finalVersion = uint32(in.Version)
		}

		// 6.2 更新标签（如果需要）
		if in.UpdateTags {
			if err := l.updateTags(tx, uint64(in.Id), in.TagIds, activityData); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		if errors.Is(err, model.ErrActivityConcurrentUpdate) {
			return nil, errorx.New(errorx.CodeActivityConcurrentUpdate)
		}
		l.Errorf("更新活动失败: id=%d, err=%v", in.Id, err)
		return nil, errorx.ErrDBError(err)
	}

	// 确定最终状态
	finalStatus := activityData.Status
	if newStatus != activityData.Status {
		finalStatus = newStatus
	}

	// 删除缓存（更新成功后）
	if l.svcCtx.ActivityCache != nil {
		if err := l.svcCtx.ActivityCache.Invalidate(l.ctx, uint64(in.Id)); err != nil {
			// 缓存删除失败不影响主流程，记录日志即可
			l.Infof("[WARNING] 删除活动缓存失败: id=%d, err=%v", in.Id, err)
		}
	}

	l.Infof("活动更新成功: id=%d, status=%d, newVersion=%d", in.Id, finalStatus, finalVersion)

	return &activity.UpdateActivityResp{
		Status:     int32(finalStatus),
		NewVersion: int32(finalVersion),
	}, nil
}

// validateBasicParams 基础参数校验
func (l *UpdateActivityLogic) validateBasicParams(in *activity.UpdateActivityReq) error {
	if in.Id <= 0 {
		return errorx.ErrInvalidParams("活动ID无效")
	}
	if in.Version < 0 {
		return errorx.ErrInvalidParams("版本号无效")
	}
	if in.OperatorId <= 0 {
		return errorx.ErrInvalidParams("操作者信息缺失")
	}
	return nil
}

// buildUpdates 根据状态构建更新字段
// 返回：更新字段map、新状态、错误
//
// MVP 版本说明：
// - 待审核状态(1)在MVP中不会出现（自动审批），但保留代码兼容性
// - 已拒绝状态(5)在MVP中不会出现（无后台管理），但保留代码兼容性
func (l *UpdateActivityLogic) buildUpdates(in *activity.UpdateActivityReq, activityData *model.Activity) (map[string]interface{}, int8, error) {
	updates := make(map[string]interface{})
	newStatus := activityData.Status

	switch activityData.Status {
	case model.StatusDraft:
		// 草稿：可编辑所有字段（MVP 主要使用状态）
		if err := l.buildAllFieldUpdates(in, activityData, updates); err != nil {
			return nil, 0, err
		}

	case model.StatusPending:
		// 待审核：可编辑所有字段（MVP中不会出现此状态，保留兼容性）
		if err := l.buildAllFieldUpdates(in, activityData, updates); err != nil {
			return nil, 0, err
		}

	case model.StatusRejected:
		// 已拒绝：可编辑所有字段，编辑后变为草稿状态（MVP中不会出现，保留兼容性）
		if err := l.buildAllFieldUpdates(in, activityData, updates); err != nil {
			return nil, 0, err
		}
		// 如果有任何字段更新，状态变为草稿
		if len(updates) > 0 || in.UpdateTags {
			newStatus = model.StatusDraft
			updates["reject_reason"] = "" // 清空拒绝原因
		}

	case model.StatusPublished:
		// 已发布：只能修改特定字段（MVP 主要使用状态）
		if err := l.buildPublishedUpdates(in, activityData, updates); err != nil {
			return nil, 0, err
		}

	case model.StatusOngoing, model.StatusFinished, model.StatusCancelled:
		// 进行中、已结束、已取消：不可编辑
		return nil, 0, errorx.New(errorx.CodeActivityStatusInvalid)

	default:
		return nil, 0, errorx.New(errorx.CodeActivityStatusInvalid)
	}

	return updates, newStatus, nil
}

// buildAllFieldUpdates 构建所有字段的更新（草稿、待审核、已拒绝状态）
func (l *UpdateActivityLogic) buildAllFieldUpdates(in *activity.UpdateActivityReq, activityData *model.Activity, updates map[string]interface{}) error {
	// 收集时间字段，用于后续校验
	registerStartTime := activityData.RegisterStartTime
	registerEndTime := activityData.RegisterEndTime
	activityStartTime := activityData.ActivityStartTime
	activityEndTime := activityData.ActivityEndTime

	// 标题
	if in.Title != nil {
		titleLen := len([]rune(*in.Title))
		if titleLen < 2 || titleLen > 100 {
			return errorx.ErrInvalidParams("标题长度需在2-100字符之间")
		}
		updates["title"] = *in.Title
	}

	// 封面
	if in.CoverUrl != nil {
		if *in.CoverUrl == "" {
			return errorx.ErrInvalidParams("封面URL不能为空")
		}
		updates["cover_url"] = *in.CoverUrl
	}
	if in.CoverType != nil {
		if *in.CoverType != 1 && *in.CoverType != 2 {
			return errorx.ErrInvalidParams("封面类型无效")
		}
		updates["cover_type"] = *in.CoverType
	}

	// 描述
	if in.Content != nil {
		updates["description"] = *in.Content
	}

	// 分类
	if in.CategoryId != nil {
		if *in.CategoryId <= 0 {
			return errorx.ErrInvalidParams("分类ID无效")
		}
		// 验证分类是否存在
		_, err := l.svcCtx.CategoryModel.FindByID(l.ctx, uint64(*in.CategoryId))
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errorx.New(errorx.CodeCategoryNotFound)
			}
			return errorx.ErrDBError(err)
		}
		updates["category_id"] = *in.CategoryId
	}

	// 联系电话
	if in.ContactPhone != nil {
		updates["contact_phone"] = *in.ContactPhone
	}

	// 时间字段
	if in.RegisterStartTime != nil {
		registerStartTime = *in.RegisterStartTime
		updates["register_start_time"] = *in.RegisterStartTime
	}
	if in.RegisterEndTime != nil {
		registerEndTime = *in.RegisterEndTime
		updates["register_end_time"] = *in.RegisterEndTime
	}
	if in.ActivityStartTime != nil {
		activityStartTime = *in.ActivityStartTime
		updates["activity_start_time"] = *in.ActivityStartTime
	}
	if in.ActivityEndTime != nil {
		activityEndTime = *in.ActivityEndTime
		updates["activity_end_time"] = *in.ActivityEndTime
	}

	// 时间逻辑校验（只有传入了时间字段才校验）
	if in.RegisterStartTime != nil || in.RegisterEndTime != nil ||
		in.ActivityStartTime != nil || in.ActivityEndTime != nil {
		if err := l.validateTimeLogic(registerStartTime, registerEndTime, activityStartTime, activityEndTime); err != nil {
			return err
		}
	}

	// 地点
	if in.Location != nil {
		if *in.Location == "" {
			return errorx.ErrInvalidParams("活动地点不能为空")
		}
		updates["location"] = *in.Location
	}
	if in.AddressDetail != nil {
		updates["address_detail"] = *in.AddressDetail
	}

	// 经纬度
	if in.Longitude != nil {
		if *in.Longitude < -180 || *in.Longitude > 180 {
			return errorx.ErrInvalidParams("经度范围无效")
		}
		updates["longitude"] = *in.Longitude
	}
	if in.Latitude != nil {
		if *in.Latitude < -90 || *in.Latitude > 90 {
			return errorx.ErrInvalidParams("纬度范围无效")
		}
		updates["latitude"] = *in.Latitude
	}

	// 人数上限
	if in.MaxParticipants != nil {
		if *in.MaxParticipants < 0 {
			return errorx.ErrInvalidParams("人数上限不能为负数")
		}
		updates["max_participants"] = *in.MaxParticipants
	}

	// 报名规则
	if in.RequireApproval != nil {
		updates["require_approval"] = *in.RequireApproval
	}
	if in.RequireStudentVerify != nil {
		updates["require_student_verify"] = *in.RequireStudentVerify
	}
	if in.MinCreditScore != nil {
		if *in.MinCreditScore < 0 || *in.MinCreditScore > 100 {
			return errorx.ErrInvalidParams("信用分要求需在0-100之间")
		}
		updates["min_credit_score"] = *in.MinCreditScore
	}

	return nil
}

// buildPublishedUpdates 构建已发布状态的更新（只能修改特定字段）
func (l *UpdateActivityLogic) buildPublishedUpdates(in *activity.UpdateActivityReq, activityData *model.Activity, updates map[string]interface{}) error {
	// 已发布状态只能修改：description, cover_url, cover_type

	// 检查是否尝试修改不允许的字段
	if in.Title != nil {
		return errorx.NewWithMessage(errorx.CodeActivityStatusInvalid, "已发布的活动不能修改标题")
	}
	if in.CategoryId != nil {
		return errorx.NewWithMessage(errorx.CodeActivityStatusInvalid, "已发布的活动不能修改分类")
	}
	if in.RegisterStartTime != nil || in.RegisterEndTime != nil ||
		in.ActivityStartTime != nil || in.ActivityEndTime != nil {
		return errorx.NewWithMessage(errorx.CodeActivityStatusInvalid, "已发布的活动不能修改时间")
	}
	if in.Location != nil || in.AddressDetail != nil ||
		in.Longitude != nil || in.Latitude != nil {
		return errorx.NewWithMessage(errorx.CodeActivityStatusInvalid, "已发布的活动不能修改地点")
	}
	if in.RequireApproval != nil || in.RequireStudentVerify != nil || in.MinCreditScore != nil {
		return errorx.NewWithMessage(errorx.CodeActivityStatusInvalid, "已发布的活动不能修改报名规则")
	}
	if in.UpdateTags {
		return errorx.NewWithMessage(errorx.CodeActivityStatusInvalid, "已发布的活动不能修改标签")
	}

	// 人数上限特殊处理：有报名后不能减少
	if in.MaxParticipants != nil {
		if activityData.CurrentParticipants > 0 && uint32(*in.MaxParticipants) < activityData.CurrentParticipants {
			return errorx.NewWithMessage(errorx.CodeActivityStatusInvalid,
				"有报名记录的活动不能减少人数上限到低于当前报名人数")
		}
		// 已发布状态允许增加人数上限
		if *in.MaxParticipants >= 0 {
			updates["max_participants"] = *in.MaxParticipants
		}
	}

	// 允许修改的字段
	if in.Content != nil {
		updates["description"] = *in.Content
	}
	if in.CoverUrl != nil {
		if *in.CoverUrl == "" {
			return errorx.ErrInvalidParams("封面URL不能为空")
		}
		updates["cover_url"] = *in.CoverUrl
	}
	if in.CoverType != nil {
		if *in.CoverType != 1 && *in.CoverType != 2 {
			return errorx.ErrInvalidParams("封面类型无效")
		}
		updates["cover_type"] = *in.CoverType
	}
	if in.ContactPhone != nil {
		updates["contact_phone"] = *in.ContactPhone
	}

	return nil
}

// validateTimeLogic 校验时间逻辑
func (l *UpdateActivityLogic) validateTimeLogic(registerStartTime, registerEndTime, activityStartTime, activityEndTime int64) error {
	now := time.Now().Unix()

	// 报名开始时间必须在当前时间之后（仅对新创建或修改报名开始时间时校验）
	// 注意：编辑已存在的活动时，如果报名已开始，这个校验需要特殊处理
	// 这里假设只有在修改时间时才调用此方法，所以需要校验未来时间
	if registerStartTime <= now {
		return errorx.NewWithMessage(errorx.CodeActivityTimeInvalid, "报名开始时间必须在当前时间之后")
	}
	if registerEndTime <= registerStartTime {
		return errorx.NewWithMessage(errorx.CodeActivityTimeInvalid, "报名截止时间必须在报名开始时间之后")
	}
	if activityStartTime <= registerEndTime {
		return errorx.NewWithMessage(errorx.CodeActivityTimeInvalid, "活动开始时间必须在报名截止时间之后")
	}
	if activityEndTime <= activityStartTime {
		return errorx.NewWithMessage(errorx.CodeActivityTimeInvalid, "活动结束时间必须在活动开始时间之后")
	}

	return nil
}

// updateTags 更新标签（在事务内执行）
func (l *UpdateActivityLogic) updateTags(tx *gorm.DB, activityID uint64, tagIds []int64, activityData *model.Activity) error {
	// 1. 获取旧标签ID（用于统计更新）
	oldTagIDs, err := l.svcCtx.ActivityTagModel.FindIDsByActivityID(l.ctx, activityID)
	if err != nil {
		l.Errorf("获取旧标签ID失败: activityID=%d, err=%v", activityID, err)
		// 继续执行，不影响主流程
		oldTagIDs = nil
	}

	// 2. 删除旧绑定
	if err := l.svcCtx.ActivityTagModel.UnbindFromActivity(l.ctx, tx, activityID); err != nil {
		l.Errorf("解绑旧标签失败: activityID=%d, err=%v", activityID, err)
		return err
	}

	// 3. 去重并过滤无效ID
	seen := make(map[int64]bool)
	uniqueIds := make([]uint64, 0, len(tagIds))
	for _, id := range tagIds {
		if id > 0 && !seen[id] {
			seen[id] = true
			uniqueIds = append(uniqueIds, uint64(id))
		}
	}

	// 限制最多5个
	if len(uniqueIds) > 5 {
		uniqueIds = uniqueIds[:5]
	}

	// 4. 验证标签是否存在（从 tag_cache 查询）
	if len(uniqueIds) > 0 {
		existIDs, invalidIDs, err := l.svcCtx.TagCacheModel.ExistsByIDs(l.ctx, uniqueIds)
		if err != nil {
			l.Errorf("验证标签失败: %v", err)
			return err
		}
		if len(invalidIDs) > 0 {
			l.Infof("[WARN] 部分标签不存在或已禁用: %v", invalidIDs)
			// 只绑定存在的标签，忽略不存在的
			uniqueIds = existIDs
		}
	}

	// 5. 绑定新标签
	if len(uniqueIds) > 0 {
		if err := l.svcCtx.ActivityTagModel.BindToActivity(l.ctx, tx, activityID, uniqueIds); err != nil {
			l.Errorf("绑定新标签失败: activityID=%d, err=%v", activityID, err)
			return err
		}
	}

	// 6. 更新活动标签统计（activity_tag_stats 表）
	// 先减少旧标签的 activity_count
	if len(oldTagIDs) > 0 {
		if err := l.svcCtx.TagStatsModel.BatchDecrActivityCount(l.ctx, tx, oldTagIDs); err != nil {
			l.Errorf("减少旧标签统计失败: %v", err)
			// 不影响主流程，统计数据可后续修复
		}
	}
	// 再增加新标签的 activity_count
	if len(uniqueIds) > 0 {
		if err := l.svcCtx.TagStatsModel.BatchIncrActivityCount(l.ctx, tx, uniqueIds); err != nil {
			l.Errorf("增加新标签统计失败: %v", err)
			// 不影响主流程，统计数据可后续修复
		}
	}

	return nil
}
