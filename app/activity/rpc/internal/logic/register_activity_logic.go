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
	"activity-platform/common/ctxdata"

	mysqlerr "github.com/go-sql-driver/mysql"
	"github.com/zeromicro/go-zero/core/breaker"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
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
func (l *RegisterActivityLogic) RegisterActivity(in *activity.RegisterActivityRequest) (*activity.RegisterActivityResponse, error) {
	userID := ctxdata.GetUserIDFromCtx(l.ctx)
	if userID <= 0 || in.GetActivityId() <= 0 {
		return &activity.RegisterActivityResponse{
			Result: "fail",
			Reason: "参数错误",
		}, nil
	}

	// ==================== 第一步：限流检查 ====================
	if !l.svcCtx.RegistrationLimiter.AllowCtx(l.ctx) {
		return &activity.RegisterActivityResponse{
			Result: "fail",
			Reason: "请求过于频繁，请稍后再试",
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
	if !creditResp.GetAllowed() {
		_ = l.svcCtx.ActivityRegistrationModel.Create(l.ctx, &model.ActivityRegistration{
			ActivityID: uint64(in.GetActivityId()),
			UserID:     uint64(userID),
			Status:     model.RegistrationStatusFailed,
		})
		return &activity.RegisterActivityResponse{
			Result: "fail",
			Reason: creditResp.GetReason(),
		}, nil
	}

	// ==================== 实名校验 ====================
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
	if !verifyResp.GetIsVerified() {
		_ = l.svcCtx.ActivityRegistrationModel.Create(l.ctx, &model.ActivityRegistration{
			ActivityID: uint64(in.GetActivityId()),
			UserID:     uint64(userID),
			Status:     model.RegistrationStatusFailed,
		})
		return &activity.RegisterActivityResponse{
			Result: "fail",
			Reason: "请先完成学生认证",
		}, nil
	}

	// TODO: 检查活动是否存在/状态允许报名（同事实现）
	// TODO: 报名成功后名额-1（同事实现）

	// ==================== 第二步：熔断保护 ====================
	alreadyRegistered := false
	err = l.svcCtx.RegistrationBreaker.DoWithFallbackAcceptable(
		func() error {
			return l.svcCtx.DB.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
				reg := &model.ActivityRegistration{
					ActivityID: uint64(in.GetActivityId()),
					UserID:     uint64(userID),
					Status:     model.RegistrationStatusSuccess,
				}
				if err := tx.Create(reg).Error; err != nil {
					if isDuplicateKeyErr(err) {
						alreadyRegistered = true
						return nil
					}
					return err
				}

				for i := 0; i < 3; i++ {
					ticketCode, err := generateTicketCode()
					if err != nil {
						return err
					}
					secret := deriveTotpSecret(in.GetActivityId(), userID, ticketCode)
					qrPayload := buildTicketQrPayload(in.GetActivityId(), ticketCode)

					ticket := &model.ActivityTicket{
						ActivityID:     uint64(in.GetActivityId()),
						UserID:         uint64(userID),
						RegistrationID: reg.ID,
						TicketCode:     ticketCode,
						TicketUUID:     qrPayload,
						TotpSecret:     secret,
						Status:         model.TicketStatusUnused,
					}
					if err := tx.Create(ticket).Error; err != nil {
						if isDuplicateKeyErr(err) {
							continue
						}
						return err
					}
					return nil
				}

				return errors.New("生成票据失败")
			})
		},
		func(err error) error {
			return breaker.ErrServiceUnavailable
		},
		func(err error) bool {
			return err == nil
		},
	)
	if err != nil {
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

	return &activity.RegisterActivityResponse{
		Result: "success",
		Reason: "",
	}, nil
}

// isDuplicateKeyErr 判断是否为重复键错误
func isDuplicateKeyErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	var mysqlErr *mysqlerr.MySQLError
	if errors.As(err, &mysqlErr) {
		return mysqlErr.Number == 1062
	}
	return false
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
