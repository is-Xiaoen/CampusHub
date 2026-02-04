package qqemaillogic

import (
	"context"

	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendQQEmailLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendQQEmailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendQQEmailLogic {
	return &SendQQEmailLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SendQQEmailLogic) SendQQEmail(in *pb.SendQQEmailReq) (*pb.SendQQEmailResponse, error) {
	// todo: add your logic here and delete this line

	return &pb.SendQQEmailResponse{}, nil
}
