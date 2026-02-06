// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package group

import (
	"context"
	"fmt"

	"activity-platform/app/chat/api/internal/svc"
	"activity-platform/app/chat/api/internal/types"
	"activity-platform/app/chat/rpc/chat"

	"github.com/zeromicro/go-zero/core/logx"
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

func (l *GetGroupMembersLogic) GetGroupMembers(req *types.GetGroupMembersReq) (resp *types.GetGroupMembersResp, err error) {
	// 调用 RPC 服务获取群成员列表
	rpcResp, err := l.svcCtx.ChatRpc.GetGroupMembers(l.ctx, &chat.GetGroupMembersReq{
		GroupId:  req.GroupId,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		l.Errorf("调用 RPC 获取群成员列表失败: %v", err)
		return &types.GetGroupMembersResp{
			Code:    500,
			Message: fmt.Sprintf("获取群成员列表失败: %v", err),
			Data: types.GetGroupMembersData{
				Members:  []types.GroupMemberInfo{},
				Total:    0,
				Page:     req.Page,
				PageSize: req.PageSize,
			},
		}, nil
	}

	// 转换成员列表
	members := make([]types.GroupMemberInfo, 0, len(rpcResp.Members))
	for _, member := range rpcResp.Members {
		members = append(members, types.GroupMemberInfo{
			UserId:   mustParseInt64(member.UserId),
			Username: "", // TODO: 需要调用 UserRpc 获取用户信息
			Avatar:   "", // TODO: 需要调用 UserRpc 获取用户信息
			Role:     getRoleString(member.Role),
			JoinedAt: formatTimestamp(member.JoinedAt),
		})
	}

	return &types.GetGroupMembersResp{
		Code:    0,
		Message: "success",
		Data: types.GetGroupMembersData{
			Members:  members,
			Total:    rpcResp.Total,
			Page:     req.Page,
			PageSize: req.PageSize,
		},
	}, nil
}
