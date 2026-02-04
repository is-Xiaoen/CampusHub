package qqemaillogic

import (
	"activity-platform/common/errorx"
	"context"
	"fmt"
	"time"

	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
)

type CheckQQEmailLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCheckQQEmailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CheckQQEmailLogic {
	return &CheckQQEmailLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CheckQQEmailLogic) CheckQQEmail(in *pb.CheckQQEmailReq) (*pb.CheckQQEmailResponse, error) {
	key := fmt.Sprintf("captcha:email:%s:%s", in.Scene, in.QqEmail)
	errKey := fmt.Sprintf("captcha:email:err:%s:%s", in.Scene, in.QqEmail)

	// 1. 获取验证码
	val, err := l.svcCtx.Redis.Get(l.ctx, key).Result()
	if err == redis.Nil {
		return nil, errorx.NewDefaultError("验证码已过期或未发送")
	}
	if err != nil {
		l.Logger.Errorf("Get email captcha failed: %v", err)
		return nil, errorx.NewSystemError("系统繁忙，请稍后再试")
	}

	// 2. 校验验证码
	if val != in.QqCode {
		// 记录错误次数
		count, err := l.svcCtx.Redis.Incr(l.ctx, errKey).Result()
		if err != nil {
			l.Logger.Errorf("Incr email captcha error count failed: %v", err)
			// 即使记录失败也返回验证码错误
			return nil, errorx.NewDefaultError("验证码错误")
		}

		// 第一次错误设置过期时间，跟随验证码有效期（假设这里设为5分钟，覆盖大部分场景）
		if count == 1 {
			l.svcCtx.Redis.Expire(l.ctx, errKey, 5*time.Minute)
		}

		// 输错3次以上（即第4次错误时，或者大于3次时），清除验证码
		if count > 3 {
			l.svcCtx.Redis.Del(l.ctx, key, errKey)
			return nil, errorx.NewDefaultError("验证码错误次数过多，请重新获取")
		}

		return nil, errorx.NewDefaultError(fmt.Sprintf("验证码错误，您还有%d次机会", 3-count))
	}

	// 3. 验证成功，清除验证码和错误次数
	l.svcCtx.Redis.Del(l.ctx, key, errKey)
	return &pb.CheckQQEmailResponse{}, nil
}
