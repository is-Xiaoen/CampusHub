package logic

import (
	"context"

	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/activity/rpc/internal/svc"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

// ==================== ListTagsLogic 获取标签列表 ====================
//
// 功能说明：
//   - 获取活动服务可用的标签列表
//   - 数据来源：tag_cache 表（从用户服务同步）
//   - 支持限制返回数量（热门标签场景）
//
// 缓存策略：
//   - MVP 阶段：直接查数据库
//   - 后续优化：添加 Redis 缓存（TTL 5 分钟）
//
// 调用方：
//   - Activity API（活动创建/编辑页面选择标签）
//   - 首页（热门标签展示）

type ListTagsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListTagsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListTagsLogic {
	return &ListTagsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// ListTags 获取标签列表
//
// 请求参数：
//   - limit: 返回数量限制，0 表示返回全部，>0 表示返回热门标签
//
// 返回值：
//   - list: 标签列表（按热度排序或按 ID 排序）
//
// 错误码：
//   - 无特殊错误码，数据库错误返回通用错误
func (l *ListTagsLogic) ListTags(in *activity.ListTagsReq) (*activity.ListTagsResp, error) {
	l.Logger.Infof("ListTags 请求: limit=%d", in.Limit)

	var tags []tagItem
	var err error

	// 根据 limit 参数决定查询方式
	if in.Limit > 0 {
		// 查询热门标签（按活动使用次数排序）
		tags, err = l.getHotTags(int(in.Limit))
	} else {
		// 查询全部启用的标签
		tags, err = l.getAllTags()
	}

	if err != nil {
		l.Logger.Errorf("查询标签失败: %v", err)
		return nil, errorx.ErrDBError(err)
	}

	// 转换为 Proto 格式
	list := make([]*activity.Tag, len(tags))
	for i, tag := range tags {
		list[i] = &activity.Tag{
			Id:    int64(tag.ID),
			Name:  tag.Name,
			Color: tag.Color,
			Icon:  tag.Icon,
		}
	}

	l.Logger.Infof("ListTags 成功: 返回 %d 个标签", len(list))
	return &activity.ListTagsResp{
		List: list,
	}, nil
}

// tagItem 内部使用的标签结构
type tagItem struct {
	ID    uint64
	Name  string
	Color string
	Icon  string
}

// getAllTags 获取全部启用的标签
func (l *ListTagsLogic) getAllTags() ([]tagItem, error) {
	// 从 tag_cache 表查询所有启用的标签
	tagCaches, err := l.svcCtx.TagCacheModel.FindAll(l.ctx)
	if err != nil {
		return nil, err
	}

	// 转换为内部结构
	tags := make([]tagItem, len(tagCaches))
	for i, tc := range tagCaches {
		tags[i] = tagItem{
			ID:    tc.ID,
			Name:  tc.Name,
			Color: tc.Color,
			Icon:  tc.Icon,
		}
	}

	return tags, nil
}

// getHotTags 获取热门标签（按活动使用次数排序）
func (l *ListTagsLogic) getHotTags(limit int) ([]tagItem, error) {
	// 从 tag_cache 表查询热门标签（已关联 activity_tag_stats 排序）
	tagCaches, err := l.svcCtx.TagCacheModel.FindHot(l.ctx, limit)
	if err != nil {
		return nil, err
	}

	// 转换为内部结构
	tags := make([]tagItem, len(tagCaches))
	for i, tc := range tagCaches {
		tags[i] = tagItem{
			ID:    tc.ID,
			Name:  tc.Name,
			Color: tc.Color,
			Icon:  tc.Icon,
		}
	}

	return tags, nil
}
