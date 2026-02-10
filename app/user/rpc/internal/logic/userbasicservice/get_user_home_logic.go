package userbasicservicelogic

import (
	"context"
	"time"

	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/user/model"
	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/mr"
	"gorm.io/gorm"
)

type GetUserHomeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetUserHomeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserHomeLogic {
	return &GetUserHomeLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取用户主页信息
func (l *GetUserHomeLogic) GetUserHome(in *pb.GetUserHomeReq) (*pb.GetUserHomeResp, error) {
	resp := &pb.GetUserHomeResp{
		UserInfo:            &pb.UserHomeInfo{},
		Tags:                make([]*pb.UserHomeTag, 0),
		JoinedActivities:    &pb.UserHomeActivityList{List: make([]*pb.UserHomeActivityItem, 0)},
		PublishedActivities: &pb.UserHomeActivityList{List: make([]*pb.UserHomeActivityItem, 0)},
	}

	// 并行处理：获取用户信息+标签、获取参加活动、获取发起活动
	err := mr.Finish(func() error {
		// 1. 获取用户信息和标签
		user, err := l.svcCtx.UserModel.FindByUserID(l.ctx, in.UserId)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return errorx.New(errorx.CodeUserNotFound)
			}
			l.Logger.Errorf("Failed to find user: %v", err)
			return errorx.New(errorx.CodeDBError)
		}
		if user.Status == model.UserStatusDeleted {
			return errorx.New(errorx.CodeUserNotFound)
		}

		// 填充用户信息
		resp.UserInfo = &pb.UserHomeInfo{
			UserId:       user.UserID,
			Nickname:     user.Nickname,
			AvatarUrl:    user.AvatarURL,
			Introduction: user.Introduction,
			Gender:       int32(user.Gender),
			Age:          int32(user.Age),
		}

		// 获取用户标签
		relations, err := l.svcCtx.UserInterestRelationModel.ListByUserID(l.ctx, user.UserID)
		if err != nil {
			l.Logger.Errorf("Failed to list user interest relations: %v", err)
			// 降级：不返回标签，但不报错
			return nil
		}

		if len(relations) > 0 {
			tagIDs := make([]int64, len(relations))
			for i, r := range relations {
				tagIDs[i] = r.TagID
			}

			tags, err := l.svcCtx.InterestTagModel.FindByIDs(l.ctx, tagIDs)
			if err != nil {
				l.Logger.Errorf("Failed to find interest tags: %v", err)
			} else {
				for _, tag := range tags {
					if tag.Status == model.TagStatusNormal {
						resp.Tags = append(resp.Tags, &pb.UserHomeTag{
							TagId:   tag.TagID,
							TagName: tag.TagName,
							Color:   tag.Color,
							Icon:    tag.Icon,
							TagDesc: tag.TagDesc,
						})
					}
				}
			}
		}
		return nil
	}, func() error {
		// 2. 获取参加活动列表
		if l.svcCtx.ActivityRpc == nil {
			l.Logger.Error("ActivityRpc client is nil")
			return nil
		}

		// 处理分页默认值
		page := in.JoinedPage
		if page <= 0 {
			page = 1
		}
		pageSize := in.JoinedPageSize
		if pageSize <= 0 {
			pageSize = 10
		}

		actResp, err := l.svcCtx.ActivityRpc.GetActivityList(l.ctx, &activity.GetActivityListRequest{
			Page:     page,
			PageSize: pageSize,
			Type:     in.JoinedType,
			UserId:   in.UserId,
		})
		if err != nil {
			l.Logger.Errorf("Failed to get joined activity list: %v", err)
			return nil // 降级
		}

		resp.JoinedActivities.Total = actResp.Total
		for _, item := range actResp.Items {
			resp.JoinedActivities.List = append(resp.JoinedActivities.List, &pb.UserHomeActivityItem{
				Id:       item.Id,
				Name:     item.Name,
				Time:     item.Time,
				Status:   item.Status,
				ImageUrl: item.ImageUrl,
			})
		}
		return nil
	}, func() error {
		// 3. 获取发起活动列表
		if l.svcCtx.ActivityRpc == nil {
			return nil
		}

		// 处理分页默认值
		page := in.PublishedPage
		if page <= 0 {
			page = 1
		}
		pageSize := in.PublishedPageSize
		if pageSize <= 0 {
			pageSize = 10
		}

		pubResp, err := l.svcCtx.ActivityRpc.GetUserPublishedActivities(l.ctx, &activity.GetUserPublishedActivitiesReq{
			UserId:   in.UserId,
			Page:     page,
			PageSize: pageSize,
			Status:   in.PublishedStatus,
		})
		if err != nil {
			l.Logger.Errorf("Failed to get published activity list: %v", err)
			return nil // 降级
		}

		resp.PublishedActivities.Total = int32(pubResp.Pagination.Total)
		for _, item := range pubResp.List {
			timeStr := ""
			if item.ActivityStartTime > 0 {
				timeStr = time.Unix(item.ActivityStartTime, 0).Format("2006-01-02 15:04:05")
			}

			resp.PublishedActivities.List = append(resp.PublishedActivities.List, &pb.UserHomeActivityItem{
				Id:       item.Id,
				Name:     item.Title,
				Time:     timeStr,
				Status:   item.StatusText,
				ImageUrl: item.CoverUrl,
			})
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return resp, nil
}
