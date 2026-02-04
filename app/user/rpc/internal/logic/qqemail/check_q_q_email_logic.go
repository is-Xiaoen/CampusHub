package qqemaillogic

import (
	"context"

	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type CheckQQEmailLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCheckQQEmailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CheckQQEmailLogic {
	return &CheckQQEmailLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CheckQQEmailLogic) CheckQQEmail(in *pb.CheckQQEmailReq) (*pb.CheckQQEmailResponse, error) {
	// todo: add your logic here and delete this line

	return &pb.CheckQQEmailResponse{}, nil
}
