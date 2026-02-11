package logic

import (
	"context"
	"errors"
	"time"

	"activity-platform/app/activity/model"
	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/activity/rpc/internal/dtm"
	"activity-platform/app/activity/rpc/internal/svc"
	userpb "activity-platform/app/user/rpc/pb/pb"
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
//
// 使用 DTM SAGA 保证跨服务数据一致性：
// - 分支1：创建活动（Activity 服务）
// - 分支2：增加标签使用计数（User 服务）
// 任一步骤失败，自动执行补偿操作回滚
func (l *CreateActivityLogic) CreateActivity(in *activity.CreateActivityReq) (*activity.CreateActivityResp, error) {
	// 1. 参数校验
	if err := l.validateParams(in); err != nil {
		return nil, err
	}

	// 2. 校验发布资格（信用分）—— 仅非草稿模式需要校验
	if !in.IsDraft {
		canPublishResp, err := l.svcCtx.CreditRpc.CanPublish(l.ctx, &userpb.CanPublishReq{
			UserId: in.OrganizerId,
		})
		if err != nil {
			l.Errorf("[CreateActivity] 信用分校验失败: userID=%d, err=%v", in.OrganizerId, err)
			return nil, errorx.NewWithMessage(errorx.CodeInternalError, "信用分校验服务异常，请稍后重试")
		}
		if !canPublishResp.Allowed {
			l.Infof("[CreateActivity] 信用分不足，禁止发布: userID=%d, score=%d, reason=%s",
				in.OrganizerId, canPublishResp.Score, canPublishResp.Reason)
			return nil, errorx.NewWithMessage(errorx.CodeCreditCannotPublish, canPublishResp.Reason)
		}
		l.Infof("[CreateActivity] 信用分校验通过: userID=%d, score=%d, level=%d",
			in.OrganizerId, canPublishResp.Score, canPublishResp.Level)
	}

	// 3. 获取组织者信息（昵称、头像）—— 非关键路径，失败不阻塞创建
	if in.OrganizerName == "" {
		userInfoResp, err := l.svcCtx.UserBasicRpc.GetUserInfo(l.ctx, &userpb.GetUserInfoReq{
			UserId: in.OrganizerId,
		})
		if err != nil {
			l.Errorf("[CreateActivity] 获取用户信息失败（不影响创建）: userID=%d, err=%v", in.OrganizerId, err)
		} else if userInfoResp.UserInfo != nil {
			in.OrganizerName = userInfoResp.UserInfo.Nickname
			in.OrganizerAvatar = userInfoResp.UserInfo.AvatarUrl
		}
	}

	// 4. 验证分类是否存在且启用
	_, err := l.svcCtx.CategoryModel.FindByID(l.ctx, uint64(in.CategoryId))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorx.New(errorx.CodeCategoryNotFound)
		}
		l.Errorf("查询分类失败: %v", err)
		return nil, errorx.ErrDBError(err)
	}

	// 5. 确定初始状态：草稿 or 已发布
	status := model.StatusDraft
	if !in.IsDraft {
		status = model.StatusPublished // MVP: 跳过待审核，直接发布
	}

	// 6. 检查 DTM 是否可用
	if l.svcCtx.DTMClient != nil && l.svcCtx.DTMClient.IsHealthy() {
		// 使用 DTM SAGA 创建活动
		return l.createActivityWithDTM(in, int32(status))
	}

	// DTM 不可用，使用本地事务（降级模式）
	l.Infof("[CreateActivity] DTM 不可用，使用本地事务")
	return l.createActivityLocal(in, int8(status))
}

// resolveCoverURL 通过 SysImage 服务解析封面图片 URL
func (l *CreateActivityLogic) resolveCoverURL(organizerID, coverImageID int64) (string, error) {
	resp, err := l.svcCtx.UserBasicRpc.GetSysImage(l.ctx, &userpb.GetSysImageReq{
		UserId:  organizerID,
		ImageId: coverImageID,
	})
	if err != nil {
		l.Errorf("[CreateActivity] 获取封面图片信息失败: organizerId=%d, imageId=%d, err=%v",
			organizerID, coverImageID, err)
		return "", errorx.NewWithMessage(errorx.CodeInternalError, "获取封面图片信息失败")
	}
	if resp.Url == "" {
		return "", errorx.ErrInvalidParams("封面图片不存在或已失效")
	}
	return resp.Url, nil
}

// createActivityWithDTM 使用 DTM SAGA 创建活动
func (l *CreateActivityLogic) createActivityWithDTM(in *activity.CreateActivityReq, status int32) (*activity.CreateActivityResp, error) {
	// 1. 去重并过滤无效标签 ID
	validTagIDs := l.filterValidTagIDs(in.TagIds)

	// 2. 解析封面图片 URL（写入时解析，读取时直接使用）
	coverURL, err := l.resolveCoverURL(in.OrganizerId, in.CoverImageId)
	if err != nil {
		return nil, err
	}

	// 3. 构建 Activity 分支请求
	activityReq := &activity.CreateActivityActionReq{
		Title:                in.Title,
		CoverUrl:             coverURL,
		CoverImageId:         in.CoverImageId,
		CoverType:            in.CoverType,
		Content:              in.Content,
		CategoryId:           in.CategoryId,
		ContactPhone:         in.ContactPhone,
		RegisterStartTime:    in.RegisterStartTime,
		RegisterEndTime:      in.RegisterEndTime,
		ActivityStartTime:    in.ActivityStartTime,
		ActivityEndTime:      in.ActivityEndTime,
		Location:             in.Location,
		AddressDetail:        in.AddressDetail,
		Longitude:            in.Longitude,
		Latitude:             in.Latitude,
		MaxParticipants:      in.MaxParticipants,
		RequireApproval:      in.RequireApproval,
		RequireStudentVerify: in.RequireStudentVerify,
		MinCreditScore:       in.MinCreditScore,
		TagIds:               validTagIDs,
		IsDraft:              in.IsDraft,
		Status:               status,
		OrganizerId:          in.OrganizerId,
		OrganizerName:        in.OrganizerName,
		OrganizerAvatar:      in.OrganizerAvatar,
	}

	// 4. 构建 User 标签计数请求（如果有标签）
	var tagReq *userpb.TagUsageCountReq
	hasTags := len(validTagIDs) > 0
	if hasTags {
		tagReq = &userpb.TagUsageCountReq{
			TagIds: validTagIDs,
			Delta:  1, // 增加 1
		}
	}

	// 5. 记录 SAGA 发起前的时间戳，用于后续精准查询创建的活动
	// 原理：activity.created_at >= beforeCreate，缩小查询范围避免同名活动误匹配
	beforeCreate := time.Now().Unix()

	// 6. 发起 SAGA 事务
	cfg := l.svcCtx.Config.DTM
	gid, err := l.svcCtx.DTMClient.CreateActivitySaga(l.ctx, dtm.CreateActivitySagaReq{
		ActivityRpcURL: cfg.ActivityRpcURL,
		ActivityReq:    activityReq,
		UserRpcURL:     cfg.UserRpcURL,
		TagReq:         tagReq,
		HasTags:        hasTags,
	})
	if err != nil {
		l.Errorf("[CreateActivity] DTM SAGA 事务失败: gid=%s, err=%v", gid, err)
		return nil, errorx.NewWithMessage(errorx.CodeInternalError, "创建活动失败，请稍后重试")
	}

	l.Infof("[CreateActivity] DTM SAGA 事务成功: gid=%s", gid)

	// 6. 查询创建的活动 ID（从数据库获取最新数据）
	// WaitResult=true 保证事务已完成，活动已写入 DB
	// 使用 organizer_id + title + created_at 时间窗口精准定位，防止同名活动误匹配
	var createdActivity model.Activity
	err = l.svcCtx.DB.WithContext(l.ctx).
		Where("organizer_id = ? AND title = ? AND created_at >= ?",
			in.OrganizerId, in.Title, beforeCreate).
		Order("id DESC").
		First(&createdActivity).Error
	if err != nil {
		l.Errorf("[CreateActivity] 查询创建的活动失败: gid=%s, err=%v", gid, err)
		return nil, errorx.ErrDBError(err)
	}

	// 6. 异步同步到 ES（仅发布状态需要同步）
	if createdActivity.Status == model.StatusPublished && l.svcCtx.SyncService != nil {
		l.svcCtx.SyncService.IndexActivityAsync(&createdActivity)
	}

	// 7. 异步发布活动创建事件（仅已发布状态，草稿不需要通知）
	if createdActivity.Status == model.StatusPublished && l.svcCtx.MsgProducer != nil {
		l.svcCtx.MsgProducer.PublishActivityCreated(
			l.ctx, createdActivity.ID, uint64(in.OrganizerId), in.Title,
		)
	}

	return &activity.CreateActivityResp{
		Id:     int64(createdActivity.ID),
		Status: int32(createdActivity.Status),
	}, nil
}

// createActivityLocal 使用本地事务创建活动（DTM 不可用时的降级方案）
func (l *CreateActivityLogic) createActivityLocal(in *activity.CreateActivityReq, status int8) (*activity.CreateActivityResp, error) {
	// 解析封面图片 URL
	coverURL, err := l.resolveCoverURL(in.OrganizerId, in.CoverImageId)
	if err != nil {
		return nil, err
	}

	// 构建活动对象
	activityData := &model.Activity{
		Title:                in.Title,
		CoverURL:             coverURL,
		CoverImageID:         in.CoverImageId,
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

	// 事务：创建活动 + 绑定标签
	err = l.svcCtx.DB.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
		// 创建活动
		if err := tx.Create(activityData).Error; err != nil {
			return err
		}

		// 绑定标签（如果有）
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

	l.Infof("活动创建成功（本地事务）: id=%d, title=%s, status=%d", activityData.ID, activityData.Title, activityData.Status)

	// 异步同步到 ES（仅发布状态需要同步）
	if activityData.Status == model.StatusPublished && l.svcCtx.SyncService != nil {
		l.svcCtx.SyncService.IndexActivityAsync(activityData)
	}

	// 异步发布活动创建事件（仅已发布状态，草稿不需要通知）
	if activityData.Status == model.StatusPublished && l.svcCtx.MsgProducer != nil {
		l.svcCtx.MsgProducer.PublishActivityCreated(
			l.ctx, activityData.ID, uint64(in.OrganizerId), in.Title,
		)
	}

	return &activity.CreateActivityResp{
		Id:     int64(activityData.ID),
		Status: int32(activityData.Status),
	}, nil
}

// filterValidTagIDs 去重并过滤无效的标签 ID
func (l *CreateActivityLogic) filterValidTagIDs(tagIds []int64) []int64 {
	seen := make(map[int64]bool)
	result := make([]int64, 0, len(tagIds))
	for _, id := range tagIds {
		if id > 0 && !seen[id] {
			seen[id] = true
			result = append(result, id)
		}
	}
	// 限制最多 5 个
	if len(result) > 5 {
		result = result[:5]
	}
	return result
}

// validateParams 参数校验
func (l *CreateActivityLogic) validateParams(in *activity.CreateActivityReq) error {
	// 1. 标题校验
	titleLen := len([]rune(in.Title)) // 使用 rune 计算中文字符长度
	if titleLen < 2 || titleLen > 100 {
		return errorx.ErrInvalidParams("标题长度需在2-100字符之间")
	}

	// 2. 封面校验
	if in.CoverImageId <= 0 {
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
