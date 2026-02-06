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

type RegisterActivityLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 报名活动
func NewRegisterActivityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterActivityLogic {
	return &RegisterActivityLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RegisterActivityLogic) RegisterActivity(req *types.RegisterActivityRequest) (resp *types.RegisterActivityResponse, err error) {
	// 1. 获取当前用户 ID
	userID := ctxdata.GetUserIDFromCtx(l.ctx)
	if userID <= 0 {
		return nil, errorx.ErrUnauthorized()
	}

	// 2. 参数校验
	if req.ActivityId <= 0 {
		return nil, errorx.ErrInvalidParams(errMsgActivityIDInvalid)
	}

	// 3. 调用 RPC 服务
	rpcResp, err := l.svcCtx.ActivityRpc.RegisterActivity(l.ctx, &activityservice.RegisterActivityRequest{
		ActivityId: req.ActivityId,
		UserId:     userID,
	})
	if err != nil {
		l.Errorf("RPC RegisterActivity failed: activityId=%d, userID=%d, err=%v", req.ActivityId, userID, err)
		return nil, errorx.FromError(err)
	}

	// 4. 返回响应
	return &types.RegisterActivityResponse{
		Result: rpcResp.Result,
		Reason: rpcResp.Reason,
	}, nil
}
