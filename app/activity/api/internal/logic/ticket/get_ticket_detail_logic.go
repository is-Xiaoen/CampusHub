// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package ticket

import (
	"context"

	"activity-platform/app/activity/api/internal/svc"
	"activity-platform/app/activity/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetTicketDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取票券详情
func NewGetTicketDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTicketDetailLogic {
	return &GetTicketDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetTicketDetailLogic) GetTicketDetail(req *types.GetTicketDetailRequest) (resp *types.GetTicketDetailResponse, err error) {
	// todo: add your logic here and delete this line

	return
}
