package tagservicelogic

import (
	"context"

	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"

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
	// 1. 获取用户ID (优先使用请求参数中的UserId)
	userID := in.UserId
	if userID == 0 {
		return &pb.GetUserTagsResponse{}, nil
	}

	// 2. 查询用户关联的标签ID列表
	// 使用 ListByUserID 方法，根据 user_id 字段查询
	relations, err := l.svcCtx.UserInterestRelationModel.ListByUserID(l.ctx, userID)
	if err != nil {
		l.Logger.Errorf("ListByUserID failed: %v", err)
		return nil, err
	}

	if len(relations) == 0 {
		return &pb.GetUserTagsResponse{}, nil
	}

	// 3. 提取标签ID
	var tagIDs []int64
	for _, r := range relations {
		tagIDs = append(tagIDs, r.TagID)
	}

	// 4. 批量查询标签详情
	// 使用 FindByIDs 方法，根据 tag_id 列表查询指定字段信息
	tags, err := l.svcCtx.InterestTagModel.FindByIDs(l.ctx, tagIDs)
	if err != nil {
		l.Logger.Errorf("FindByIDs failed: %v", err)
		return nil, err
	}

	// 5. 组装响应
	var respTags []*pb.TagInfo
	for _, t := range tags {
		respTags = append(respTags, &pb.TagInfo{
			Id:          uint64(t.TagID),
			Name:        t.TagName,
			Color:       t.Color,
			Icon:        t.Icon,
			Status:      uint64(t.Status),
			Description: t.TagDesc,
			CreatedAt:   t.CreateTime.Unix(),
			UpdatedAt:   t.UpdateTime.Unix(),
		})
	}

	return &pb.GetUserTagsResponse{
		Tags: respTags,
	}, nil
}
