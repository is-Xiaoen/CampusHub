// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package ticket

import (
	"context"

	"activity-platform/app/activity/api/internal/svc"
	"activity-platform/app/activity/api/internal/types"
	"activity-platform/app/activity/rpc/activityservice"
	"activity-platform/common/ctxdata"
	"activity-platform/common/errorx"

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
	// 1. 获取当前用户 ID
	userID := ctxdata.GetUserIDFromCtx(l.ctx)
	if userID <= 0 {
		return nil, errorx.ErrUnauthorized()
	}

	// 2. 参数校验
	if req.TicketId <= 0 {
		return nil, errorx.ErrInvalidParams(errMsgTicketIDInvalid)
	}

	// 3. 调用 RPC 服务
	rpcResp, err := l.svcCtx.ActivityRpc.GetTicketDetail(l.ctx, &activityservice.GetTicketDetailRequest{
		TicketId: req.TicketId,
	})
	if err != nil {
		l.Errorf("RPC GetTicketDetail failed: ticketId=%d, userID=%d, err=%v", req.TicketId, userID, err)
		return nil, errorx.FromError(err)
	}

	// 4. 返回响应
	return &types.GetTicketDetailResponse{
		TicketId:     rpcResp.TicketId,
		TicketCode:   rpcResp.TicketCode,
		ActivityId:   rpcResp.ActivityId,
		ActivityName: rpcResp.ActivityName,
		ActivityTime: rpcResp.ActivityTime,
		QrCodeUrl:    rpcResp.QrCodeUrl,
	}, nil
}
