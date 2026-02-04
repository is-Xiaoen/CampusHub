package qqemaillogic

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"
	"activity-platform/common/errorx"
	"activity-platform/common/utils/email"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendQQEmailLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendQQEmailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendQQEmailLogic {
	return &SendQQEmailLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SendQQEmailLogic) SendQQEmail(in *pb.SendQQEmailReq) (*pb.SendQQEmailResponse, error) {
	// 1. 生成6位验证码
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	code := fmt.Sprintf("%06d", rnd.Intn(1000000))

	// 2. 尝试存储到Redis (SetNX: 只有不存在时才设置成功，原子操作实现频率限制)
	key := fmt.Sprintf("captcha:email:%s:%s", in.Scene, in.QqEmail)
	success, err := l.svcCtx.Redis.SetNX(l.ctx, key, code, 2*time.Minute).Result()
	if err != nil {
		l.Logger.Errorf("Redis SetNX failed: %v", err)
		return nil, errorx.NewSystemError("系统繁忙，请稍后再试")
	}
	if !success {
		return nil, errorx.NewDefaultError("验证码发送太频繁，请2分钟后再试")
	}

	// 3. 发送邮件
	emailCfg := email.EmailConfig{
		Host:     l.svcCtx.Config.Email.Host,
		Port:     l.svcCtx.Config.Email.Port,
		Username: l.svcCtx.Config.Email.Username,
		Password: l.svcCtx.Config.Email.Password,
		FromName: l.svcCtx.Config.Email.FromName,
		Subject:  l.svcCtx.Config.Email.Subject,
	}

	// 这里选择同步发送以确保发送成功，如果失败则通知前端
	err = email.SendQQEmail(emailCfg, in.QqEmail, code)
	if err != nil {
		l.Logger.Errorf("Send email failed: %v", err)
		return nil, errorx.NewDefaultError("邮件发送失败，请检查邮箱是否正确")
	}

	return &pb.SendQQEmailResponse{}, nil
}
