package public

import (
	"context"

	"activity-platform/app/activity/api/internal/logic"
	"activity-platform/app/activity/api/internal/svc"
	"activity-platform/app/activity/api/internal/types"
	"activity-platform/app/activity/rpc/activityservice"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetHotActivityLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 热门活动
func NewGetHotActivityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetHotActivityLogic {
	return &GetHotActivityLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetHotActivityLogic) GetHotActivity(req *types.GetHotActivityReq) (resp *types.GetHotActivityResp, err error) {
	// 1. 参数校验
	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 20 {
		limit = 20 // 最多返回20条热门活动
	}

	// 2. 调用 RPC 服务
	rpcResp, err := l.svcCtx.ActivityRpc.GetHotActivities(l.ctx, &activityservice.GetHotActivitiesReq{
		Limit: limit,
	})
	if err != nil {
		l.Errorf("RPC GetHotActivities failed: limit=%d, err=%v", limit, err)
		return nil, errorx.FromError(err)
	}

	// 3. 转换响应类型
	return &types.GetHotActivityResp{
		List: logic.ConvertRpcActivityListItemsToApi(rpcResp.List),
	}, nil
}
