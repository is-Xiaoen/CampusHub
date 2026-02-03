// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package ticket

import (
	"context"

	"activity-platform/app/activity/api/internal/svc"
	"activity-platform/app/activity/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type VerifyTicketLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 核销票券
func NewVerifyTicketLogic(ctx context.Context, svcCtx *svc.ServiceContext) *VerifyTicketLogic {
	return &VerifyTicketLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *VerifyTicketLogic) VerifyTicket(req *types.VerifyTicketRequest) (resp *types.VerifyTicketResponse, err error) {
	// todo: add your logic here and delete this line

	return
}
