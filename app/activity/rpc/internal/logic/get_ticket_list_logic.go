package logic

import (
	"context"
	"errors"

	"activity-platform/app/activity/model"
	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/activity/rpc/internal/svc"
	"activity-platform/common/ctxdata"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetTicketListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetTicketListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTicketListLogic {
	return &GetTicketListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetTicketList 获取个人票券列表
func (l *GetTicketListLogic) GetTicketList(in *activity.GetTicketListRequest) (*activity.GetTicketListResponse, error) {
	userID := ctxdata.GetUserIDFromCtx(l.ctx)
	if userID <= 0 {
		return nil, errors.New("用户未登录")
	}

	pagination := model.Pagination{
		Page:     int(in.GetPage()),
		PageSize: int(in.GetPageSize()),
	}
	pagination.Normalize()

	total, err := l.svcCtx.ActivityTicketModel.CountByUserID(l.ctx, uint64(userID))
	if err != nil {
		return nil, err
	}

	if total == 0 {
		return &activity.GetTicketListResponse{
			Total:    0,
			Items:    []*activity.TicketListItem{},
			Page:     int32(pagination.Page),
			PageSize: int32(pagination.PageSize),
		}, nil
	}

	tickets, err := l.svcCtx.ActivityTicketModel.ListByUserID(
		l.ctx, uint64(userID), pagination.Offset(), pagination.PageSize,
	)
	if err != nil {
		return nil, err
	}

	items := make([]*activity.TicketListItem, 0, len(tickets))
	for _, ticket := range tickets {
		item := &activity.TicketListItem{
			TicketId:   int64(ticket.ID),
			ActivityId: int64(ticket.ActivityID),
			Status:     mapTicketStatus(ticket.Status),
		}

		// TODO: 调用同事的方法，通过 activity_id 获取活动名称、时间、封面图
		// activityInfo := l.svcCtx.ActivityModel.???
		// item.ActivityName = activityInfo.Name
		// item.ActivityTime = activityInfo.Time
		// item.ActivityImageUrl = activityInfo.ImageUrl

		items = append(items, item)
	}

	return &activity.GetTicketListResponse{
		Total:    int32(total),
		Items:    items,
		Page:     int32(pagination.Page),
		PageSize: int32(pagination.PageSize),
	}, nil
}

func mapTicketStatus(status int8) int32 {
	return int32(status)
}
