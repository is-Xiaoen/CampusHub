// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"context"
	"html"
	"net/http"
	"strings"

	"activity-platform/app/user/api/internal/svc"
	"activity-platform/app/user/api/internal/types"
	"activity-platform/app/user/rpc/client/userbasicservice"
	"activity-platform/common/errorx"
	ctxUtils "activity-platform/common/utils/context"
	"activity-platform/common/utils/email"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateUserInfoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

// 修改用户信息
func NewUpdateUserInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *UpdateUserInfoLogic {
	return &UpdateUserInfoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *UpdateUserInfoLogic) UpdateUserInfo(req *types.UpdateUserInfoReq) (resp *types.UpdateUserInfoResp, err error) {
	nickname := strings.TrimSpace(req.Nickname)
	introduction := strings.TrimSpace(req.Introduction)
	avatarURL := strings.TrimSpace(req.AvatarUrl)
	gender := strings.TrimSpace(req.Gender)

	if nickname == "" || introduction == "" || avatarURL == "" || gender == "" || req.AvatarId <= 0 || len(req.InterestTagIds) == 0 {
		return nil, errorx.NewWithMessage(errorx.CodeInvalidParams, "参数不能为空")
	}

	if req.Age <= 0 || req.Age >= 200 {
		return nil, errorx.NewWithMessage(errorx.CodeInvalidParams, "年龄必须为正数并且小于两百岁")
	}

	var genderInt int64
	if gender == "男" || gender == "1" {
		genderInt = 1
	} else if gender == "女" || gender == "2" {
		genderInt = 2
	} else {
		return nil, errorx.NewWithMessage(errorx.CodeInvalidParams, "性别只能是男或女")
	}

	userId, err := ctxUtils.GetUserIdFromCtx(l.ctx)
	if err != nil {
		return nil, err
	}

	// 调用 RPC
	_, err = l.svcCtx.UserBasicServiceRpc.UpdateUserInfo(l.ctx, &userbasicservice.UpdateUserInfoReq{
		UserId:    userId,
		Nickname:  html.EscapeString(nickname),
		Introduce: html.EscapeString(introduction),
		Gender:    genderInt,
		AvatarId:  req.AvatarId,
		AvatarUrl: avatarURL,
		Age:       req.Age,
		TagIds:    req.InterestTagIds,
	})
	if err != nil {
		return nil, err
	}

	// 获取最新用户信息
	userInfoResp, err := l.svcCtx.UserBasicServiceRpc.GetUserInfo(l.ctx, &userbasicservice.GetUserInfoReq{
		UserId: userId,
	})
	if err != nil {
		return nil, err
	}

	userInfo := userInfoResp.UserInfo
	if userInfo == nil {
		l.Logger.Error("GetUserInfo returned nil userInfo")
		return nil, errorx.NewWithMessage(errorx.CodeInternalError, "获取用户信息失败")
	}

	var interestTags []types.InterestTag
	if userInfo.InterestTags != nil {
		for _, tag := range userInfo.InterestTags {
			if tag == nil {
				continue
			}
			interestTags = append(interestTags, types.InterestTag{
				Id:       int64(tag.Id),
				TagName:  tag.TagName,
				TagColor: tag.TagColor,
				TagIcon:  tag.TagIcon,
				TagDesc:  tag.TagDesc,
			})
		}
	}

	return &types.UpdateUserInfoResp{
		UserInfo: types.UserInfo{
			UserId:            int64(userInfo.UserId),
			Nickname:          userInfo.Nickname,
			AvatarUrl:         userInfo.AvatarUrl,
			Introduction:      userInfo.Introduction,
			Gender:            userInfo.Gender,
			Age:               userInfo.Age,
			ActivitiesNum:     int64(userInfo.ActivitiesNum),
			InitiateNum:       int64(userInfo.InitiateNum),
			Credit:            userInfo.Credit,
			IsStudentVerified: userInfo.IsStudentVerified,
			InterestTags:      interestTags,
			QqEmail:           email.DesensitizeEmail(userInfo.QqEmail),
		},
	}, nil
}
