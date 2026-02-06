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

type GetGroupInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetGroupInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetGroupInfoLogic {
	return &GetGroupInfoLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetGroupInfo 获取群聊信息
func (l *GetGroupInfoLogic) GetGroupInfo(in *chat.GetGroupInfoReq) (*chat.GetGroupInfoResp, error) {
	// 1. 参数验证
	if in.GroupId == "" {
		return nil, status.Error(codes.InvalidArgument, "群聊ID不能为空")
	}

	// 2. 查询群聊信息
	group, err := l.svcCtx.GroupModel.FindOne(l.ctx, in.GroupId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "群聊不存在")
		}
		l.Errorf("查询群聊信息失败: %v", err)
		return nil, status.Error(codes.Internal, "查询群聊信息失败")
	}

	// 3. 构造响应
	return &chat.GetGroupInfoResp{
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
