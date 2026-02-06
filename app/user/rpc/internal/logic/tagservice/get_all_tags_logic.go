package tagservicelogic

import (
	"context"
	"time"

	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetAllTagsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetAllTagsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAllTagsLogic {
	return &GetAllTagsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 马肖阳的标签接口
func (l *GetAllTagsLogic) GetAllTags(in *pb.GetAllTagsReq) (*pb.GetAllTagsResp, error) {
	sinceTime := time.Unix(0, 0)
	if in.SinceTimestamp > 0 {
		sinceTime = time.Unix(in.SinceTimestamp, 0)
	}

	tags, err := l.svcCtx.InterestTagModel.ListSince(l.ctx, sinceTime)
	if err != nil {
		l.Logger.Errorf("ListSince tags failed: %v", err)
		return nil, err
	}

	var respTags []*pb.TagInfo
	for _, tag := range tags {
		respTags = append(respTags, &pb.TagInfo{
			Id:          uint64(tag.TagID),
			Name:        tag.TagName,
			Color:       tag.Color,
			Icon:        tag.Icon,
			Status:      uint64(tag.Status),
			Description: tag.TagDesc,
			CreatedAt:   tag.CreateTime.Unix(),
			UpdatedAt:   tag.UpdateTime.Unix(),
		})
	}

	return &pb.GetAllTagsResp{
		Tags:            respTags,
		ServerTimestamp: time.Now().Unix(),
	}, nil
}
