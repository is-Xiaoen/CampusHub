package userbasicservicelogic

import (
	"context"
	"strconv"
	"strings"

	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetUserInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserInfoLogic {
	return &GetUserInfoLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取用户的信息
func (l *GetUserInfoLogic) GetUserInfo(in *pb.GetUserInfoReq) (*pb.GetUserInfoResponse, error) {
	// 1. 查询用户基础信息
	user, err := l.svcCtx.UserModel.FindByUserID(l.ctx, int64(in.UserId))
	if err != nil {
		if strings.Contains(err.Error(), "record not found") {
			return nil, errorx.New(errorx.CodeUserNotFound)
		}
		return nil, errorx.NewSystemError("系统繁忙，请稍后再试")
	}

	// 2. 获取附加信息
	// 2.1 真实的学生认证状态 (通过StudentVerificationModel)
	// StudentVerificationModel.IsVerified
	isVerified, err := l.svcCtx.StudentVerificationModel.IsVerified(l.ctx, user.UserID)
	if err != nil {
		l.Logger.Errorf("Check IsVerified failed: %v", err)
		isVerified = false
	}

	// 2.2 活动统计 (ActivityRpc，可选依赖)
	var activitiesNum, initiateNum uint32
	if l.svcCtx.ActivityRpc != nil {
		registeredCountResp, err := l.svcCtx.ActivityRpc.GetRegisteredCount(l.ctx, &activity.GetRegisteredCountRequest{
			UserId: user.UserID,
		})
		if err == nil {
			activitiesNum = uint32(registeredCountResp.Count)
		} else {
			l.Logger.Errorf("GetRegisteredCount failed: %v", err)
		}

		publishedResp, err := l.svcCtx.ActivityRpc.GetUserPublishedActivities(l.ctx, &activity.GetUserPublishedActivitiesReq{
			UserId:   user.UserID,
			Page:     1,
			PageSize: 1, // 只需要总数
		})
		if err == nil && publishedResp.Pagination != nil {
			initiateNum = uint32(publishedResp.Pagination.Total)
		} else {
			l.Logger.Errorf("GetUserPublishedActivities failed: %v", err)
		}
	} else {
		l.Logger.Infof("ActivityRpc 不可用，跳过活动统计")
	}

	// 2.3 兴趣标签 (Database)
	relations, err := l.svcCtx.UserInterestRelationModel.ListByUserID(l.ctx, user.UserID)
	var interestTags []*pb.InterestTag
	if err == nil {
		for _, rel := range relations {
			tag, err := l.svcCtx.InterestTagModel.FindByID(l.ctx, rel.TagID)
			if err == nil && tag != nil {
				interestTags = append(interestTags, &pb.InterestTag{
					Id:       uint64(tag.TagID),
					TagName:  tag.TagName,
					TagColor: tag.Color,
					TagIcon:  tag.Icon,
				})
			}
		}
	}

	// 3. 组装响应
	var genderStr string
	switch user.Gender {
	case 1:
		genderStr = "男"
	case 2:
		genderStr = "女"
	default:
		genderStr = "未知"
	}

	return &pb.GetUserInfoResponse{
		UserInfo: &pb.UserInfo{
			UserId:            uint64(user.UserID),
			Nickname:          user.Nickname,
			AvatarUrl:         user.AvatarURL,
			Introduction:      user.Introduction,
			Gender:            genderStr,
			Age:               strconv.FormatInt(user.Age, 10),
			ActivitiesNum:     activitiesNum,
			IsStudentVerified: isVerified,
			InitiateNum:       initiateNum,
			InterestTags:      interestTags,
		},
	}, nil
}
