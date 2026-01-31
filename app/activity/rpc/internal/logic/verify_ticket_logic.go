package logic

import (
	"context"

	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/activity/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type VerifyTicketLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewVerifyTicketLogic(ctx context.Context, svcCtx *svc.ServiceContext) *VerifyTicketLogic {
	return &VerifyTicketLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// VerifyTicket 核销票券
func (l *VerifyTicketLogic) VerifyTicket(in *activity.VerifyTicketRequest) (*activity.VerifyTicketResponse, error) {
	// todo: add your logic here and delete this line

	return &activity.VerifyTicketResponse{}, nil
}
