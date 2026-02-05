package activity

import (
	"context"

	"activity-platform/app/activity/api/internal/logic"
	"activity-platform/app/activity/api/internal/svc"
	"activity-platform/app/activity/api/internal/types"
	"activity-platform/app/activity/rpc/activityservice"
	"activity-platform/common/ctxdata"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type MyCreatedActivityLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 我创建的活动
func NewMyCreatedActivityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MyCreatedActivityLogic {
	return &MyCreatedActivityLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *MyCreatedActivityLogic) MyCreatedActivity(req *types.MyActivityReq) (resp *types.MyActivityResp, err error) {
	// 1. 获取当前用户 ID
	userID := ctxdata.GetUserIDFromCtx(l.ctx)
	if userID <= 0 {
		return nil, errorx.ErrUnauthorized()
	}

	// 2. 分页参数校验和默认值处理
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 50 {
		pageSize = 50 // 最大每页 50 条
	}

	// 3. 调用 RPC 查询我创建的活动
	// status = -2 表示查询全部状态（包括草稿、待审核等）
	rpcResp, err := l.svcCtx.ActivityRpc.ListActivities(l.ctx, &activityservice.ListActivitiesReq{
		Page:        page,
		PageSize:    pageSize,
		OrganizerId: userID, // 关键：按组织者 ID 筛选
		Status:      -2,     // 全部状态
		Sort:        "created_at",
		ViewerId:    userID,
		IsAdmin:     false,
	})
	if err != nil {
		l.Errorf("RPC ListActivities failed: userID=%d, err=%v", userID, err)
		return nil, errorx.FromError(err)
	}

	// 4. 转换响应类型
	return &types.MyActivityResp{
		List:       logic.ConvertRpcActivityListItemsToApi(rpcResp.List),
		Pagination: logic.ConvertRpcPaginationToApi(rpcResp.Pagination),
	}, nil
}
