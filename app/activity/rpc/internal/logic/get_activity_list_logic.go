package logic

import (
	"context"
	"errors"
	"strings"
	"time"

	"activity-platform/app/activity/model"
	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/activity/rpc/internal/svc"
	"activity-platform/common/ctxdata"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetActivityListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetActivityListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetActivityListLogic {
	return &GetActivityListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetActivityList 获取待参加/已参加活动信息列表
func (l *GetActivityListLogic) GetActivityList(in *activity.GetActivityListRequest) (*activity.GetActivityListResponse, error) {
	userID := ctxdata.GetUserIDFromCtx(l.ctx)
	if userID <= 0 {
		return nil, errors.New("用户未登录")
	}

	attendStatus, err := parseAttendStatus(in.GetType())
	if err != nil {
		return nil, err
	}

	pagination := model.Pagination{
		Page:     int(in.GetPage()),
		PageSize: int(in.GetPageSize()),
	}
	pagination.Normalize()

	total, err := l.svcCtx.ActivityRegistrationModel.CountByUserAttendStatus(
		l.ctx, uint64(userID), attendStatus,
	)
	if err != nil {
		return nil, err
	}

	if total == 0 {
		return &activity.GetActivityListResponse{
			Total:    0,
			Items:    []*activity.ActivityListItems{},
			Page:     int32(pagination.Page),
			PageSize: int32(pagination.PageSize),
		}, nil
	}

	registrations, err := l.svcCtx.ActivityRegistrationModel.ListByUserAttendStatus(
		l.ctx, uint64(userID), attendStatus, pagination.Offset(), pagination.PageSize,
	)
	if err != nil {
		return nil, err
	}
	if len(registrations) == 0 {
		return &activity.GetActivityListResponse{
			Total:    int32(total),
			Items:    []*activity.ActivityListItems{},
			Page:     int32(pagination.Page),
			PageSize: int32(pagination.PageSize),
		}, nil
	}

	activityIDs := make([]uint64, 0, len(registrations))
	for _, reg := range registrations {
		activityIDs = append(activityIDs, reg.ActivityID)
	}

	activities, err := l.svcCtx.ActivityModel.FindByIDs(l.ctx, activityIDs)
	if err != nil {
		return nil, err
	}
	activityMap := make(map[uint64]*model.Activity, len(activities))
	for i := range activities {
		act := activities[i]
		activityMap[act.ID] = &act
	}

	now := time.Now().Unix()
	items := make([]*activity.ActivityListItems, 0, len(registrations))
	for _, reg := range registrations {
		act := activityMap[reg.ActivityID]
		item := &activity.ActivityListItems{
			Id:       int64(reg.ActivityID),
			Name:     "",
			Time:     "",
			Status:   "未知",
			ImageUrl: "",
		}
		if act == nil {
			l.Infof("[WARNING] 活动不存在: activityId=%d, userId=%d", reg.ActivityID, userID)
			items = append(items, item)
			continue
		}

		item.Name = act.Title
		if act.ActivityStartTime > 0 {
			item.Time = time.Unix(act.ActivityStartTime, 0).Format("2006-01-02 15:04:05")
		}
		item.Status = buildActivityStatus(act, now)
		item.ImageUrl = act.CoverURL
		items = append(items, item)
	}

	return &activity.GetActivityListResponse{
		Total:    int32(total),
		Items:    items,
		Page:     int32(pagination.Page),
		PageSize: int32(pagination.PageSize),
	}, nil
}

func parseAttendStatus(value string) (int, error) {
	switch strings.TrimSpace(value) {
	case "待参加", "pending", "not_joined":
		return int(model.AttendStatusNotJoined), nil
	case "已参加", "joined":
		return int(model.AttendStatusJoined), nil
	default:
		return 0, errors.New("无效的类型")
	}
}

func buildActivityStatus(act *model.Activity, now int64) string {
	if act == nil {
		return "未知"
	}
	switch act.Status {
	case model.StatusDraft:
		return "草稿"
	case model.StatusPending:
		return "待审核"
	case model.StatusCancelled:
		return "已取消"
	case model.StatusRejected:
		return "已拒绝"
	case model.StatusFinished:
		return "已结束"
	case model.StatusOngoing:
		return "进行中"
	case model.StatusPublished:
		if act.ActivityStartTime > 0 && now < act.ActivityStartTime {
			return "待开始"
		}
		if act.ActivityEndTime > 0 && now > act.ActivityEndTime {
			return "已结束"
		}
		return "进行中"
	default:
		if text, ok := model.StatusText[act.Status]; ok {
			return text
		}
		return "未知"
	}
}
