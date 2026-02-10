package logic

import (
	"context"
	"time"

	"activity-platform/app/activity/model"
	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/activity/rpc/internal/svc"
	"activity-platform/common/errorx"

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
	userID := in.GetUserId()
	if userID <= 0 {
		return nil, errorx.ErrUnauthorized()
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

	// 批量查询活动信息（避免 N+1 查询）
	activityIDSet := make(map[uint64]bool, len(tickets))
	for _, ticket := range tickets {
		activityIDSet[ticket.ActivityID] = true
	}
	activityIDs := make([]uint64, 0, len(activityIDSet))
	for id := range activityIDSet {
		activityIDs = append(activityIDs, id)
	}

	activities, err := l.svcCtx.ActivityModel.FindByIDs(l.ctx, activityIDs)
	if err != nil {
		l.Errorf("批量查询活动失败: %v", err)
		return nil, err
	}

	activityMap := make(map[uint64]*model.Activity, len(activities))
	for i := range activities {
		activityMap[activities[i].ID] = &activities[i]
	}

	items := make([]*activity.TicketListItem, 0, len(tickets))
	for _, ticket := range tickets {
		item := &activity.TicketListItem{
			TicketId:   int64(ticket.ID),
			ActivityId: int64(ticket.ActivityID),
			Status:     mapTicketStatus(ticket.Status),
		}

		activityInfo, ok := activityMap[ticket.ActivityID]
		if !ok {
			l.Infof("[WARNING] 活动不存在: activityId=%d, ticketId=%d", ticket.ActivityID, ticket.ID)
		} else {
			item.ActivityName = activityInfo.Title
			if activityInfo.ActivityStartTime > 0 {
				item.ActivityTime = time.Unix(activityInfo.ActivityStartTime, 0).Format("2006-01-02 15:04:05")
			}
			item.ActivityImageUrl = activityInfo.CoverURL
		}

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
