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

type ListActivityLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 活动列表（公开接口）
func NewListActivityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListActivityLogic {
	return &ListActivityLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListActivityLogic) ListActivity(req *types.ListActivityReq) (resp *types.ListActivityResp, err error) {
	// 1. 参数校验
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	if req.PageSize > 50 {
		req.PageSize = 50
	}

	// 2. 公开接口状态参数校验
	// 公开接口只允许：-1（公开状态）或具体的公开状态值（2,3,4）
	status := req.Status
	if status != -1 && !(status >= 2 && status <= 4) {
		// 明确拒绝非法状态值，而不是静默覆盖
		return nil, errorx.ErrInvalidParams("公开接口只能查询已发布、进行中、已结束的活动")
	}

	// 3. 调用 RPC 服务
	rpcResp, err := l.svcCtx.ActivityRpc.ListActivities(l.ctx, &activityservice.ListActivitiesReq{
		Page:       req.Page,
		PageSize:   req.PageSize,
		CategoryId: req.CategoryId,
		Status:     status,
		Sort:       req.Sort,
		ViewerId:   0, // 公开接口不需要登录
		IsAdmin:    false,
	})
	if err != nil {
		l.Errorf("RPC ListActivities failed: %v", err)
		return nil, errorx.FromError(err)
	}

	// 4. 转换响应类型
	return &types.ListActivityResp{
		List:       logic.ConvertRpcActivityListItemsToApi(rpcResp.List),
		Pagination: logic.ConvertRpcPaginationToApi(rpcResp.Pagination),
	}, nil
}
