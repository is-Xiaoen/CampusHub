package logic

import (
	"context"

	"activity-platform/app/activity/model"
	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/activity/rpc/internal/svc"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

// ==================== ListCategoriesLogic 获取分类列表 ====================
//
// 功能说明：
//   - 获取活动分类列表
//   - 数据来源：categories 表（活动服务自有数据）
//   - 返回所有启用的分类，按排序权重降序
//
// 缓存策略：
//   - MVP 阶段：直接查数据库
//   - 后续优化：添加 Redis 缓存（TTL 24 小时，分类变动少）
//
// 调用方：
//   - Activity API（活动创建/编辑页面选择分类）
//   - Activity API（活动列表页面分类筛选）

type ListCategoriesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListCategoriesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListCategoriesLogic {
	return &ListCategoriesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// ListCategories 获取分类列表
//
// 请求参数：
//   - 无参数
//
// 返回值：
//   - list: 分类列表（按排序权重降序，ID 升序）
//
// 错误码：
//   - 无特殊错误码，数据库错误返回通用错误
//
// 缓存策略：
//   - 使用 CategoryCache（TTL 30min）
//   - 分类数据变化少，使用较长 TTL
func (l *ListCategoriesLogic) ListCategories(in *activity.ListCategoriesReq) (*activity.ListCategoriesResp, error) {
	l.Logger.Info("ListCategories 请求")

	// 查询所有启用的分类（优先从缓存获取）
	var categories []model.Category
	var err error

	if l.svcCtx.CategoryCache != nil {
		categories, err = l.svcCtx.CategoryCache.GetList(l.ctx)
	} else {
		// 缓存服务未初始化，降级查 DB
		categories, err = l.svcCtx.CategoryModel.FindAll(l.ctx)
	}

	if err != nil {
		l.Logger.Errorf("查询分类失败: %v", err)
		return nil, errorx.ErrDBError(err)
	}

	// 转换为 Proto 格式
	list := make([]*activity.Category, len(categories))
	for i, cat := range categories {
		list[i] = &activity.Category{
			Id:   int64(cat.ID),
			Name: cat.Name,
			Icon: cat.Icon,
			Sort: int32(cat.Sort),
		}
	}

	l.Logger.Infof("ListCategories 成功: 返回 %d 个分类", len(list))
	return &activity.ListCategoriesResp{
		List: list,
	}, nil
}
