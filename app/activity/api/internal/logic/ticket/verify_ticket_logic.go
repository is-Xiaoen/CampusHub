// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package ticket

import (
	"context"
	"strings"

	"activity-platform/app/activity/api/internal/svc"
	"activity-platform/app/activity/api/internal/types"
	"activity-platform/app/activity/rpc/activityservice"
	"activity-platform/common/ctxdata"
	"activity-platform/common/errorx"

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
	// 1. 获取当前用户 ID
	userID := ctxdata.GetUserIDFromCtx(l.ctx)
	if userID <= 0 {
		return nil, errorx.ErrUnauthorized()
	}

	// 2. 参数校验
	if req.ActivityId <= 0 {
		return nil, errorx.ErrInvalidParams(errMsgActivityIDInvalid)
	}
	if strings.TrimSpace(req.TicketCode) == "" {
		return nil, errorx.ErrInvalidParams(errMsgTicketCodeEmpty)
	}

	// 3. 调用 RPC 服务
	rpcResp, err := l.svcCtx.ActivityRpc.VerifyTicket(l.ctx, &activityservice.VerifyTicketRequest{
		ActivityId: req.ActivityId,
		TicketCode: req.TicketCode,
		TotpCode:   req.TotpCode,
	})
	if err != nil {
		l.Errorf("RPC VerifyTicket failed: activityId=%d, userID=%d, err=%v", req.ActivityId, userID, err)
		return nil, errorx.FromError(err)
	}

	// 4. 返回响应
	return &types.VerifyTicketResponse{
		Result: rpcResp.Result,
	}, nil
}
