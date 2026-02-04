package userbasicservicelogic

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	captchaservicelogic "activity-platform/app/user/rpc/internal/logic/captchaservice"
	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"
	"activity-platform/common/errorx"
	"activity-platform/common/utils/encrypt"
	"activity-platform/common/utils/jwt"

	"github.com/zeromicro/go-zero/core/logx"
)

type LoginLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *LoginLogic) Login(in *pb.LoginReq) (*pb.LoginResponse, error) {
	// 1. 调用验证码校验逻辑
	checkCaptchaLogic := captchaservicelogic.NewCheckCaptchaLogic(l.ctx, l.svcCtx)
	_, err := checkCaptchaLogic.CheckCaptcha(&pb.CheckCaptchaReq{
		LotNumber:     in.LotNumber,
		CaptchaOutput: in.CaptchaOutput,
		PassToken:     in.PassToken,
		GenTime:       in.GenTime,
	})

	if err != nil {
		return nil, err
	}

	// 2. 校验账号密码
	user, err := l.svcCtx.UserModel.FindByQQEmail(l.ctx, in.QqEmail)
	if err != nil {
		if strings.Contains(err.Error(), "record not found") {
			return nil, errorx.NewDefaultError("账号或密码错误")
		}
		return nil, errorx.NewSystemError("系统繁忙，请稍后再试")
	}

	if !encrypt.ComparePassword(in.Password, user.Password) {
		return nil, errorx.NewDefaultError("账号或密码错误")
	}

	if user.Status == 0 { // 假设0是禁用
		return nil, errorx.NewDefaultError("账号已被禁用")
	}

	// 3. 生成Token
	shortToken, err := jwt.GenerateShortToken(user.UserID, jwt.RoleUser, jwt.AuthConfig(l.svcCtx.Config.Auth))
	if err != nil {
		return nil, errorx.NewSystemError("Token生成失败")
	}
	longToken, err := jwt.GenerateLongToken(user.UserID, jwt.RoleUser, jwt.AuthConfig(l.svcCtx.Config.RefreshAuth))
	if err != nil {
		return nil, errorx.NewSystemError("Token生成失败")
	}

	// 记录长token到redis
	refreshTokenKey := fmt.Sprintf("refresh_token:%d", user.UserID)
	if err := l.svcCtx.Redis.Set(l.ctx, refreshTokenKey, longToken.Token, time.Duration(l.svcCtx.Config.RefreshAuth.AccessExpire)*time.Second).Err(); err != nil {
		l.Logger.Errorf("Set refresh token to redis failed: %v", err)
		return nil, errorx.NewSystemError("系统繁忙，请稍后再试")
	}

	// 4. 获取用户详细信息 (调用 GetUserInfoLogic 复用逻辑)
	getUserInfoLogic := NewGetUserInfoLogic(l.ctx, l.svcCtx)
	userInfoResp, err := getUserInfoLogic.GetUserInfo(&pb.GetUserInfoReq{
		UserId: int64(user.UserID),
	})
	if err != nil {
		// 即使获取详情失败，登录也算成功，只是信息不全? 或者记录日志返回基础信息
		// 这里选择记录日志，返回基础信息，或者让 GetUserInfoLogic 保证尽可能返回数据
		l.Logger.Errorf("GetUserInfo failed during login: %v", err)
		// 如果必须返回 UserInfo，可以手动构造一个基础的，或者直接返回错误取决于业务
		// 既然 GetUserInfoLogic 内部处理了错误并返回 errorx，这里如果报错可能说明系统问题或用户不存在
		// 鉴于 user 已经查到了，GetUserInfo 应该能查到。
		// 为了健壮性，这里如果失败，我们还是返回一个基础的 UserInfo
		var genderStr string
		switch user.Gender {
		case 1:
			genderStr = "男"
		case 2:
			genderStr = "女"
		default:
			genderStr = "未知"
		}
		userInfoResp = &pb.GetUserInfoResponse{
			UserInfo: &pb.UserInfo{
				UserId:       uint64(user.UserID),
				Nickname:     user.Nickname,
				AvatarUrl:    user.AvatarURL,
				Introduction: user.Introduction,
				Gender:       genderStr,
				Age:          strconv.FormatInt(user.Age, 10),
			},
		}
	}

	return &pb.LoginResponse{
		AccessToken:  shortToken.Token,
		RefreshToken: longToken.Token,
		UserInfo: &pb.LoginUserInfo{
			UserInfo: userInfoResp.UserInfo,
		},
	}, nil
}
