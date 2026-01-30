/**
 * @projectName: CampusHub
 * @package: verifyservice
 * @className: VerifyServiceServer
 * @author: lijunqi
 * @description: 学生认证服务Server层实现，负责路由转发到Logic层
 * @date: 2026-01-30
 * @version: 1.0
 */

package verifyservice

import (
	"context"

	verifyservicelogic "activity-platform/app/user/rpc/internal/logic/verifyservice"
	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"
)

// VerifyServiceServer 学生认证服务Server
// 实现 pb.VerifyServiceServer 接口
type VerifyServiceServer struct {
	svcCtx *svc.ServiceContext
	pb.UnimplementedVerifyServiceServer
}

// NewVerifyServiceServer 创建学生认证服务Server实例
func NewVerifyServiceServer(svcCtx *svc.ServiceContext) *VerifyServiceServer {
	return &VerifyServiceServer{
		svcCtx: svcCtx,
	}
}

// IsVerified 查询用户是否已完成学生认证
// 调用方: Activity服务（报名/发布活动前校验）
func (s *VerifyServiceServer) IsVerified(ctx context.Context, in *pb.IsVerifiedReq) (*pb.IsVerifiedResp, error) {
	l := verifyservicelogic.NewIsVerifiedLogic(ctx, s.svcCtx)
	return l.IsVerified(in)
}

// UpdateVerifyStatus 更新认证状态
// 调用方: MQ Consumer（OCR回调、人工审核结果）
func (s *VerifyServiceServer) UpdateVerifyStatus(ctx context.Context, in *pb.UpdateVerifyStatusReq) (*pb.UpdateVerifyStatusResp, error) {
	l := verifyservicelogic.NewUpdateVerifyStatusLogic(ctx, s.svcCtx)
	return l.UpdateVerifyStatus(in)
}
