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

type DeleteActivityLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 删除活动
func NewDeleteActivityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteActivityLogic {
	return &DeleteActivityLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteActivityLogic) DeleteActivity(req *types.DeleteActivityReq) (resp *types.DeleteActivityResp, err error) {
	// 1. 获取当前用户 ID
	userID := ctxdata.GetUserIDFromCtx(l.ctx)
	if userID <= 0 {
		return nil, errorx.ErrUnauthorized()
	}

	// 2. 参数校验
	if req.Id <= 0 {
		return nil, errorx.ErrInvalidParams("活动ID无效")
	}

	// 3. 调用 RPC 删除活动
	// 普通用户只能删除自己创建的活动，RPC 层会校验权限
	rpcResp, err := l.svcCtx.ActivityRpc.DeleteActivity(l.ctx, &activityservice.DeleteActivityReq{
		Id:         req.Id,
		OperatorId: userID,
		IsAdmin:    false, // 普通用户接口，非管理员
	})
	if err != nil {
		l.Errorf("RPC DeleteActivity failed: id=%d, userID=%d, err=%v", req.Id, userID, err)
		return nil, errorx.FromError(err)
	}

	// 4. 返回响应
	return &types.DeleteActivityResp{
		Success: rpcResp.Success,
	}, nil
}
