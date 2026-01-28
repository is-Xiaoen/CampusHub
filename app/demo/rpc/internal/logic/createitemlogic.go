// ============================================================================
// Logic 层 - CreateItem 业务逻辑
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

// CreateItemLogic 创建资源的业务逻辑
type CreateItemLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

// NewCreateItemLogic 创建 Logic 实例
func NewCreateItemLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateItemLogic {
	return &CreateItemLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// CreateItem 创建资源
// 实现 DemoService.CreateItem RPC 方法
func (l *CreateItemLogic) CreateItem(in *pb.CreateItemRequest) (*pb.CreateItemResponse, error) {
	// ==================== 1. 参数校验 ====================
	if in.Name == "" {
		return nil, errorx.ErrInvalidParamsWithMsg("name 不能为空")
	}
	if len(in.Name) > 100 {
		return nil, errorx.ErrInvalidParamsWithMsg("name 长度不能超过 100")
	}

	// ==================== 2. 构造实体 ====================
	item := &model.Item{
		// ID 使用雪花算法生成，这里简化使用数据库自增
		// 实际项目中建议使用 common/utils/snowflake 包生成
		Name:        in.Name,
		Description: in.Description,
		Status:      1, // 默认状态：正常
	}

	// ==================== 3. 保存到数据库 ====================
	err := l.svcCtx.ItemModel.Create(l.ctx, item)
	if err != nil {
		l.Errorf("创建 item 失败: name=%s, err=%v", in.Name, err)
		return nil, errorx.ErrDBError(err)
	}

	// ==================== 4. 返回结果 ====================
	l.Infof("创建 item 成功: id=%d, name=%s", item.ID, item.Name)
	return &pb.CreateItemResponse{
		Id: item.ID,
	}, nil
}
