// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"context"

	"activity-platform/app/user/api/internal/svc"
	"activity-platform/app/user/api/internal/types"
	"activity-platform/app/user/rpc/client/userbasicservice"

	"github.com/zeromicro/go-zero/core/logx"
)

type LoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 用户登录
func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LoginLogic) Login(req *types.LoginReq) (resp *types.LoginResp, err error) {
	rpcResp, err := l.svcCtx.UserBasicServiceRpc.Login(l.ctx, &userbasicservice.LoginReq{
		QqEmail:       req.QqEmail,
		Password:      req.Password,
		LotNumber:     req.LotNumber,
		CaptchaOutput: req.CaptchaOutput,
		PassToken:     req.PassToken,
		GenTime:       req.GenTime,
	})
	if err != nil {
		return nil, err
	}

	var interestTags []types.InterestTag

	rpcUserInfo := rpcResp.UserInfo.UserInfo
	if rpcUserInfo.InterestTags != nil {
		for _, tag := range rpcUserInfo.InterestTags {
			interestTags = append(interestTags, types.InterestTag{
				Id:       int64(tag.Id),
				TagName:  tag.TagName,
				TagColor: tag.TagColor,
				TagIcon:  tag.TagIcon,
				TagDesc:  tag.TagDesc,
			})
		}
	}

	return &types.LoginResp{
		AccessToken:  rpcResp.AccessToken,
		RefreshToken: rpcResp.RefreshToken,
		UserInfo: types.UserInfo{
			UserId:            int64(rpcUserInfo.UserId),
			Nickname:          rpcUserInfo.Nickname,
			AvatarUrl:         rpcUserInfo.AvatarUrl,
			Introduction:      rpcUserInfo.Introduction,
			Gender:            rpcUserInfo.Gender,
			Age:               rpcUserInfo.Age,
			ActivitiesNum:     int64(rpcUserInfo.ActivitiesNum),
			InitiateNum:       int64(rpcUserInfo.InitiateNum),
			Credit:            rpcUserInfo.Credit,
			IsStudentVerified: rpcUserInfo.IsStudentVerified,
			InterestTags:      interestTags,
		},
	}, nil
}
