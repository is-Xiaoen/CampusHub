package userbasicservicelogic

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"activity-platform/app/user/model"
	captchaservicelogic "activity-platform/app/user/rpc/internal/logic/captchaservice"
	creditservicelogic "activity-platform/app/user/rpc/internal/logic/creditservice"
	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"
	"activity-platform/common/errorx"
	"activity-platform/common/utils/email"
	"activity-platform/common/utils/encrypt"
	"activity-platform/common/utils/jwt"

	"github.com/google/uuid"
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
			return nil, errorx.New(errorx.CodeLoginFailed)
		}
		return nil, errorx.ErrDBError(err)
	}

	if !encrypt.ComparePassword(in.Password, user.Password) {
		return nil, errorx.New(errorx.CodeLoginFailed)
	}

	if user.Status == model.UserStatusDisabled {
		return nil, errorx.New(errorx.CodeUserDisabled)
	}
	if user.Status == model.UserStatusDeleted {
		return nil, errorx.New(errorx.CodeUserNotFound)
	}

	// 3. 生成Token
	accessJwtId := uuid.New().String()
	refreshJwtId := uuid.New().String()

	shortToken, err := jwt.GenerateShortToken(user.UserID, jwt.RoleUser, jwt.AuthConfig{
		Secret: l.svcCtx.Config.JWT.AccessSecret,
		Expire: l.svcCtx.Config.JWT.AccessExpire,
	}, accessJwtId, refreshJwtId)
	if err != nil {
		return nil, errorx.New(errorx.CodeTokenGenerateFailed)
	}
	longToken, err := jwt.GenerateLongToken(user.UserID, jwt.RoleUser, jwt.AuthConfig{
		Secret: l.svcCtx.Config.JWT.RefreshSecret,
		Expire: l.svcCtx.Config.JWT.RefreshExpire,
	}, accessJwtId, refreshJwtId)
	if err != nil {
		return nil, errorx.New(errorx.CodeTokenGenerateFailed)
	}

	// 记录长token到redis
	// key: token:refresh:{refreshJwtId}  value: userId
	refreshTokenKey := fmt.Sprintf("token:refresh:%s", refreshJwtId)
	if err := l.svcCtx.Redis.Set(l.ctx, refreshTokenKey, user.UserID, time.Duration(l.svcCtx.Config.JWT.RefreshExpire)*time.Second).Err(); err != nil {
		l.Logger.Errorf("Set refresh token to redis failed: %v", err)
		return nil, errorx.ErrCacheError(err)
	}

	// 4. 获取用户详细信息 (调用 GetUserInfoLogic 复用逻辑)
	// 4.1 获取信誉分 (独立调用，确保即使 GetUserInfo 失败也能尝试获取，或者补充 GetUserInfo 缺失的字段)
	var creditScore int64
	getCreditInfoLogic := creditservicelogic.NewGetCreditInfoLogic(l.ctx, l.svcCtx)
	creditInfo, errCredit := getCreditInfoLogic.GetCreditInfo(&pb.GetCreditInfoReq{
		UserId: int64(user.UserID),
	})
	if errCredit == nil {
		creditScore = creditInfo.Score
	} else {
		l.Logger.Errorf("Get credit info failed: %v, userId: %d", errCredit, user.UserID)
	}

	getUserInfoLogic := NewGetUserInfoLogic(l.ctx, l.svcCtx)
	userInfoResp, err := getUserInfoLogic.GetUserInfo(&pb.GetUserInfoReq{
		UserId: int64(user.UserID),
	})
	if err != nil {
		// 即使获取详情失败，登录也算成功，只是信息不全
		l.Logger.Errorf("GetUserInfo failed during login: %v", err)
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
				AvatarUrl:    "", // AvatarURL needs to be fetched separately, leaving empty on fallback
				Introduction: user.Introduction,
				Gender:       genderStr,
				Age:          strconv.FormatInt(user.Age, 10),
				QqEmail:      email.DesensitizeEmail(user.QQEmail),
				Credit:       creditScore,
			},
		}
	} else {
		// 成功获取详情后，也需要对邮箱脱敏
		userInfoResp.UserInfo.QqEmail = email.DesensitizeEmail(userInfoResp.UserInfo.QqEmail)
		// 填充 Credit 字段
		userInfoResp.UserInfo.Credit = creditScore
	}

	return &pb.LoginResponse{
		AccessToken:  shortToken.Token,
		RefreshToken: longToken.Token,
		UserInfo: &pb.LoginUserInfo{
			UserInfo: userInfoResp.UserInfo,
		},
	}, nil
}
