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

type GetActivityBasicLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetActivityBasicLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetActivityBasicLogic {
	return &GetActivityBasicLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// ==================== 内部接口（供其他微服务调用）====================

// GetActivityBasic 获取活动基本信息
//
// 业务逻辑：
//  1. 参数校验（活动 ID 必须大于 0）
//  2. 查询活动基本信息
//  3. 查询分类名称
//  4. 构建精简响应（不含富文本等大字段）
//
// 设计说明：
//   - 内部接口，供 Registration 等模块调用
//   - 返回字段精简，只包含报名/票券展示所需信息
//   - 不做权限校验（由调用方负责）
//   - 不返回敏感信息（如联系电话）
func (l *GetActivityBasicLogic) GetActivityBasic(in *activity.GetActivityBasicReq) (*activity.GetActivityBasicResp, error) {
	// 1. 参数校验
	if in.GetId() <= 0 {
		return nil, errorx.ErrInvalidParams("活动ID无效")
	}

	// 2. 查询活动基本信息
	activityData, err := l.svcCtx.ActivityModel.FindByID(l.ctx, uint64(in.GetId()))
	if err != nil {
		if errors.Is(err, model.ErrActivityNotFound) {
			return nil, errorx.New(errorx.CodeActivityNotFound)
		}
		l.Errorf("查询活动失败: id=%d, err=%v", in.GetId(), err)
		return nil, errorx.ErrDBError(err)
	}

	// 3. 查询分类名称
	var categoryName string
	category, err := l.svcCtx.CategoryModel.FindByID(l.ctx, activityData.CategoryID)
	if err != nil {
		// 分类可能被删除或禁用，记录日志但不影响主流程
		l.Infof("[WARNING] 查询分类失败: category_id=%d, err=%v", activityData.CategoryID, err)
		categoryName = "未知分类"
	} else {
		categoryName = category.Name
	}

	// 4. 构建响应
	l.Debugf("获取活动基本信息成功: id=%d, title=%s", activityData.ID, activityData.Title)

	return &activity.GetActivityBasicResp{
		Id:                  int64(activityData.ID),
		Title:               activityData.Title,
		Status:              int32(activityData.Status),
		CoverUrl:            activityData.CoverURL,
		CoverType:           int32(activityData.CoverType),
		Location:            activityData.Location,
		ActivityStartTime:   activityData.ActivityStartTime,
		ActivityEndTime:     activityData.ActivityEndTime,
		MaxParticipants:     int32(activityData.MaxParticipants),
		CurrentParticipants: int32(activityData.CurrentParticipants),
		OrganizerId:         int64(activityData.OrganizerID),
		OrganizerName:       activityData.OrganizerName,
		OrganizerAvatar:     activityData.OrganizerAvatar,
		CategoryName:        categoryName,
	}, nil
}
