package userbasicservicelogic

import (
	"context"

	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetGroupUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetGroupUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetGroupUserLogic {
	return &GetGroupUserLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 批量获取群聊用户的信息
func (l *GetGroupUserLogic) GetGroupUser(in *pb.GetGroupUserReq) (*pb.GetGroupUserResponse, error) {
	if len(in.Ids) == 0 {
		return &pb.GetGroupUserResponse{}, nil
	}

	users, err := l.svcCtx.UserModel.FindByIDs(l.ctx, in.Ids)
	if err != nil {
		l.Logger.Errorf("FindByIDs failed: %v", err)
		return nil, errorx.ErrDBError(err)
	}

	var respUsers []*pb.GroupUserInfo
	for _, u := range users {
		var avatarUrl string
		if u.AvatarID > 0 {
			imgResp, err := NewGetSysImageLogic(l.ctx, l.svcCtx).GetSysImage(&pb.GetSysImageReq{
				UserId:  u.UserID,
				ImageId: u.AvatarID,
			})
			if err == nil {
				avatarUrl = imgResp.Url
			} else {
				l.Logger.Errorf("GetSysImage failed for group user %d: %v", u.UserID, err)
			}
		}
		respUsers = append(respUsers, &pb.GroupUserInfo{
			Id:        uint64(u.UserID),
			Nickname:  u.Nickname,
			AvatarUrl: avatarUrl,
		})
	}

	return &pb.GetGroupUserResponse{
		Users: respUsers,
	}, nil
}
