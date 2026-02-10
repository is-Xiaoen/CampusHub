package base

import (
	"context"

	"activity-platform/app/user/api/internal/svc"
	"activity-platform/app/user/api/internal/types"
	"activity-platform/app/user/rpc/client/userbasicservice"

	"github.com/zeromicro/go-zero/core/logx"
)

type RegisterLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 注册
func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RegisterLogic) Register(req *types.RegisterReq) (resp *types.RegisterResp, err error) {
	// 调用 RPC 层注册接口
	rpcResp, err := l.svcCtx.UserBasicServiceRpc.Register(l.ctx, &userbasicservice.RegisterReq{
		QqEmail:  req.QqEmail,
		QqCode:   req.QqCode,
		Password: req.Password,
		Nickname: req.Nickname,
	})
	if err != nil {
		return nil, err
	}

	// 转换 InterestTags
	var interestTags []types.InterestTag
	if rpcResp.UserInfo.InterestTags != nil {
		interestTags = make([]types.InterestTag, 0, len(rpcResp.UserInfo.InterestTags))
		for _, tag := range rpcResp.UserInfo.InterestTags {
			interestTags = append(interestTags, types.InterestTag{
				Id:       int64(tag.Id),
				TagName:  tag.TagName,
				TagColor: tag.TagColor,
				TagIcon:  tag.TagIcon,
				TagDesc:  tag.TagDesc,
			})
		}
	}

	return &types.RegisterResp{
		AccessToken:  rpcResp.AccessToken,
		RefreshToken: rpcResp.RefreshToken,
		UserInfo: types.UserInfo{
			UserId:            int64(rpcResp.UserInfo.UserId),
			Nickname:          rpcResp.UserInfo.Nickname,
			AvatarUrl:         rpcResp.UserInfo.AvatarUrl,
			Introduction:      rpcResp.UserInfo.Introduction,
			Gender:            rpcResp.UserInfo.Gender,
			Age:               rpcResp.UserInfo.Age,
			ActivitiesNum:     int64(rpcResp.UserInfo.ActivitiesNum),
			InitiateNum:       int64(rpcResp.UserInfo.InitiateNum),
			Credit:            rpcResp.UserInfo.Credit,
			IsStudentVerified: rpcResp.UserInfo.IsStudentVerified,
			InterestTags:      interestTags,
			QqEmail:           rpcResp.UserInfo.QqEmail,
		},
	}, nil
}
