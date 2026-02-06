// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"context"

	"activity-platform/app/user/api/internal/svc"
	"activity-platform/app/user/api/internal/types"
	"activity-platform/app/user/rpc/client/userbasicservice"
	ctxUtils "activity-platform/common/utils/context"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserInfoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取用户信息
func NewGetUserInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserInfoLogic {
	return &GetUserInfoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUserInfoLogic) GetUserInfo() (resp *types.UserInfo, err error) {
	userId, err := ctxUtils.GetUserIdFromCtx(l.ctx)
	if err != nil {
		return nil, err
	}

	rpcResp, err := l.svcCtx.UserBasicServiceRpc.GetUserInfo(l.ctx, &userbasicservice.GetUserInfoReq{
		UserId: userId,
	})
	if err != nil {
		return nil, err
	}

	userInfo := rpcResp.UserInfo

	var interestTags []types.InterestTag
	if userInfo.InterestTags != nil {
		for _, tag := range userInfo.InterestTags {
			interestTags = append(interestTags, types.InterestTag{
				Id:       int64(tag.Id),
				TagName:  tag.TagName,
				TagColor: tag.TagColor,
				TagIcon:  tag.TagIcon,
				TagDesc:  tag.TagDesc,
			})
		}
	}

	return &types.UserInfo{
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
	}, nil
}
