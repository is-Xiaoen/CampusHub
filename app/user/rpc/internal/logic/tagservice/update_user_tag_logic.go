package tagservicelogic

import (
	"context"

	"activity-platform/app/user/model"
	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UpdateUserTagLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateUserTagLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateUserTagLogic {
	return &UpdateUserTagLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 修改用户兴趣
func (l *UpdateUserTagLogic) UpdateUserTag(in *pb.UpdateUserTagReq) (*pb.UpdateUserTagResponse, error) {
	// 1. 删除用户所有旧标签关联
	err := l.svcCtx.UserInterestRelationModel.DeleteByUserID(l.ctx, in.UserId)
	if err != nil {
		l.Logger.Errorf("DeleteByUserID error: %v, userId: %d", err, in.UserId)
		return nil, status.Error(codes.Internal, "failed to delete old tags")
	}

	// 2. 如果没有新标签，直接返回空
	if len(in.Ids) == 0 {
		return &pb.UpdateUserTagResponse{
			Tags: []*pb.TagBasicInfo{},
		}, nil
	}

	// 3. 构建新标签关联
	var relations []*model.UserInterestRelation
	for _, tagID := range in.Ids {
		relations = append(relations, &model.UserInterestRelation{
			UserID: in.UserId,
			TagID:  tagID,
		})
	}

	// 4. 批量插入新关联
	err = l.svcCtx.UserInterestRelationModel.BatchCreate(l.ctx, relations)
	if err != nil {
		l.Logger.Errorf("BatchCreate relations error: %v, userId: %d", err, in.UserId)
		return nil, status.Error(codes.Internal, "failed to add new tags")
	}

	// 5. 查询新标签详情以返回
	tags, err := l.svcCtx.InterestTagModel.FindByIDs(l.ctx, in.Ids)
	if err != nil {
		l.Logger.Errorf("FindByIDs error: %v, tagIds: %v", err, in.Ids)
		return nil, status.Error(codes.Internal, "failed to retrieve tag info")
	}

	// 6. 转换为 Proto 响应格式
	var pbTags []*pb.TagBasicInfo
	for _, tag := range tags {
		pbTags = append(pbTags, &pb.TagBasicInfo{
			Id:          uint64(tag.TagID),
			Name:        tag.TagName,
			Color:       tag.Color,
			Icon:        tag.Icon,
			Description: tag.TagDesc,
		})
	}

	return &pb.UpdateUserTagResponse{
		Tags: pbTags,
	}, nil
}
