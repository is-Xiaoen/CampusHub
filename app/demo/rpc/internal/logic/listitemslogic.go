// ============================================================================
// Logic 层 - ListItems 业务逻辑
// ============================================================================

package logic

import (
	"context"

	"activity-platform/app/demo/rpc/internal/model"
	"activity-platform/app/demo/rpc/internal/svc"
	"activity-platform/app/demo/rpc/pb"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

// ListItemsLogic 列表查询的业务逻辑
type ListItemsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

// NewListItemsLogic 创建 Logic 实例
func NewListItemsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListItemsLogic {
	return &ListItemsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// ListItems 列表查询
// 实现 DemoService.ListItems RPC 方法
func (l *ListItemsLogic) ListItems(in *pb.ListItemsRequest) (*pb.ListItemsResponse, error) {
	// ==================== 1. 参数校验和默认值 ====================
	page := int(in.Page)
	pageSize := int(in.PageSize)

	// 设置默认值
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	// 限制最大页大小，防止一次查询过多数据
	if pageSize > 100 {
		pageSize = 100
	}

	// ==================== 2. 构造查询条件 ====================
	opt := &model.ListOption{
		Page:     page,
		PageSize: pageSize,
		Keyword:  in.Keyword,
		Status:   in.Status,
	}

	// ==================== 3. 查询数据 ====================
	items, total, err := l.svcCtx.ItemModel.List(l.ctx, opt)
	if err != nil {
		l.Errorf("查询 item 列表失败: err=%v", err)
		return nil, errorx.ErrDBError(err)
	}

	// ==================== 4. 转换为 Proto 格式 ====================
	pbItems := make([]*pb.Item, 0, len(items))
	for _, item := range items {
		pbItems = append(pbItems, &pb.Item{
			Id:          item.ID,
			Name:        item.Name,
			Description: item.Description,
			Status:      item.Status,
			CreatedAt:   item.CreatedAt.Unix(),
			UpdatedAt:   item.UpdatedAt.Unix(),
		})
	}

	return &pb.ListItemsResponse{
		List:  pbItems,
		Total: total,
	}, nil
}
