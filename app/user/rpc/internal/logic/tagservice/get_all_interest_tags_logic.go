package tagservicelogic

import (
	"context"

	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetAllInterestTagsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetAllInterestTagsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAllInterestTagsLogic {
	return &GetAllInterestTagsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetAllInterestTags 获取所有的兴趣标签
func (l *GetAllInterestTagsLogic) GetAllInterestTags(in *pb.GetAllInterestTagsReq) (*pb.GetAllInterestTagsResp, error) {
	tags, err := l.svcCtx.InterestTagModel.ListAll(l.ctx)
	if err != nil {
		l.Logger.Errorf("Failed to list all tags: %v", err)
		return nil, errorx.ErrDBError(err)
	}

	var respTags []*pb.InterestTag
	for _, tag := range tags {
		respTags = append(respTags, &pb.InterestTag{
			Id:       uint64(tag.TagID),
			TagName:  tag.TagName,
			TagColor: tag.Color,
			TagIcon:  tag.Icon,
			TagDesc:  tag.TagDesc,
		})
	}

	return &pb.GetAllInterestTagsResp{
		InterestTags: respTags,
	}, nil
}
