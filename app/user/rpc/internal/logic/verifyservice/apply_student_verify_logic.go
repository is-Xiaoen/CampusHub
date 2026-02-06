/**
 * @projectName: CampusHub
 * @package: verifyservicelogic
 * @className: ApplyStudentVerifyLogic
 * @author: lijunqi
 * @description: 提交学生认证申请逻辑层
 * @date: 2026-01-31
 * @version: 1.0
 */

package verifyservicelogic

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"activity-platform/app/user/model"
	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"
	"activity-platform/common/constants"
	"activity-platform/common/errorx"
	"activity-platform/common/messaging"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

// ApplyStudentVerifyLogic 提交学生认证申请逻辑处理器
type ApplyStudentVerifyLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

// NewApplyStudentVerifyLogic 创建提交认证申请逻辑实例
func NewApplyStudentVerifyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ApplyStudentVerifyLogic {
	return &ApplyStudentVerifyLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// ApplyStudentVerify 提交学生认证申请
// 业务逻辑:
//  1. 限流检查（20s内≤2次）
//  2. 唯一性校验（学校+学号）
//  3. 状态机校验
//  4. 创建/更新记录并触发OCR
func (l *ApplyStudentVerifyLogic) ApplyStudentVerify(in *pb.ApplyStudentVerifyReq) (*pb.ApplyStudentVerifyResp, error) {
	// 1. 参数校验
	if err := l.validateParams(in); err != nil {
		return nil, err
	}

	// 2. 限流检查
	if err := l.checkRateLimit(in.UserId); err != nil {
		return nil, err
	}

	// 3. 唯一性校验（学校+学号是否已被其他用户认证）
	exists, err := l.svcCtx.StudentVerificationModel.ExistsBySchoolAndStudentID(
		l.ctx, in.SchoolName, in.StudentId, in.UserId)
	if err != nil {
		l.Errorf("ApplyStudentVerify 唯一性校验失败: err=%v", err)
		return nil, errorx.ErrDBError(err)
	}
	if exists {
		l.Infof("[WARN] ApplyStudentVerify 学号已被占用: school=%s, studentId=%s",
			in.SchoolName, in.StudentId)
		return nil, errorx.ErrVerifyStudentIDUsed()
	}

	// 4. 查询现有记录
	verification, err := l.svcCtx.StudentVerificationModel.FindByUserID(l.ctx, in.UserId)
	if err != nil && err != gorm.ErrRecordNotFound {
		l.Errorf("ApplyStudentVerify 查询记录失败: userId=%d, err=%v", in.UserId, err)
		return nil, errorx.ErrDBError(err)
	}

	var verifyID int64
	var createdAt time.Time

	if verification == nil {
		// 5a. 首次申请，创建新记录
		newVerification := &model.StudentVerification{
			UserID:        in.UserId,
			Status:        constants.VerifyStatusOcrPending,
			RealName:      in.RealName,
			SchoolName:    in.SchoolName,
			StudentID:     in.StudentId,
			Department:    in.Department,
			AdmissionYear: in.AdmissionYear,
			FrontImageURL: in.FrontImageUrl,
			BackImageURL:  in.BackImageUrl,
			Operator:      constants.VerifyOperatorUserApply,
		}

		if err := l.svcCtx.StudentVerificationModel.Create(l.ctx, newVerification); err != nil {
			l.Errorf("ApplyStudentVerify 创建记录失败: userId=%d, err=%v", in.UserId, err)
			return nil, errorx.ErrDBError(err)
		}

		verifyID = newVerification.ID
		createdAt = newVerification.CreatedAt
		l.Infof("ApplyStudentVerify 创建新记录成功: userId=%d, verifyId=%d",
			in.UserId, verifyID)
	} else {
		// 5b. 已有记录，检查状态是否允许申请
		if !constants.CanApply(verification.Status) {
			l.Infof("[WARN] ApplyStudentVerify 当前状态不允许申请: userId=%d, status=%d",
				in.UserId, verification.Status)
			return nil, errorx.ErrVerifyCannotApply()
		}

		// 检查拒绝后冷却期
		if verification.Status == constants.VerifyStatusRejected {
			if verification.ReviewedAt != nil {
				cooldownEnd := verification.ReviewedAt.Add(
					time.Duration(constants.VerifyRejectCooldownHours) * time.Hour)
				if time.Now().Before(cooldownEnd) {
					l.Infof("[WARN] ApplyStudentVerify 拒绝后冷却期内: userId=%d, cooldownEnd=%v",
						in.UserId, cooldownEnd)
					return nil, errorx.ErrVerifyRejectCooldown()
				}
			}
		}

		// 更新记录
		verification.Status = constants.VerifyStatusOcrPending
		verification.RealName = in.RealName
		verification.SchoolName = in.SchoolName
		verification.StudentID = in.StudentId
		verification.Department = in.Department
		verification.AdmissionYear = in.AdmissionYear
		verification.FrontImageURL = in.FrontImageUrl
		verification.BackImageURL = in.BackImageUrl
		verification.Operator = constants.VerifyOperatorUserApply
		verification.RejectReason = ""
		verification.CancelReason = ""

		if err := l.svcCtx.StudentVerificationModel.Update(l.ctx, verification); err != nil {
			l.Errorf("ApplyStudentVerify 更新记录失败: userId=%d, err=%v", in.UserId, err)
			return nil, errorx.ErrDBError(err)
		}

		verifyID = verification.ID
		createdAt = verification.UpdatedAt
		l.Infof("ApplyStudentVerify 更新记录成功: userId=%d, verifyId=%d",
			in.UserId, verifyID)
	}

	// 6. 发布认证申请事件到 MQ，异步触发 OCR 处理
	l.publishVerifyEvent(verifyID, in)

	return &pb.ApplyStudentVerifyResp{
		VerifyId:   verifyID,
		Status:     int32(constants.VerifyStatusOcrPending),
		StatusDesc: constants.GetVerifyStatusName(constants.VerifyStatusOcrPending),
		CreatedAt:  createdAt.Unix(),
	}, nil
}

// validateParams 参数校验
func (l *ApplyStudentVerifyLogic) validateParams(in *pb.ApplyStudentVerifyReq) error {
	if in.UserId <= 0 {
		return errorx.ErrInvalidParams("用户ID无效")
	}
	if in.RealName == "" {
		return errorx.ErrInvalidParams("真实姓名不能为空")
	}
	if in.SchoolName == "" {
		return errorx.ErrInvalidParams("学校名称不能为空")
	}
	if in.StudentId == "" {
		return errorx.ErrInvalidParams("学号不能为空")
	}
	if in.FrontImageUrl == "" {
		return errorx.ErrInvalidParams("学生证正面图片不能为空")
	}
	if in.BackImageUrl == "" {
		return errorx.ErrInvalidParams("学生证详情面图片不能为空")
	}
	return nil
}

// publishVerifyEvent 发布认证申请事件到 MQ
// 异步触发 OCR 处理，不阻塞主流程（发布失败只记录日志）
func (l *ApplyStudentVerifyLogic) publishVerifyEvent(
	verifyID int64,
	in *pb.ApplyStudentVerifyReq,
) {
	if l.svcCtx.MsgClient == nil {
		l.Infof("[WARN] 消息客户端未初始化，跳过事件发布: verifyId=%d", verifyID)
		return
	}

	// 构造事件数据
	eventData := messaging.VerifyApplyEventData{
		VerifyID:      verifyID,
		UserID:        in.UserId,
		FrontImageURL: in.FrontImageUrl,
		BackImageURL:  in.BackImageUrl,
		Timestamp:     time.Now().Unix(),
	}

	// 序列化内层数据
	dataBytes, err := json.Marshal(eventData)
	if err != nil {
		l.Errorf("ApplyStudentVerify 序列化事件数据失败: verifyId=%d, err=%v", verifyID, err)
		return
	}

	// 包装为通用消息格式（与 credit_event 保持一致）
	rawMsg := messaging.NewRawMessage(messaging.VerifyEventApplyOcr, string(dataBytes))
	payload, err := json.Marshal(rawMsg)
	if err != nil {
		l.Errorf("ApplyStudentVerify 序列化消息失败: verifyId=%d, err=%v", verifyID, err)
		return
	}

	// 发布到 verify:events Topic
	if err := l.svcCtx.MsgClient.Publish(l.ctx, messaging.TopicVerifyEvent, payload); err != nil {
		l.Errorf("ApplyStudentVerify 发布事件失败: verifyId=%d, err=%v", verifyID, err)
		return
	}

	l.Infof("ApplyStudentVerify 事件已发布: verifyId=%d, userId=%d, topic=%s",
		verifyID, in.UserId, messaging.TopicVerifyEvent)
}

// checkRateLimit 限流检查
// 20秒内最多提交2次
func (l *ApplyStudentVerifyLogic) checkRateLimit(userID int64) error {
	key := fmt.Sprintf("%s%d", constants.VerifyRateLimitPrefix, userID)

	// 获取当前计数
	count, err := l.svcCtx.Redis.Incr(l.ctx, key).Result()
	if err != nil {
		l.Errorf("ApplyStudentVerify 限流检查失败: userId=%d, err=%v", userID, err)
		// 限流检查失败不阻塞业务，只记录日志
		return nil
	}

	// 首次设置过期时间
	if count == 1 {
		l.svcCtx.Redis.Expire(l.ctx, key, time.Duration(constants.VerifyRateLimitWindow)*time.Second)
	}

	// 超过限制
	if count > int64(constants.VerifyRateLimitMax) {
		l.Infof("[WARN] ApplyStudentVerify 触发限流: userId=%d, count=%d", userID, count)
		return errorx.ErrVerifyRateLimit()
	}

	return nil
}
