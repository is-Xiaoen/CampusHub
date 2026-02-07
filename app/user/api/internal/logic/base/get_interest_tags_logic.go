package base

import (
	"context"

	"activity-platform/app/user/api/internal/svc"
	"activity-platform/app/user/api/internal/types"
	"activity-platform/app/user/rpc/client/tagservice"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetInterestTagsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetInterestTagsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetInterestTagsLogic {
	return &GetInterestTagsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetInterestTagsLogic) GetInterestTags() (resp *types.GetInterestTagsResp, err error) {
	rpcResp, err := l.svcCtx.TagServiceRpc.GetAllInterestTags(l.ctx, &tagservice.GetAllInterestTagsReq{})
	if err != nil {
		l.Logger.Errorf("Failed to get all interest tags: %v", err)
		return nil, err
	}

	var tags []types.InterestTag
	for _, tag := range rpcResp.InterestTags {
		tags = append(tags, types.InterestTag{
			Id:       int64(tag.Id),
			TagName:  tag.TagName,
			TagColor: tag.TagColor,
			TagIcon:  tag.TagIcon,
			TagDesc:  tag.TagDesc,
		})
	}

	return &types.GetInterestTagsResp{
		InterestTags: tags,
	}, nil
}
