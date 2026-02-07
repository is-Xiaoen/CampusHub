// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package group

import (
	"context"

	"activity-platform/app/chat/api/internal/svc"
	"activity-platform/app/chat/api/internal/types"
	"activity-platform/app/chat/rpc/chat"
	"activity-platform/app/user/rpc/pb/pb"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GetGroupMembersLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewGetGroupMembersLogic 查询群成员列表
func NewGetGroupMembersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetGroupMembersLogic {
	return &GetGroupMembersLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetGroupMembersLogic) GetGroupMembers(req *types.GetGroupMembersReq) (resp *types.GetGroupMembersData, err error) {
	// 调用 RPC 服务获取群成员列表
	rpcResp, err := l.svcCtx.ChatRpc.GetGroupMembers(l.ctx, &chat.GetGroupMembersReq{
		GroupId:  req.GroupId,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		l.Errorf("调用 RPC 获取群成员列表失败: %v", err)
		// 处理 gRPC 错误
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.NotFound:
				return nil, errorx.New(errorx.CodeGroupNotFound)
			case codes.PermissionDenied:
				return nil, errorx.New(errorx.CodeGroupPermissionDenied)
			default:
				return nil, errorx.NewWithMessage(errorx.CodeRPCError, "获取群成员列表失败")
			}
		}
		return nil, errorx.NewWithMessage(errorx.CodeInternalError, "获取群成员列表失败")
	}

	// 收集所有用户ID
	userIds := make([]int64, 0, len(rpcResp.Members))
	for _, member := range rpcResp.Members {
		userIds = append(userIds, mustParseInt64(member.UserId))
	}

	// 批量调用 UserRpc 获取用户信息
	var userInfoMap map[int64]*pb.GroupUserInfo
	if len(userIds) > 0 {
		userResp, err := l.svcCtx.UserRpc.GetGroupUser(l.ctx, &pb.GetGroupUserReq{
			Ids: userIds,
		})
		if err != nil {
			l.Errorf("调用 UserRpc 获取用户信息失败: %v", err)
			// 不中断流程，继续返回成员列表，只是用户名和头像为空
		} else {
			// 构建用户信息映射表，方便后续查找
			userInfoMap = make(map[int64]*pb.GroupUserInfo, len(userResp.Users))
			for _, user := range userResp.Users {
				userInfoMap[int64(user.Id)] = user
			}
		}
	}

	// 转换成员列表
	members := make([]types.GroupMemberInfo, 0, len(rpcResp.Members))
	for _, member := range rpcResp.Members {
		userId := mustParseInt64(member.UserId)
		username := ""
		avatar := ""

		// 从映射表中获取用户信息
		if userInfo, ok := userInfoMap[userId]; ok {
			username = userInfo.Nickname
			avatar = userInfo.AvatarUrl
		}

		members = append(members, types.GroupMemberInfo{
			UserId:   userId,
			Username: username,
			Avatar:   avatar,
			Role:     getRoleString(member.Role),
			JoinedAt: formatTimestamp(member.JoinedAt),
		})
	}

	return &types.GetGroupMembersData{
		Members:  members,
		Total:    rpcResp.Total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}
