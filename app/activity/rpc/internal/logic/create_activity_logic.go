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

type CreateActivityLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateActivityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateActivityLogic {
	return &CreateActivityLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// CreateActivity 创建活动
func (l *CreateActivityLogic) CreateActivity(in *activity.CreateActivityReq) (*activity.CreateActivityResp, error) {
	// 1. 参数校验
	if err := l.validateParams(in); err != nil {
		return nil, err
	}

	// 2. 验证分类是否存在且启用
	_, err := l.svcCtx.CategoryModel.FindByID(l.ctx, uint64(in.CategoryId))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorx.New(errorx.CodeCategoryNotFound)
		}
		l.Errorf("查询分类失败: %v", err)
		return nil, errorx.ErrDBError(err)
	}

	// 3. 确定初始状态：草稿 or 已发布
	// MVP 版本：没有后台管理，审批自动通过，直接发布
	status := model.StatusDraft
	if !in.IsDraft {
		status = model.StatusPublished // MVP: 跳过待审核，直接发布
	}

	// 4. 构建活动对象
	activityData := &model.Activity{
		Title:                in.Title,
		CoverURL:             in.CoverUrl,
		CoverType:            int8(in.CoverType),
		Description:          in.Content,
		CategoryID:           uint64(in.CategoryId),
		OrganizerID:          uint64(in.OrganizerId),
		OrganizerName:        in.OrganizerName,
		OrganizerAvatar:      in.OrganizerAvatar,
		ContactPhone:         in.ContactPhone,
		RegisterStartTime:    in.RegisterStartTime,
		RegisterEndTime:      in.RegisterEndTime,
		ActivityStartTime:    in.ActivityStartTime,
		ActivityEndTime:      in.ActivityEndTime,
		Location:             in.Location,
		AddressDetail:        in.AddressDetail,
		Longitude:            in.Longitude,
		Latitude:             in.Latitude,
		MaxParticipants:      uint32(in.MaxParticipants),
		RequireApproval:      in.RequireApproval,
		RequireStudentVerify: in.RequireStudentVerify,
		MinCreditScore:       int(in.MinCreditScore),
		Status:               status,
	}

	// 5. 事务：创建活动 + 绑定标签
	err = l.svcCtx.DB.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
		// 5.1 创建活动
		if err := tx.Create(activityData).Error; err != nil {
			return err
		}

		// 5.2 绑定标签（如果有）
		if len(in.TagIds) > 0 {
			if err := l.bindTags(tx, activityData.ID, in.TagIds); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		l.Errorf("创建活动失败: %v", err)
		return nil, errorx.ErrDBError(err)
	}

	l.Infof("活动创建成功: id=%d, title=%s, status=%d", activityData.ID, activityData.Title, activityData.Status)

	// 6. 异步同步到 ES（仅发布状态需要同步）
	if activityData.Status == model.StatusPublished && l.svcCtx.SyncService != nil {
		l.svcCtx.SyncService.IndexActivityAsync(activityData)
	}

	// 7. 返回结果
	return &activity.CreateActivityResp{
		Id:     int64(activityData.ID),
		Status: int32(activityData.Status),
	}, nil
}

// validateParams 参数校验
func (l *CreateActivityLogic) validateParams(in *activity.CreateActivityReq) error {
	// 1. 标题校验
	titleLen := len([]rune(in.Title)) // 使用 rune 计算中文字符长度
	if titleLen < 2 || titleLen > 100 {
		return errorx.ErrInvalidParams("标题长度需在2-100字符之间")
	}

	// 2. 封面校验
	if in.CoverUrl == "" {
		return errorx.ErrInvalidParams("请上传活动封面")
	}
	if in.CoverType != 1 && in.CoverType != 2 {
		return errorx.ErrInvalidParams("封面类型无效")
	}

	// 3. 分类校验
	if in.CategoryId <= 0 {
		return errorx.ErrInvalidParams("请选择活动分类")
	}

	// 4. 地点校验
	if in.Location == "" {
		return errorx.ErrInvalidParams("请填写活动地点")
	}

	// 5. 组织者信息校验
	if in.OrganizerId <= 0 {
		return errorx.ErrInvalidParams("组织者信息缺失")
	}

	// 6. 时间逻辑校验
	now := time.Now().Unix()

	if in.RegisterStartTime <= now {
		return errorx.NewWithMessage(errorx.CodeActivityTimeInvalid, "报名开始时间必须在当前时间之后")
	}
	if in.RegisterEndTime <= in.RegisterStartTime {
		return errorx.NewWithMessage(errorx.CodeActivityTimeInvalid, "报名截止时间必须在报名开始时间之后")
	}
	if in.ActivityStartTime <= in.RegisterEndTime {
		return errorx.NewWithMessage(errorx.CodeActivityTimeInvalid, "活动开始时间必须在报名截止时间之后")
	}
	if in.ActivityEndTime <= in.ActivityStartTime {
		return errorx.NewWithMessage(errorx.CodeActivityTimeInvalid, "活动结束时间必须在活动开始时间之后")
	}

	// 7. 标签数量校验（最多5个）
	if len(in.TagIds) > 5 {
		return errorx.New(errorx.CodeTagLimitExceeded)
	}

	// 8. 人数上限校验
	if in.MaxParticipants < 0 {
		return errorx.ErrInvalidParams("人数上限不能为负数")
	}

	// 9. 信用分校验
	if in.MinCreditScore < 0 || in.MinCreditScore > 100 {
		return errorx.ErrInvalidParams("信用分要求需在0-100之间")
	}

	return nil
}

// bindTags 绑定标签到活动（在事务内执行）
func (l *CreateActivityLogic) bindTags(tx *gorm.DB, activityID uint64, tagIds []int64) error {
	// 1. 去重并过滤无效ID
	seen := make(map[int64]bool)
	uniqueIds := make([]uint64, 0, len(tagIds))
	for _, id := range tagIds {
		if id > 0 && !seen[id] {
			seen[id] = true
			uniqueIds = append(uniqueIds, uint64(id))
		}
	}

	if len(uniqueIds) == 0 {
		return nil
	}

	// 2. 限制最多5个
	if len(uniqueIds) > 5 {
		uniqueIds = uniqueIds[:5]
	}

	// 3. 验证标签是否存在（从 tag_cache 查询）
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
	if len(uniqueIds) == 0 {
		return nil
	}

	// 4. 调用 ActivityTagModel 绑定标签（操作 activity_tags 关联表）
	if err := l.svcCtx.ActivityTagModel.BindToActivity(l.ctx, tx, activityID, uniqueIds); err != nil {
		l.Errorf("绑定标签失败: activityID=%d, err=%v", activityID, err)
		return err
	}

	// 5. 更新活动维度的标签使用统计（activity_tag_stats 表）
	if err := l.svcCtx.TagStatsModel.BatchIncrActivityCount(l.ctx, tx, uniqueIds); err != nil {
		l.Errorf("更新标签统计失败: %v", err)
		// 不影响主流程，记录日志即可（统计数据可后续修复）
	}

	return nil
}
