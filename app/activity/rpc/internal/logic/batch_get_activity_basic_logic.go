package logic

import (
	"context"

	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/activity/rpc/internal/svc"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type BatchGetActivityBasicLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewBatchGetActivityBasicLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchGetActivityBasicLogic {
	return &BatchGetActivityBasicLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 批量获取活动基本信息
//
// 业务逻辑：
//  1. 参数校验（IDs 不能为空，单次最多 100 个）
//  2. 批量查询活动信息
//  3. 批量查询分类名称
//  4. 构建响应（保持输入顺序）
//
// 设计说明：
//   - 内部接口，供 Registration/Ticket 等模块调用
//   - 使用 FindByIDs 批量查询，避免 N+1 问题
//   - 不存在的 ID 将被跳过（不报错）
//   - 限制单次最多 100 个，防止查询过慢
func (l *BatchGetActivityBasicLogic) BatchGetActivityBasic(in *activity.BatchGetActivityBasicReq) (*activity.BatchGetActivityBasicResp, error) {
	// 1. 参数校验
	ids := in.GetIds()
	if len(ids) == 0 {
		return &activity.BatchGetActivityBasicResp{
			Activities: []*activity.GetActivityBasicResp{},
		}, nil
	}

	// 单次最多 100 个
	if len(ids) > 100 {
		return nil, errorx.ErrInvalidParams("单次最多查询 100 个活动")
	}

	// 2. 转换为 uint64
	activityIDs := make([]uint64, len(ids))
	for i, id := range ids {
		if id <= 0 {
			return nil, errorx.ErrInvalidParams("活动ID无效")
		}
		activityIDs[i] = uint64(id)
	}

	// 3. 批量查询活动
	activities, err := l.svcCtx.ActivityModel.FindByIDs(l.ctx, activityIDs)
	if err != nil {
		l.Errorf("批量查询活动失败: err=%v", err)
		return nil, errorx.ErrDBError(err)
	}

	// 空结果直接返回
	if len(activities) == 0 {
		return &activity.BatchGetActivityBasicResp{
			Activities: []*activity.GetActivityBasicResp{},
		}, nil
	}

	// 4. 收集分类 ID 并批量查询
	categoryIDSet := make(map[uint64]bool)
	for _, act := range activities {
		categoryIDSet[act.CategoryID] = true
	}

	// 加载分类映射表
	categoryMap := l.loadCategoryMap()

	// 5. 构建 ID -> Activity 映射
	activityMap := make(map[uint64]*activity.GetActivityBasicResp, len(activities))
	for _, act := range activities {
		categoryName := categoryMap[act.CategoryID]
		if categoryName == "" {
			categoryName = "未知分类"
		}

		activityMap[act.ID] = &activity.GetActivityBasicResp{
			Id:                  int64(act.ID),
			Title:               act.Title,
			Status:              int32(act.Status),
			CoverUrl:            act.CoverURL,
			CoverType:           int32(act.CoverType),
			Location:            act.Location,
			ActivityStartTime:   act.ActivityStartTime,
			ActivityEndTime:     act.ActivityEndTime,
			MaxParticipants:     int32(act.MaxParticipants),
			CurrentParticipants: int32(act.CurrentParticipants),
			OrganizerId:         int64(act.OrganizerID),
			OrganizerName:       act.OrganizerName,
			OrganizerAvatar:     act.OrganizerAvatar,
			CategoryName:        categoryName,
		}
	}

	// 6. 按输入顺序构建结果（跳过不存在的 ID）
	result := make([]*activity.GetActivityBasicResp, 0, len(ids))
	for _, id := range ids {
		if act, ok := activityMap[uint64(id)]; ok {
			result = append(result, act)
		}
	}

	l.Infof("批量获取活动基本信息成功: requested=%d, returned=%d", len(ids), len(result))

	return &activity.BatchGetActivityBasicResp{
		Activities: result,
	}, nil
}

// loadCategoryMap 加载分类映射表
func (l *BatchGetActivityBasicLogic) loadCategoryMap() map[uint64]string {
	categoryMap := make(map[uint64]string)

	categories, err := l.svcCtx.CategoryModel.FindAll(l.ctx)
	if err != nil {
		l.Infof("[WARNING] 加载分类列表失败: %v", err)
		return categoryMap
	}

	for _, cat := range categories {
		categoryMap[cat.ID] = cat.Name
	}
	return categoryMap
}
