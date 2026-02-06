// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package ticket

import (
	"context"
	"strings"

	"activity-platform/app/activity/api/internal/svc"
	"activity-platform/app/activity/api/internal/types"
	"activity-platform/app/activity/rpc/activityservice"
	"activity-platform/common/ctxdata"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetActivityListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取待参加/已参加活动列表
func NewGetActivityListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetActivityListLogic {
	return &GetActivityListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetActivityListLogic) GetActivityList(req *types.GetActivityListRequest) (resp *types.GetActivityListResponse, err error) {
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

	// 3. 类型校验
	typeValue := strings.TrimSpace(req.Type)
	if typeValue == "" {
		typeValue = "待参加"
	}
	switch typeValue {
	case "待参加", "pending", "not_joined", "已参加", "joined":
	default:
		return nil, errorx.ErrInvalidParams(errMsgTypeInvalid)
	}

	// 4. 调用 RPC 服务
	rpcResp, err := l.svcCtx.ActivityRpc.GetActivityList(l.ctx, &activityservice.GetActivityListRequest{
		Page:     page,
		PageSize: pageSize,
		Type:     typeValue,
		UserId:   userID,
	})
	if err != nil {
		l.Errorf("RPC GetActivityList failed: userID=%d, err=%v", userID, err)
		return nil, errorx.FromError(err)
	}

	// 5. 转换响应
	items := make([]types.ActivityListItems, 0, len(rpcResp.Items))
	for _, item := range rpcResp.Items {
		if item == nil {
			continue
		}
		items = append(items, types.ActivityListItems{
			Id:       item.Id,
			Name:     item.Name,
			Time:     item.Time,
			Status:   item.Status,
			ImageUrl: item.ImageUrl,
		})
	}

	return &types.GetActivityListResponse{
		Total:    rpcResp.Total,
		Items:    items,
		Page:     rpcResp.Page,
		PageSize: rpcResp.PageSize,
	}, nil
}
