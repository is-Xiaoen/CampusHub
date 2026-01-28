// ============================================================================
// Logic 层 - GetItem 业务逻辑
// ============================================================================
//
// 文件说明：
//   Logic 层是业务逻辑的核心实现层，负责：
//   - 参数校验
//   - 业务规则处理
//   - 调用 Model 层获取数据
//   - 错误处理和转换
//
// 设计原则：
//   1. 一个 RPC 方法对应一个 Logic 文件
//   2. Logic 不直接操作数据库，通过 Model 层
//   3. 返回业务错误码，而非底层错误
//
// 命名规范：
//   - 文件名：{方法名小写}logic.go，如 getitemlogic.go
//   - 结构体：{方法名}Logic，如 GetItemLogic
//
// ============================================================================

package logic

import (
	"context"

	"activity-platform/app/demo/rpc/internal/model"
	"activity-platform/app/demo/rpc/internal/svc"
	"activity-platform/app/demo/rpc/pb"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

// GetItemLogic 获取单个资源的业务逻辑
type GetItemLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger // 嵌入日志器，可直接使用 l.Info(), l.Error() 等
}

// NewGetItemLogic 创建 Logic 实例
// 每次 RPC 调用都会创建新的 Logic 实例
func NewGetItemLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetItemLogic {
	return &GetItemLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx), // 带上下文的日志器，会自动记录 TraceID
	}
}

// GetItem 获取单个资源
// 实现 DemoService.GetItem RPC 方法
func (l *GetItemLogic) GetItem(in *pb.GetItemRequest) (*pb.GetItemResponse, error) {
	// ==================== 1. 参数校验 ====================
	if in.Id <= 0 {
		return nil, errorx.ErrInvalidParamsWithMsg("id 必须大于 0")
	}

	// ==================== 2. 业务逻辑 ====================
	// 调用 Model 层查询数据
	item, err := l.svcCtx.ItemModel.FindByID(l.ctx, in.Id)
	if err != nil {
		// 区分"未找到"和"数据库错误"
		if err == gorm.ErrRecordNotFound {
			return nil, errorx.ErrNotFound()
		}
		// 记录错误日志（包含 TraceID，便于排查）
		l.Errorf("查询 item 失败: id=%d, err=%v", in.Id, err)
		return nil, errorx.ErrDBError(err)
	}

	// ==================== 3. 构造响应 ====================
	return &pb.GetItemResponse{
		Item: convertItemToPb(item),
	}, nil
}

// ============================================================================
// 私有辅助函数
// ============================================================================

// convertItemToPb 将 Model 实体转换为 Proto 消息
// 说明：Model 和 Proto 是独立的，需要显式转换
// 原因：
//   1. 解耦数据层和接口层
//   2. Proto 字段可能与数据库字段不完全一致
//   3. 可以在转换时做数据脱敏、格式化等
func convertItemToPb(item *model.Item) *pb.Item {
	if item == nil {
		return nil
	}
	return &pb.Item{
		Id:          item.ID,
		Name:        item.Name,
		Description: item.Description,
		Status:      item.Status,
		CreatedAt:   item.CreatedAt.Unix(),
		UpdatedAt:   item.UpdatedAt.Unix(),
	}
}
