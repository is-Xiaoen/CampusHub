package activity

import (
	"context"

	"activity-platform/app/activity/api/internal/svc"
	"activity-platform/app/activity/api/internal/types"
	"activity-platform/app/activity/rpc/activityservice"
	"activity-platform/common/ctxdata"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type CancelActivityLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 取消活动
func NewCancelActivityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CancelActivityLogic {
	return &CancelActivityLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CancelActivityLogic) CancelActivity(req *types.CancelActivityReq) (resp *types.CancelActivityResp, err error) {
	// 1. 获取当前用户 ID
	userID := ctxdata.GetUserIDFromCtx(l.ctx)
	if userID <= 0 {
		return nil, errorx.ErrUnauthorized()
	}

	// 2. 参数校验
	if req.Id <= 0 {
		return nil, errorx.ErrInvalidParams("活动ID无效")
	}

	// 取消原因可选，但如果提供则限制长度
	if len([]rune(req.Reason)) > 500 {
		return nil, errorx.ErrInvalidParams("取消原因不能超过500字符")
	}

	// 3. 调用 RPC 取消活动
	// 状态流转：待审核(1)/已发布(2)/进行中(3) -> 已取消(6)
	// RPC 层会校验：只有组织者能取消、状态必须允许取消
	rpcResp, err := l.svcCtx.ActivityRpc.CancelActivity(l.ctx, &activityservice.CancelActivityReq{
		Id:         req.Id,
		Reason:     req.Reason,
		OperatorId: userID,
		IsAdmin:    false, // 普通用户接口，非管理员
	})
	if err != nil {
		l.Errorf("RPC CancelActivity failed: id=%d, userID=%d, err=%v", req.Id, userID, err)
		return nil, errorx.FromError(err)
	}

	// 4. 返回响应
	return &types.CancelActivityResp{
		Status: rpcResp.Status,
	}, nil
}
