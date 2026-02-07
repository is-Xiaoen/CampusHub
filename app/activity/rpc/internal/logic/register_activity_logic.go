package logic

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"activity-platform/app/activity/model"
	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/activity/rpc/internal/svc"
	"activity-platform/app/user/rpc/client/creditservice"
	"activity-platform/app/user/rpc/client/verifyservice"

	"github.com/zeromicro/go-zero/core/breaker"
	"github.com/zeromicro/go-zero/core/logx"
)

type RegisterActivityLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

// NewRegisterActivityLogic 创建报名逻辑
func NewRegisterActivityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterActivityLogic {
	return &RegisterActivityLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// RegisterActivity 报名活动
func (l *RegisterActivityLogic) RegisterActivity(in *activity.RegisterActivityRequest) (resp *activity.RegisterActivityResponse, err error) {
	if in == nil {
		return &activity.RegisterActivityResponse{
			Result: "fail",
			Reason: "参数错误",
		}, nil
	}
	userID := in.GetUserId()
	activityID := in.GetActivityId()
	defer func() {
		if r := recover(); r != nil {
			l.Errorf("报名逻辑panic: userId=%d, activityId=%d, err=%v", userID, activityID, r)
			resp = &activity.RegisterActivityResponse{
				Result: "fail",
				Reason: "报名失败，请稍后重试",
			}
			err = nil
		}
	}()
	if userID <= 0 || activityID <= 0 {
		return &activity.RegisterActivityResponse{
			Result: "fail",
			Reason: "参数错误",
		}, nil
	}

	// ==================== 第一步：限流检查 ====================
	if l.svcCtx.RegistrationLimiter != nil && !l.svcCtx.RegistrationLimiter.AllowCtx(l.ctx) {
		return &activity.RegisterActivityResponse{
			Result: "fail",
			Reason: "请求过于频繁，请稍后再试",
		}, nil
	}

	// ==================== 活动校验 ====================
	activityData, err := l.svcCtx.ActivityModel.FindByID(l.ctx, uint64(activityID))
	if err != nil {
		if errors.Is(err, model.ErrActivityNotFound) {
			return &activity.RegisterActivityResponse{
				Result: "fail",
				Reason: "活动不存在",
			}, nil
		}
		l.Errorf("活动查询失败: activityId=%d, err=%v", activityID, err)
		return &activity.RegisterActivityResponse{
			Result: "fail",
			Reason: "活动查询失败，请稍后重试",
		}, nil
	}
	now := time.Now().Unix()
	if activityData.Status != model.StatusPublished {
		return &activity.RegisterActivityResponse{
			Result: "fail",
			Reason: "活动不在报名中",
		}, nil
	}
	if activityData.RegisterStartTime > 0 && now < activityData.RegisterStartTime {
		return &activity.RegisterActivityResponse{
			Result: "fail",
			Reason: "报名未开始",
		}, nil
	}
	if activityData.RegisterEndTime > 0 && now > activityData.RegisterEndTime {
		return &activity.RegisterActivityResponse{
			Result: "fail",
			Reason: "报名已结束",
		}, nil
	}
	if activityData.MaxParticipants > 0 && activityData.CurrentParticipants >= activityData.MaxParticipants {
		return &activity.RegisterActivityResponse{
			Result: "fail",
			Reason: "活动名额已满",
		}, nil
	}

	// ==================== 信誉校验 ====================
	creditResp, err := l.svcCtx.CreditRpc.CanParticipate(l.ctx, &creditservice.CanParticipateReq{
		UserId: userID,
	})
	if err != nil {
		l.Errorf("信誉校验失败: userId=%d, err=%v", userID, err)
		return &activity.RegisterActivityResponse{
			Result: "fail",
			Reason: "信誉校验失败，请稍后重试",
		}, nil
	}
	if creditResp == nil {
		l.Errorf("信誉校验返回空响应: userId=%d", userID)
		return &activity.RegisterActivityResponse{
			Result: "fail",
			Reason: "信誉校验失败，请稍后重试",
		}, nil
	}
	if !creditResp.GetAllowed() {
		_ = l.svcCtx.ActivityRegistrationModel.Create(l.ctx, &model.ActivityRegistration{
			ActivityID: uint64(activityID),
			UserID:     uint64(userID),
			Status:     model.RegistrationStatusFailed,
		})
		return &activity.RegisterActivityResponse{
			Result: "fail",
			Reason: creditResp.GetReason(),
		}, nil
	}

	// ==================== 实名校验 ====================
	if activityData.RequireStudentVerify {
		verifyResp, err := l.svcCtx.VerifyService.IsVerified(l.ctx, &verifyservice.IsVerifiedReq{
			UserId: userID,
		})
		if err != nil {
			l.Errorf("实名校验失败: userId=%d, err=%v", userID, err)
			return &activity.RegisterActivityResponse{
				Result: "fail",
				Reason: "实名校验失败，请稍后重试",
			}, nil
		}
		if verifyResp == nil {
			l.Errorf("实名校验返回空响应: userId=%d", userID)
			return &activity.RegisterActivityResponse{
				Result: "fail",
				Reason: "实名校验失败，请稍后重试",
			}, nil
		}
		if !verifyResp.GetIsVerified() {
			_ = l.svcCtx.ActivityRegistrationModel.Create(l.ctx, &model.ActivityRegistration{
				ActivityID: uint64(activityID),
				UserID:     uint64(userID),
				Status:     model.RegistrationStatusFailed,
			})
			return &activity.RegisterActivityResponse{
				Result: "fail",
				Reason: "请先完成学生认证",
			}, nil
		}
	}

	// ==================== 第二步：熔断保护 ====================
	alreadyRegistered := false
	genTicketPayload := func() (*model.TicketPayload, error) {
		ticketCode, err := generateTicketCode()
		if err != nil {
			return nil, err
		}
		return &model.TicketPayload{
			TicketCode: ticketCode,
			TicketUUID: buildTicketQrPayload(activityID, ticketCode),
			TotpSecret: deriveTotpSecret(activityID, userID, ticketCode),
		}, nil
	}
	registerFn := func() error {
		registered, err := l.registerWithConsistency(activityID, userID, genTicketPayload)
		if err != nil {
			return err
		}
		alreadyRegistered = registered
		return nil
	}
	if l.svcCtx.RegistrationBreaker != nil {
		err = l.svcCtx.RegistrationBreaker.DoWithFallbackAcceptable(
			registerFn,
			func(err error) error {
				return breaker.ErrServiceUnavailable
			},
			func(err error) bool {
				if err == nil {
					return true
				}
				return errors.Is(err, model.ErrActivityQuotaFull)
			},
		)
	} else {
		err = registerFn()
	}
	if err != nil {
		if errors.Is(err, model.ErrActivityQuotaFull) {
			return &activity.RegisterActivityResponse{
				Result: "fail",
				Reason: "活动名额已满",
			}, nil
		}
		l.Errorf("报名记录写入失败: userId=%d, activityId=%d, err=%v", userID, in.GetActivityId(), err)
		return &activity.RegisterActivityResponse{
			Result: "fail",
			Reason: "报名失败，请稍后重试",
		}, nil
	}

	if alreadyRegistered {
		return &activity.RegisterActivityResponse{
			Result: "success",
			Reason: "已报名",
		}, nil
	}

	// 报名成功且为首次/重新报名时发布事件（用于 Chat 服务自动加群）
	// 事件发布为异步执行，失败不会影响报名主流程
	l.publishMemberJoinedEvent(activityData.ID, userID)

	return &activity.RegisterActivityResponse{
		Result: "success",
		Reason: "",
	}, nil
}

// publishMemberJoinedEvent 发布用户报名成功事件
// - 仅处理有效 ID
// - Producer 未启用时直接跳过
// - 发布失败由 Producer 内部记录，不影响主流程
func (l *RegisterActivityLogic) publishMemberJoinedEvent(activityID uint64, userID int64) {
	if activityID == 0 || userID <= 0 {
		return
	}
	if l.svcCtx == nil || l.svcCtx.MsgProducer == nil {
		return
	}
	l.svcCtx.MsgProducer.PublishMemberJoined(l.ctx, activityID, uint64(userID))
}

func (l *RegisterActivityLogic) registerWithConsistency(
	activityID, userID int64,
	gen func() (*model.TicketPayload, error),
) (bool, error) {
	if gen == nil {
		return false, errors.New("ticket generator is nil")
	}

	result, err := l.svcCtx.ActivityRegistrationModel.RegisterWithTicket(
		l.ctx,
		uint64(activityID),
		uint64(userID),
		gen,
	)
	if err != nil {
		return false, err
	}

	verifyErr := l.verifyRegistrationWithTicket(activityID, userID)
	if verifyErr == nil {
		return result.AlreadyRegistered, nil
	}
	if !errors.Is(verifyErr, model.ErrRegistrationNotFound) &&
		!errors.Is(verifyErr, model.ErrTicketNotFound) {
		return result.AlreadyRegistered, verifyErr
	}

	// 数据缺失时修复一次，确保报名记录与票券一致
	_, err = l.svcCtx.ActivityRegistrationModel.RegisterWithTicket(
		l.ctx,
		uint64(activityID),
		uint64(userID),
		gen,
	)
	if err != nil {
		return false, err
	}
	if err := l.verifyRegistrationWithTicket(activityID, userID); err != nil {
		return false, err
	}
	if errors.Is(verifyErr, model.ErrRegistrationNotFound) {
		return false, nil
	}
	return result.AlreadyRegistered, nil
}

func (l *RegisterActivityLogic) verifyRegistrationWithTicket(activityID, userID int64) error {
	reg, err := l.svcCtx.ActivityRegistrationModel.FindByActivityUser(
		l.ctx,
		uint64(activityID),
		uint64(userID),
	)
	if err != nil {
		return err
	}
	if reg.Status != model.RegistrationStatusSuccess {
		return errors.New("registration status not success")
	}
	_, err = l.svcCtx.ActivityTicketModel.FindByRegistrationID(l.ctx, reg.ID)
	return err
}

const (
	totpDigits       = 6
	totpStepSeconds  = 30
	totpServerSecret = "campushub_totp_secret"
)

// generateTicketCode 生成短券码
func generateTicketCode() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	code := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf)
	code = strings.TrimRight(code, "=")
	if len(code) < 10 {
		return "", errors.New("ticket code too short")
	}
	return "TK" + code[:10], nil
}

// buildTicketQrPayload 组装二维码载荷
func buildTicketQrPayload(activityID int64, ticketCode string) string {
	activityPart := strconv.FormatInt(activityID, 36)
	return fmt.Sprintf("a%s|c%s", activityPart, ticketCode)
}

// deriveTotpSecret 派生TOTP密钥
func deriveTotpSecret(activityID, userID int64, ticketCode string) string {
	msg := fmt.Sprintf("%d:%d:%s", activityID, userID, ticketCode)
	h := hmac.New(sha1.New, []byte(totpServerSecret))
	_, _ = h.Write([]byte(msg))
	sum := h.Sum(nil)
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(sum)
}

// generateTotpCode 生成TOTP动态码
func generateTotpCode(secret string, now time.Time) (string, error) {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		return "", err
	}
	counter := uint64(now.Unix() / totpStepSeconds)
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], counter)

	h := hmac.New(sha1.New, key)
	_, _ = h.Write(buf[:])
	sum := h.Sum(nil)

	offset := sum[len(sum)-1] & 0x0f
	code := (int(sum[offset])&0x7f)<<24 |
		(int(sum[offset+1])&0xff)<<16 |
		(int(sum[offset+2])&0xff)<<8 |
		(int(sum[offset+3]) & 0xff)
	code %= 1000000
	return fmt.Sprintf("%06d", code), nil
}
