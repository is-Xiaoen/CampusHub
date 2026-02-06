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
	"activity-platform/common/utils/encrypt"

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
	qqEmail := in.QqEmail

	// 1. 生成6位验证码
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	code := fmt.Sprintf("%06d", rnd.Intn(1000000))

	// 2. 频率限制：1小时内最多发送10次
	limitKey := fmt.Sprintf("captcha:email:limit:%s", qqEmail)
	count, err := l.svcCtx.Redis.Incr(l.ctx, limitKey).Result()
	if err != nil {
		l.Logger.Errorf("Redis Incr failed: %v", err)
		return nil, errorx.NewSystemError("系统繁忙，请稍后再试")
	}
	if count == 1 {
		l.svcCtx.Redis.Expire(l.ctx, limitKey, time.Hour)
	}
	if count > 10 {
		return nil, errorx.NewDefaultError("验证码发送太频繁，请1小时后再试")
	}

	// 3. 存储验证码到Redis (有效期3分钟)
	key := fmt.Sprintf("captcha:email:%s:%s", in.Scene, qqEmail)
	encrypted := encrypt.EncryptPassword(code)
	err = l.svcCtx.Redis.Set(l.ctx, key, encrypted, 3*time.Minute).Err()
	if err != nil {
		l.Logger.Errorf("Redis Set failed: %v", err)
		return nil, errorx.NewSystemError("系统繁忙，请稍后再试")
	}

	// 4. 发送邮件
	emailCfg := email.EmailConfig{
		Host:     l.svcCtx.Config.Email.Host,
		Port:     l.svcCtx.Config.Email.Port,
		Username: l.svcCtx.Config.Email.Username,
		Password: l.svcCtx.Config.Email.Password,
		FromName: l.svcCtx.Config.Email.FromName,
		Subject:  l.svcCtx.Config.Email.Subject,
	}

	// 映射场景为中文
	var sceneStr string
	switch in.Scene {
	case "register":
		sceneStr = "注册账号"
	case "login":
		sceneStr = "登录账号"
	case "forget_password":
		sceneStr = "找回密码"
	case "delete_user":
		sceneStr = "注销账号"
	case "change_email":
		sceneStr = "更换邮箱"
	default:
		sceneStr = "身份验证"
	}

	// 这里选择同步发送以确保发送成功，如果失败则通知前端
	err = email.SendQQEmail(emailCfg, qqEmail, code, sceneStr)
	if err != nil {
		l.Logger.Errorf("Send email failed: %v", err)
		return nil, errorx.NewDefaultError("邮件发送失败，请检查邮箱是否正确")
	}

	return &pb.SendQQEmailResponse{}, nil
}
