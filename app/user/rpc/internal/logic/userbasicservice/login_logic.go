package userbasicservicelogic

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"activity-platform/app/activity/rpc/activity"
	captchaservicelogic "activity-platform/app/user/rpc/internal/logic/captchaservice"
	creditservicelogic "activity-platform/app/user/rpc/internal/logic/creditservice"
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

	// 4. 获取附加信息
	// 4.1 信用分 (用于isStudentVerified? 用户需求如此，暂且获取)
	// 注意：这里按照用户需求调用 GetCreditInfo 逻辑
	getCreditLogic := creditservicelogic.NewGetCreditInfoLogic(l.ctx, l.svcCtx)
	creditResp, err := getCreditLogic.GetCreditInfo(&pb.GetCreditInfoReq{
		UserId: user.UserID,
	})
	if err != nil {
		l.Logger.Errorf("GetCreditInfo failed: %v", err)
	} else {
		l.Logger.Infof("User credit score: %d", creditResp.Score)
	}

	// 4.2 真实的学生认证状态 (通过StudentVerificationModel)
	// StudentVerificationModel.IsVerified 
	isVerified, err := l.svcCtx.StudentVerificationModel.IsVerified(l.ctx, user.UserID)
	if err != nil {
		l.Logger.Errorf("Check IsVerified failed: %v", err)
		isVerified = false
	}

	// 4.3 活动统计 (ActivityRpc)
	registeredCountResp, err := l.svcCtx.ActivityRpc.GetRegisteredCount(l.ctx, &activity.GetRegisteredCountRequest{
		UserId: user.UserID,
	})
	var activitiesNum uint32
	if err == nil {
		activitiesNum = uint32(registeredCountResp.Count)
	} else {
		l.Logger.Errorf("GetRegisteredCount failed: %v", err)
	}

	publishedResp, err := l.svcCtx.ActivityRpc.GetUserPublishedActivities(l.ctx, &activity.GetUserPublishedActivitiesReq{
		UserId:   user.UserID,
		Page:     1,
		PageSize: 1, // 只需要总数? 接口似乎返回List和Pagination
	})
	var initiateNum uint32
	if err == nil && publishedResp.Pagination != nil {
		initiateNum = uint32(publishedResp.Pagination.Total)
	} else {
		l.Logger.Errorf("GetUserPublishedActivities failed: %v", err)
	}

	// 4.4 兴趣标签 (Database)
	relations, err := l.svcCtx.UserInterestRelationModel.ListByUserID(l.ctx, user.UserID)
	var interestTags []*pb.InterestTag
	if err == nil {
		for _, rel := range relations {
			tag, err := l.svcCtx.InterestTagModel.FindByID(l.ctx, rel.TagID)
			if err == nil && tag != nil {
				interestTags = append(interestTags, &pb.InterestTag{
					Id:       uint64(tag.TagID),
					TagName:  tag.TagName,
					TagColor: tag.Color,
					TagIcon:  tag.Icon,
				})
			}
		}
	}

	// 5. 组装响应
	var genderStr string
	switch user.Gender {
	case 1:
		genderStr = "男"
	case 2:
		genderStr = "女"
	default:
		genderStr = "未知"
	}

	return &pb.LoginResponse{
		AccessToken:  shortToken.Token,
		RefreshToken: longToken.Token,
		UserInfo: &pb.LoginUserInfo{
			UserInfo: &pb.UserInfo{
				UserId:            uint64(user.UserID),
				Nickname:          user.Nickname,
				AvatarUrl:         user.AvatarURL,
				Introduction:      user.Introduction,
				Gender:            genderStr,
				Age:               strconv.FormatInt(user.Age, 10),
				ActivitiesNum:     activitiesNum,
				IsStudentVerified: isVerified,
				InitiateNum:       initiateNum,
				InterestTags:      interestTags,
			},
		},
	}, nil
}
