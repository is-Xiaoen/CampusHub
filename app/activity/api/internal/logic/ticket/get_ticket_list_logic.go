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

type GetTicketListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取个人票券列表
func NewGetTicketListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTicketListLogic {
	return &GetTicketListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetTicketListLogic) GetTicketList(req *types.GetTicketListRequest) (resp *types.GetTicketListResponse, err error) {
	// 1. 获取当前用户 ID
	userID := ctxdata.GetUserIDFromCtx(l.ctx)
	if userID <= 0 {
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
	if pageSize > 50 {
		pageSize = 50
	}

	// 3. 调用 RPC 服务
	rpcResp, err := l.svcCtx.ActivityRpc.GetTicketList(l.ctx, &activityservice.GetTicketListRequest{
		Page:     page,
		PageSize: pageSize,
		UserId:   userID,
	})
	if err != nil {
		l.Errorf("RPC GetTicketList failed: userID=%d, err=%v", userID, err)
		return nil, errorx.FromError(err)
	}

	// 4. 转换响应
	items := make([]types.TicketListItem, 0, len(rpcResp.Items))
	for _, item := range rpcResp.Items {
		if item == nil {
			continue
		}
		items = append(items, types.TicketListItem{
			TicketId:         item.TicketId,
			ActivityId:       item.ActivityId,
			ActivityName:     item.ActivityName,
			ActivityTime:     item.ActivityTime,
			ActivityImageUrl: item.ActivityImageUrl,
			Status:           item.Status,
		})
	}

	return &types.GetTicketListResponse{
		Total:    rpcResp.Total,
		Items:    items,
		Page:     rpcResp.Page,
		PageSize: rpcResp.PageSize,
	}, nil
}
