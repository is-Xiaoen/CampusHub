/**
 * @projectName: CampusHub
 * @package: creditservice
 * @className: CreditServiceServer
 * @author: lijunqi
 * @description: 信用分服务Server层实现，负责路由转发到Logic层
 * @date: 2026-01-30
 * @version: 1.0
 */

package creditservice

import (
	"context"

	creditservicelogic "activity-platform/app/user/rpc/internal/logic/creditservice"
	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"
)

// CreditServiceServer 信用分服务Server
// 实现 pb.CreditServiceServer 接口
type CreditServiceServer struct {
	svcCtx *svc.ServiceContext
	pb.UnimplementedCreditServiceServer
}

// NewCreditServiceServer 创建信用分服务Server实例
func NewCreditServiceServer(svcCtx *svc.ServiceContext) *CreditServiceServer {
	return &CreditServiceServer{
		svcCtx: svcCtx,
	}
}

// GetCreditInfo 获取用户信用信息
// 调用方: User API（个人中心-信用信息页面）
func (s *CreditServiceServer) GetCreditInfo(ctx context.Context, in *pb.GetCreditInfoReq) (*pb.GetCreditInfoResp, error) {
	l := creditservicelogic.NewGetCreditInfoLogic(ctx, s.svcCtx)
	return l.GetCreditInfo(in)
}

// CanParticipate 校验是否允许报名
// 调用方: Activity服务（报名模块）
func (s *CreditServiceServer) CanParticipate(ctx context.Context, in *pb.CanParticipateReq) (*pb.CanParticipateResp, error) {
	l := creditservicelogic.NewCanParticipateLogic(ctx, s.svcCtx)
	return l.CanParticipate(in)
}

// CanPublish 校验是否允许发布活动
// 调用方: Activity服务（发布模块）
func (s *CreditServiceServer) CanPublish(ctx context.Context, in *pb.CanPublishReq) (*pb.CanPublishResp, error) {
	l := creditservicelogic.NewCanPublishLogic(ctx, s.svcCtx)
	return l.CanPublish(in)
}

// InitCredit 初始化信用分
// 调用方: User服务（注册流程）
func (s *CreditServiceServer) InitCredit(ctx context.Context, in *pb.InitCreditReq) (*pb.InitCreditResp, error) {
	l := creditservicelogic.NewInitCreditLogic(ctx, s.svcCtx)
	return l.InitCredit(in)
}

// UpdateScore 变更信用分
// 调用方: MQ Consumer（内部系统）
func (s *CreditServiceServer) UpdateScore(ctx context.Context, in *pb.UpdateScoreReq) (*pb.UpdateScoreResp, error) {
	l := creditservicelogic.NewUpdateScoreLogic(ctx, s.svcCtx)
	return l.UpdateScore(in)
}
