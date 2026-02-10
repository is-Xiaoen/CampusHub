package user

import (
	"context"

	"activity-platform/app/user/api/internal/svc"
	"activity-platform/app/user/api/internal/types"
	"activity-platform/app/user/rpc/pb/pb"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserHomeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取用户主页信息
func NewGetUserHomeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserHomeLogic {
	return &GetUserHomeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUserHomeLogic) GetUserHome(req *types.GetUserHomeReq) (resp *types.GetUserHomeResp, err error) {
	// 调用 RPC 获取用户主页信息
	rpcReq := &pb.GetUserHomeReq{
		UserId:            req.UserId,
		JoinedPage:        req.JoinedPage,
		JoinedPageSize:    req.JoinedPageSize,
		PublishedPage:     req.PublishedPage,
		PublishedPageSize: req.PublishedPageSize,
		PublishedStatus:   -2, // 固定查询全部活动
	}

	rpcResp, err := l.svcCtx.UserBasicServiceRpc.GetUserHome(l.ctx, rpcReq)
	if err != nil {
		return nil, errorx.FromError(err)
	}

	// 转换响应
	resp = &types.GetUserHomeResp{
		UserInfo: types.UserHomeInfo{
			UserId:       rpcResp.UserInfo.UserId,
			Nickname:     rpcResp.UserInfo.Nickname,
			AvatarUrl:    rpcResp.UserInfo.AvatarUrl,
			Introduction: rpcResp.UserInfo.Introduction,
			Gender:       rpcResp.UserInfo.Gender,
			Age:          rpcResp.UserInfo.Age,
		},
		Tags: make([]types.InterestTag, 0),
		JoinedActivities: types.UserHomeActivityList{
			Total: rpcResp.JoinedActivities.Total,
			List:  make([]types.UserHomeActivityItem, 0),
		},
		PublishedActivities: types.UserHomeActivityList{
			Total: rpcResp.PublishedActivities.Total,
			List:  make([]types.UserHomeActivityItem, 0),
		},
	}

	// 转换标签
	for _, tag := range rpcResp.Tags {
		resp.Tags = append(resp.Tags, types.InterestTag{
			Id:       tag.TagId,
			TagName:  tag.TagName,
			TagColor: tag.Color,
			TagIcon:  tag.Icon,
			TagDesc:  tag.TagDesc,
		})
	}

	// 转换参加的活动
	for _, item := range rpcResp.JoinedActivities.List {
		resp.JoinedActivities.List = append(resp.JoinedActivities.List, types.UserHomeActivityItem{
			Id:       item.Id,
			Name:     item.Name,
			Time:     item.Time,
			Status:   item.Status,
			ImageUrl: item.ImageUrl,
		})
	}

	// 转换发起的活动
	for _, item := range rpcResp.PublishedActivities.List {
		resp.PublishedActivities.List = append(resp.PublishedActivities.List, types.UserHomeActivityItem{
			Id:       item.Id,
			Name:     item.Name,
			Time:     item.Time,
			Status:   item.Status,
			ImageUrl: item.ImageUrl,
		})
	}

	return resp, nil
}
