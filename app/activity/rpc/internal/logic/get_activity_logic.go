package logic

import (
	"context"
	"errors"

	"activity-platform/app/activity/model"
	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/activity/rpc/internal/svc"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetActivityLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetActivityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetActivityLogic {
	return &GetActivityLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetActivity 获取活动详情
//
// 业务逻辑：
//  1. 参数校验（ID 必须大于 0）
//  2. 查询活动基本信息（优先从缓存获取）
//  3. 权限校验（非公开状态需要是组织者或管理员）
//  4. 聚合关联数据（分类名称、标签列表）
//  5. 构建响应（手机号脱敏）
//
// 缓存策略：
//   - 活动详情使用 ActivityCache（TTL 5min）
//   - 分类名称使用 CategoryCache（TTL 30min）
//   - 缓存失效：更新/删除活动时主动删除
func (l *GetActivityLogic) GetActivity(in *activity.GetActivityReq) (*activity.GetActivityResp, error) {
	// 1. 参数校验
	if in.Id <= 0 {
		return nil, errorx.ErrInvalidParams("活动ID无效")
	}

	// 2. 查询活动基本信息（优先从缓存获取）
	var activityData *model.Activity
	var err error

	// 优先使用缓存
	if l.svcCtx.ActivityCache != nil {
		activityData, err = l.svcCtx.ActivityCache.GetByID(l.ctx, uint64(in.Id))
	} else {
		// 缓存服务未初始化，降级查 DB
		activityData, err = l.svcCtx.ActivityModel.FindByID(l.ctx, uint64(in.Id))
	}

	if err != nil {
		if errors.Is(err, model.ErrActivityNotFound) {
			return nil, errorx.New(errorx.CodeActivityNotFound)
		}
		l.Errorf("查询活动失败: id=%d, err=%v", in.Id, err)
		return nil, errorx.ErrDBError(err)
	}

	// 3. 权限校验
	// 公开状态（已发布、进行中、已结束）：任何人可见
	// 非公开状态（草稿、待审核、已拒绝、已取消）：仅组织者可见
	if !activityData.IsPublic() {
		// 非公开状态，需要检查访问者是否为组织者
		if in.ViewerId <= 0 || uint64(in.ViewerId) != activityData.OrganizerID {
			l.Infof("权限拒绝: activity_id=%d, status=%d, viewer_id=%d, organizer_id=%d",
				in.Id, activityData.Status, in.ViewerId, activityData.OrganizerID)
			return nil, errorx.New(errorx.CodeActivityPermissionDenied)
		}
	}

	// 4. 查询分类名称（优先从缓存获取）
	var categoryName string
	if l.svcCtx.CategoryCache != nil {
		category, err := l.svcCtx.CategoryCache.GetByID(l.ctx, activityData.CategoryID)
		if err != nil {
			l.Infof("[WARNING] 查询分类失败: category_id=%d, err=%v", activityData.CategoryID, err)
			categoryName = "未知分类"
		} else {
			categoryName = category.Name
		}
	} else {
		// 缓存服务未初始化，降级查 DB
		category, err := l.svcCtx.CategoryModel.FindByID(l.ctx, activityData.CategoryID)
		if err != nil {
			l.Infof("[WARNING] 查询分类失败: category_id=%d, err=%v", activityData.CategoryID, err)
			categoryName = "未知分类"
		} else {
			categoryName = category.Name
		}
	}

	// 5. 查询标签列表（从 tag_cache 表）
	tagCaches, err := l.svcCtx.TagCacheModel.FindByActivityID(l.ctx, uint64(in.Id))
	if err != nil {
		// 标签查询失败不影响主流程
		l.Infof("[WARNING] 查询标签失败: activity_id=%d, err=%v", in.Id, err)
		tagCaches = []model.TagCache{}
	}

	// 6. 构建响应
	detail := l.buildActivityDetail(activityData, categoryName, tagCaches)

	l.Infof("获取活动详情成功: id=%d, title=%s, viewer_id=%d",
		activityData.ID, activityData.Title, in.ViewerId)

	return &activity.GetActivityResp{
		Activity: detail,
	}, nil
}

// buildActivityDetail 构建活动详情响应
func (l *GetActivityLogic) buildActivityDetail(act *model.Activity, categoryName string, tags []model.TagCache) *activity.ActivityDetail {
	return &activity.ActivityDetail{
		Id:                   int64(act.ID),
		Title:                act.Title,
		CoverUrl:             act.CoverURL,
		CoverType:            int32(act.CoverType),
		Content:              act.Description,
		CategoryId:           int64(act.CategoryID),
		CategoryName:         categoryName,
		OrganizerId:          int64(act.OrganizerID),
		OrganizerName:        act.OrganizerName,
		OrganizerAvatar:      act.OrganizerAvatar,
		ContactPhone:         maskPhone(act.ContactPhone), // 手机号脱敏
		RegisterStartTime:    act.RegisterStartTime,
		RegisterEndTime:      act.RegisterEndTime,
		ActivityStartTime:    act.ActivityStartTime,
		ActivityEndTime:      act.ActivityEndTime,
		Location:             act.Location,
		AddressDetail:        act.AddressDetail,
		Longitude:            act.Longitude,
		Latitude:             act.Latitude,
		MaxParticipants:      int32(act.MaxParticipants),
		CurrentParticipants:  int32(act.CurrentParticipants),
		RequireApproval:      act.RequireApproval,
		RequireStudentVerify: act.RequireStudentVerify,
		MinCreditScore:       int32(act.MinCreditScore),
		Status:               int32(act.Status),
		StatusText:           act.StatusText(),
		RejectReason:         act.RejectReason,
		ViewCount:            int64(act.ViewCount),
		LikeCount:            int64(act.LikeCount),
		Tags:                 convertTagCaches(tags),
		CreatedAt:            act.CreatedAt,
		UpdatedAt:            act.UpdatedAt,
		Version:              int32(act.Version),
	}
}

// maskPhone 手机号脱敏
//
// 输入: "13812345678"
// 输出: "138****5678"
//
// 设计说明：
// - 保护用户隐私，防止手机号被恶意收集
// - 保留前3位和后4位，便于用户确认
// - 非标准11位手机号直接返回原值（兼容座机等情况）
func maskPhone(phone string) string {
	if len(phone) != 11 {
		return phone
	}
	return phone[:3] + "****" + phone[7:]
}

// convertTagCaches 将 model.TagCache 转换为 proto Tag
func convertTagCaches(tags []model.TagCache) []*activity.Tag {
	if len(tags) == 0 {
		return []*activity.Tag{}
	}

	result := make([]*activity.Tag, len(tags))
	for i, tag := range tags {
		result[i] = &activity.Tag{
			Id:    int64(tag.ID),
			Name:  tag.Name,
			Color: tag.Color,
			Icon:  tag.Icon,
		}
	}
	return result
}
