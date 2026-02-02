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

type GetActivityLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 活动详情（公开接口）
func NewGetActivityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetActivityLogic {
	return &GetActivityLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetActivityLogic) GetActivity(req *types.GetActivityReq) (resp *types.GetActivityResp, err error) {
	// 1. 参数校验
	if req.Id <= 0 {
		return nil, errorx.ErrInvalidParams("活动ID无效")
	}

	// 2. 调用 RPC 服务
	// 公开接口 viewer_id = 0，RPC 层会根据活动状态判断是否可见
	rpcResp, err := l.svcCtx.ActivityRpc.GetActivity(l.ctx, &activityservice.GetActivityReq{
		Id:       req.Id,
		ViewerId: 0, // 公开接口不需要登录，由 RPC 层判断权限
	})
	if err != nil {
		l.Errorf("RPC GetActivity failed: id=%d, err=%v", req.Id, err)
		return nil, errorx.FromError(err)
	}

	// 3. 转换响应类型
	return &types.GetActivityResp{
		Activity: logic.ConvertRpcActivityDetailToApi(rpcResp.Activity),
	}, nil
}
