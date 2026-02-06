package logic

import (
	"context"
	"errors"

	"activity-platform/app/chat/rpc/chat"
	"activity-platform/app/chat/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type GetGroupByActivityIdLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetGroupByActivityIdLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetGroupByActivityIdLogic {
	return &GetGroupByActivityIdLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetGroupByActivityId 通过活动ID获取群聊
func (l *GetGroupByActivityIdLogic) GetGroupByActivityId(in *chat.GetGroupByActivityIdReq) (*chat.GetGroupByActivityIdResp, error) {
	// 1. 参数验证
	if in.ActivityId == "" {
		return nil, status.Error(codes.InvalidArgument, "活动ID不能为空")
	}

	// 2. 查询群聊
	group, err := l.svcCtx.GroupModel.FindByActivityID(l.ctx, in.ActivityId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "群聊不存在")
		}
		l.Errorf("查询群聊失败: %v", err)
		return nil, status.Error(codes.Internal, "查询群聊失败")
	}

	// 3. 返回结果
	return &chat.GetGroupByActivityIdResp{
		Group: &chat.GroupInfo{
			GroupId:     group.GroupID,
			ActivityId:  group.ActivityID,
			Name:        group.Name,
			OwnerId:     group.OwnerID,
			Status:      int32(group.Status),
			MaxMembers:  group.MaxMembers,
			MemberCount: group.MemberCount,
			CreatedAt:   group.CreatedAt.Unix(),
		},
	}, nil
}
