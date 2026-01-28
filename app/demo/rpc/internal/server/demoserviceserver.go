// ============================================================================
// Server 层 - gRPC 服务实现
// ============================================================================
//
// 文件说明：
//   Server 层是 gRPC 服务的入口，负责：
//   - 实现 Proto 定义的服务接口
//   - 创建 Logic 实例并调用
//   - 是 Proto 定义和 Logic 实现之间的桥梁
//
// 设计原则：
//   1. Server 层尽量薄，只做请求转发
//   2. 不要在 Server 层写业务逻辑
//   3. 所有业务逻辑都在 Logic 层实现
//
// 代码生成：
//   实际项目中，这个文件可以通过 goctl 自动生成：
//   goctl rpc protoc demo.proto --go_out=. --go-grpc_out=. --zrpc_out=.
//
// ============================================================================

package server

import (
	"context"

	"activity-platform/app/demo/rpc/internal/logic"
	"activity-platform/app/demo/rpc/internal/svc"
	"activity-platform/app/demo/rpc/pb"
)

// DemoServiceServer gRPC 服务实现
// 实现 pb.DemoServiceServer 接口
type DemoServiceServer struct {
	svcCtx *svc.ServiceContext
	pb.UnimplementedDemoServiceServer // 嵌入未实现的服务器（gRPC 要求）
}

// NewDemoServiceServer 创建服务实例
func NewDemoServiceServer(svcCtx *svc.ServiceContext) *DemoServiceServer {
	return &DemoServiceServer{
		svcCtx: svcCtx,
	}
}

// ============================================================================
// 接口实现
// ============================================================================

// GetItem 获取单个资源
func (s *DemoServiceServer) GetItem(ctx context.Context, in *pb.GetItemRequest) (*pb.GetItemResponse, error) {
	l := logic.NewGetItemLogic(ctx, s.svcCtx)
	return l.GetItem(in)
}

// ListItems 获取资源列表
func (s *DemoServiceServer) ListItems(ctx context.Context, in *pb.ListItemsRequest) (*pb.ListItemsResponse, error) {
	l := logic.NewListItemsLogic(ctx, s.svcCtx)
	return l.ListItems(in)
}

// CreateItem 创建资源
func (s *DemoServiceServer) CreateItem(ctx context.Context, in *pb.CreateItemRequest) (*pb.CreateItemResponse, error) {
	l := logic.NewCreateItemLogic(ctx, s.svcCtx)
	return l.CreateItem(in)
}

// UpdateItem 更新资源
func (s *DemoServiceServer) UpdateItem(ctx context.Context, in *pb.UpdateItemRequest) (*pb.UpdateItemResponse, error) {
	// TODO: 实现更新逻辑
	// l := logic.NewUpdateItemLogic(ctx, s.svcCtx)
	// return l.UpdateItem(in)
	return &pb.UpdateItemResponse{Success: true}, nil
}

// DeleteItem 删除资源
func (s *DemoServiceServer) DeleteItem(ctx context.Context, in *pb.DeleteItemRequest) (*pb.DeleteItemResponse, error) {
	// TODO: 实现删除逻辑
	// l := logic.NewDeleteItemLogic(ctx, s.svcCtx)
	// return l.DeleteItem(in)
	return &pb.DeleteItemResponse{Success: true}, nil
}
