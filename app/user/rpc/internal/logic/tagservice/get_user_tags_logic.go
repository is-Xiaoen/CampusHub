package tagservicelogic

import (
	"context"

	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserTagsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetUserTagsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserTagsLogic {
	return &GetUserTagsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetUserTagsLogic) GetUserTags(in *pb.GetUserTagsReq) (*pb.GetUserTagsResponse, error) {
	// 1. 查询用户关联的标签ID
	relations, err := l.svcCtx.UserInterestRelationModel.ListByUserID(l.ctx, in.UserId)
	if err != nil {
		l.Logger.Errorf("Failed to list user interest relations: %v", err)
		return nil, errorx.ErrDBError(err)
	}

	if len(relations) == 0 {
		return &pb.GetUserTagsResponse{}, nil
	}

	// 2. 提取标签ID列表
	var tagIDs []int64
	for _, r := range relations {
		tagIDs = append(tagIDs, r.TagID)
	}

	// 3. 查询标签基础信息（只查ID和Name）
	tags, err := l.svcCtx.InterestTagModel.FindBasicInfoByIDs(l.ctx, tagIDs)
	if err != nil {
		l.Logger.Errorf("Failed to find tags by IDs: %v", err)
		return nil, errorx.ErrDBError(err)
	}

	// 4. 构建响应
	var respTags []*pb.UserTag
	for _, t := range tags {
		respTags = append(respTags, &pb.UserTag{
			Id:   uint64(t.TagID),
			Name: t.TagName,
		})
	}

	return &pb.GetUserTagsResponse{
		Tags: respTags,
	}, nil
}
