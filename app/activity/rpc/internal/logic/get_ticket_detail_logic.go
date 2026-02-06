package logic

import (
	"context"
	"errors"
	"fmt"
	"time"

	"activity-platform/app/activity/model"
	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/activity/rpc/internal/svc"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetTicketDetailLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetTicketDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTicketDetailLogic {
	return &GetTicketDetailLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetTicketDetail 获取票券详情
func (l *GetTicketDetailLogic) GetTicketDetail(in *activity.GetTicketDetailRequest) (*activity.GetTicketDetailResponse, error) {
	if in.GetTicketId() <= 0 {
		return nil, errorx.ErrInvalidParams("票券ID无效")
	}

	userID := in.GetUserId()
	if userID <= 0 {
		return nil, errorx.ErrUnauthorized()
	}

	ticket, err := l.svcCtx.ActivityTicketModel.FindByID(l.ctx, uint64(in.GetTicketId()))
	if err != nil {
		return nil, err
	}
	if ticket.UserID != uint64(userID) {
		return nil, model.ErrTicketNotFound
	}

	secret := ticket.TotpSecret
	if secret == "" {
		secret = deriveTotpSecret(int64(ticket.ActivityID), int64(ticket.UserID), ticket.TicketCode)
	}
	totp, err := generateTotpCode(secret, time.Now())
	if err != nil {
		return nil, err
	}

	qrPayload := fmt.Sprintf("activity_id=%d|ticket_code=%s|totp=%s",
		ticket.ActivityID, ticket.TicketCode, totp)

	resp := &activity.GetTicketDetailResponse{
		TicketId:   int64(ticket.ID),
		TicketCode: ticket.TicketCode,
		ActivityId: int64(ticket.ActivityID),
		QrCodeUrl:  qrPayload,
	}

	activityInfo, err := l.svcCtx.ActivityModel.FindByID(l.ctx, ticket.ActivityID)
	if err != nil {
		if errors.Is(err, model.ErrActivityNotFound) {
			l.Infof("[WARNING] 活动不存在: activityId=%d, ticketId=%d", ticket.ActivityID, ticket.ID)
		} else {
			return nil, err
		}
	} else {
		resp.ActivityName = activityInfo.Title
		if activityInfo.ActivityStartTime > 0 {
			resp.ActivityTime = time.Unix(activityInfo.ActivityStartTime, 0).Format("2006-01-02 15:04:05")
		}
	}

	return resp, nil
}
