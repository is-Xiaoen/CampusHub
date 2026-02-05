package base

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

// 登录
func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LoginLogic) Login(req *types.LoginReq) (resp *types.LoginResp, err error) {
	// 调用 RPC 层登录接口
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

	// 转换 InterestTags
	var interestTags []types.InterestTag
	if rpcResp.UserInfo.UserInfo.InterestTags != nil {
		interestTags = make([]types.InterestTag, 0, len(rpcResp.UserInfo.UserInfo.InterestTags))
		for _, tag := range rpcResp.UserInfo.UserInfo.InterestTags {
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
			UserId:            int64(rpcResp.UserInfo.UserInfo.UserId),
			Nickname:          rpcResp.UserInfo.UserInfo.Nickname,
			AvatarUrl:         rpcResp.UserInfo.UserInfo.AvatarUrl,
			Introduction:      rpcResp.UserInfo.UserInfo.Introduction,
			Gender:            rpcResp.UserInfo.UserInfo.Gender,
			Age:               rpcResp.UserInfo.UserInfo.Age,
			ActivitiesNum:     int64(rpcResp.UserInfo.UserInfo.ActivitiesNum),
			InitiateNum:       int64(rpcResp.UserInfo.UserInfo.InitiateNum),
			Credit:            rpcResp.UserInfo.UserInfo.Credit,
			IsStudentVerified: rpcResp.UserInfo.UserInfo.IsStudentVerified,
			InterestTags:      interestTags,
		},
	}, nil
}
