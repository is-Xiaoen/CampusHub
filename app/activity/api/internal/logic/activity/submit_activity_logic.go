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

type SubmitActivityLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 提交审核
func NewSubmitActivityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SubmitActivityLogic {
	return &SubmitActivityLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SubmitActivityLogic) SubmitActivity(req *types.SubmitActivityReq) (resp *types.SubmitActivityResp, err error) {
	// 1. 获取当前用户 ID
	userID := ctxdata.GetUserIDFromCtx(l.ctx)
	if userID <= 0 {
		return nil, errorx.ErrUnauthorized()
	}

	// 2. 参数校验
	if req.Id <= 0 {
		return nil, errorx.ErrInvalidParams("活动ID无效")
	}

	// 3. 调用 RPC 提交审核
	// 状态流转：草稿(0) -> 待审核(1)
	// RPC 层会校验：只有组织者能提交、只有草稿状态能提交
	rpcResp, err := l.svcCtx.ActivityRpc.SubmitActivity(l.ctx, &activityservice.SubmitActivityReq{
		Id:         req.Id,
		OperatorId: userID,
	})
	if err != nil {
		l.Errorf("RPC SubmitActivity failed: id=%d, userID=%d, err=%v", req.Id, userID, err)
		return nil, errorx.FromError(err)
	}

	// 4. 返回响应
	return &types.SubmitActivityResp{
		Status: rpcResp.Status,
	}, nil
}
