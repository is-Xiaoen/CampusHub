// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package group

import (
	"context"
	"fmt"
	"strconv"

	"activity-platform/app/chat/api/internal/svc"
	"activity-platform/app/chat/api/internal/types"
	"activity-platform/app/chat/rpc/chat"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetGroupInfoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewGetGroupInfoLogic 查询群组信息
func NewGetGroupInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetGroupInfoLogic {
	return &GetGroupInfoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetGroupInfoLogic) GetGroupInfo(req *types.GetGroupInfoReq) (resp *types.GetGroupInfoResp, err error) {
	// 调用 RPC 服务获取群组信息
	rpcResp, err := l.svcCtx.ChatRpc.GetGroupInfo(l.ctx, &chat.GetGroupInfoReq{
		GroupId: req.GroupId,
	})
	if err != nil {
		l.Errorf("调用 RPC 获取群组信息失败: %v", err)
		return &types.GetGroupInfoResp{
			Code:    500,
			Message: fmt.Sprintf("获取群组信息失败: %v", err),
			Data:    types.GroupInfo{},
		}, nil
	}

	// 转换响应数据
	return &types.GetGroupInfoResp{
		Code:    0,
		Message: "success",
		Data: types.GroupInfo{
			GroupId:     rpcResp.Group.GroupId,
			ActivityId:  mustParseInt64(rpcResp.Group.ActivityId),
			Name:        rpcResp.Group.Name,
			OwnerId:     mustParseInt64(rpcResp.Group.OwnerId),
			MemberCount: rpcResp.Group.MemberCount,
			Status:      rpcResp.Group.Status,
			CreatedAt:   formatTimestamp(rpcResp.Group.CreatedAt),
		},
	}, nil
}

// mustParseInt64 将字符串转换为 int64，失败返回 0
func mustParseInt64(s string) int64 {
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}

// formatTimestamp 将时间戳转换为字符串格式
func formatTimestamp(timestamp int64) string {
	if timestamp == 0 {
		return ""
	}
	return fmt.Sprintf("%d", timestamp)
}

// getRoleString 将角色代码转换为字符串
func getRoleString(role int32) string {
	switch role {
	case 1:
		return "member"
	case 2:
		return "owner"
	default:
		return "member"
	}
}
