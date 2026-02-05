package admin

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

type AdminListActivityLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 管理员活动列表
func NewAdminListActivityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminListActivityLogic {
	return &AdminListActivityLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminListActivityLogic) AdminListActivity(req *types.ListActivityReq) (resp *types.ListActivityResp, err error) {
	// 1. 获取管理员用户 ID
	// 注意：此接口受 AdminAuth 中间件保护，已验证管理员身份
	adminID := ctxdata.GetUserIDFromCtx(l.ctx)
	if adminID <= 0 {
		return nil, errorx.ErrUnauthorized()
	}

	// 2. 分页参数校验
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100 // 管理员可查更多
	}

	// 3. 状态参数处理
	// -1 = 公开状态（2,3,4）
	// -2 = 全部状态（管理员可见所有）
	// 具体数值 = 筛选特定状态
	status := req.Status
	if status == 0 {
		status = -2 // 管理员默认查看全部
	}

	// 4. 调用 RPC 查询活动列表
	rpcResp, err := l.svcCtx.ActivityRpc.ListActivities(l.ctx, &activityservice.ListActivitiesReq{
		Page:       page,
		PageSize:   pageSize,
		CategoryId: req.CategoryId,
		Status:     status,
		Sort:       req.Sort,
		ViewerId:   adminID,
		IsAdmin:    true, // 关键：管理员标识，可查看所有状态
	})
	if err != nil {
		l.Errorf("RPC ListActivities (admin) failed: adminID=%d, err=%v", adminID, err)
		return nil, errorx.FromError(err)
	}

	// 5. 转换响应类型
	return &types.ListActivityResp{
		List:       logic.ConvertRpcActivityListItemsToApi(rpcResp.List),
		Pagination: logic.ConvertRpcPaginationToApi(rpcResp.Pagination),
	}, nil
}
