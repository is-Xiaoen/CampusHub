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

	"github.com/go-redis/redis/v8"
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

	// 2. 频率限制
	// 2.1 60秒内限制：只能发1次
	limit60sKey := fmt.Sprintf("captcha:email:limit:60s:%s", qqEmail)
	exists, err := l.svcCtx.Redis.Exists(l.ctx, limit60sKey).Result()
	if err != nil {
		l.Logger.Errorf("Redis Exists failed: %v", err)
		return nil, errorx.ErrCacheError(err)
	}
	if exists > 0 {
		return nil, errorx.NewWithMessage(errorx.CodeCaptchaRateLimit, "发送过于频繁，请60秒后再试")
	}

	// 2.2 1小时内限制：只能发5次
	limit1hKey := fmt.Sprintf("captcha:email:limit:1h:%s", qqEmail)
	count1h, err := l.svcCtx.Redis.Get(l.ctx, limit1hKey).Int()
	if err != nil && err != redis.Nil {
		l.Logger.Errorf("Redis Get 1h failed: %v", err)
		return nil, errorx.ErrCacheError(err)
	}
	if count1h >= 5 {
		return nil, errorx.NewWithMessage(errorx.CodeCaptchaRateLimit, "1小时内发送次数过多，请稍后再试")
	}

	// 2.3 24小时内限制：只能发10次
	limit24hKey := fmt.Sprintf("captcha:email:limit:24h:%s", qqEmail)
	count24h, err := l.svcCtx.Redis.Get(l.ctx, limit24hKey).Int()
	if err != nil && err != redis.Nil {
		l.Logger.Errorf("Redis Get 24h failed: %v", err)
		return nil, errorx.ErrCacheError(err)
	}
	if count24h >= 10 {
		return nil, errorx.NewWithMessage(errorx.CodeCaptchaRateLimit, "今日发送次数已达上限，请明天再试")
	}

	// 2.4 所有校验通过，准备发送邮件（这里不再提前增加计数）
	// 注意：为了防止并发刷接口，60秒的锁仍然需要提前设置，并使用 SetNX 保证原子性
	// 提前设置的 Redis 锁主要是为了 防并发和防重复点击 。
	// 但如果后续邮件发送失败，我们会把这个锁删掉
	setSuccess, err := l.svcCtx.Redis.SetNX(l.ctx, limit60sKey, "1", 60*time.Second).Result()
	if err != nil {
		l.Logger.Errorf("Redis SetNX 60s limit failed: %v", err)
		return nil, errorx.ErrCacheError(err)
	}
	if !setSuccess {
		// 说明有并发请求已经抢先设置了锁
		return nil, errorx.NewWithMessage(errorx.CodeCaptchaRateLimit, "发送过于频繁，请60秒后再试")
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

	// 映射场景为中文
	var sceneStr string
	switch in.Scene {
	case "register":
		sceneStr = "注册账号"
	case "forget_password":
		sceneStr = "找回密码"
	case "delete_user":
		sceneStr = "注销账号"
	default:
		sceneStr = "身份验证"
	}

	// 这里选择同步发送以确保发送成功，如果失败则通知前端
	err = email.SendQQEmail(emailCfg, qqEmail, code, sceneStr)
	if err != nil {
		// 发送失败：撤销之前加上的 60 秒锁
		l.svcCtx.Redis.Del(l.ctx, limit60sKey)
		l.Logger.Errorf("Send email failed: %v", err)
		return nil, errorx.New(errorx.CodeEmailSendFailed)
	}

	// 4. 发送成功后：正式记录1小时和24小时的发送次数
	newCount1h, _ := l.svcCtx.Redis.Incr(l.ctx, limit1hKey).Result()
	if newCount1h == 1 {
		l.svcCtx.Redis.Expire(l.ctx, limit1hKey, time.Hour)
	}

	newCount24h, _ := l.svcCtx.Redis.Incr(l.ctx, limit24hKey).Result()
	if newCount24h == 1 {
		l.svcCtx.Redis.Expire(l.ctx, limit24hKey, 24*time.Hour)
	}

	// 5. 存储验证码到Redis (有效期3分钟)
	key := fmt.Sprintf("captcha:email:%s:%s", in.Scene, qqEmail)
	encrypted := encrypt.EncryptPassword(code)
	err = l.svcCtx.Redis.Set(l.ctx, key, encrypted, 3*time.Minute).Err()
	if err != nil {
		l.Logger.Errorf("Redis Set failed: %v", err)
		return nil, errorx.ErrCacheError(err)
	}

	return &pb.SendQQEmailResponse{}, nil
}
