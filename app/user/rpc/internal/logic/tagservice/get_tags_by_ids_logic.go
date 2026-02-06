package tagservicelogic

import (
	"context"

	"activity-platform/app/user/model"
	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetTagsByIdsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetTagsByIdsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTagsByIdsLogic {
	return &GetTagsByIdsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetTagsByIdsLogic) GetTagsByIds(in *pb.GetTagsByIdsReq) (*pb.GetTagsByIdsResp, error) {
	if len(in.Ids) == 0 {
		return &pb.GetTagsByIdsResp{}, nil
	}

	// 1. 查询数据库
	tags, err := l.svcCtx.InterestTagModel.FindByIDs(l.ctx, in.Ids)
	if err != nil {
		l.Logger.Errorf("FindByIDs failed: %v", err)
		return nil, errorx.ErrDBError(err)
	}

	// 2. 构建返回结果和ID映射
	tagMap := make(map[int64]*model.InterestTag)
	var respTags []*pb.TagInfo
	for _, tag := range tags {
		tagMap[tag.TagID] = tag
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

	// 3. 找出无效ID
	var invalidIds []int64
	for _, id := range in.Ids {
		if _, ok := tagMap[id]; !ok {
			invalidIds = append(invalidIds, id)
		}
	}

	return &pb.GetTagsByIdsResp{
		Tags:       respTags,
		InvalidIds: invalidIds,
	}, nil
}
