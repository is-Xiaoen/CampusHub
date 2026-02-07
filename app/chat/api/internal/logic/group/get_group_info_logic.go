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
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func (l *GetGroupInfoLogic) GetGroupInfo(req *types.GetGroupInfoReq) (resp *types.GroupInfo, err error) {
	// 调用 RPC 服务获取群组信息
	rpcResp, err := l.svcCtx.ChatRpc.GetGroupInfo(l.ctx, &chat.GetGroupInfoReq{
		GroupId: req.GroupId,
	})
	if err != nil {
		l.Errorf("调用 RPC 获取群组信息失败: %v", err)
		// 处理 gRPC 错误
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.NotFound:
				return nil, errorx.New(errorx.CodeGroupNotFound)
			case codes.PermissionDenied:
				return nil, errorx.New(errorx.CodeGroupPermissionDenied)
			default:
				return nil, errorx.NewWithMessage(errorx.CodeRPCError, "获取群组信息失败")
			}
		}
		return nil, errorx.NewWithMessage(errorx.CodeInternalError, "获取群组信息失败")
	}

	// 转换响应数据
	return &types.GroupInfo{
		GroupId:     rpcResp.Group.GroupId,
		ActivityId:  mustParseInt64(rpcResp.Group.ActivityId),
		Name:        rpcResp.Group.Name,
		OwnerId:     mustParseInt64(rpcResp.Group.OwnerId),
		MemberCount: rpcResp.Group.MemberCount,
		Status:      rpcResp.Group.Status,
		CreatedAt:   formatTimestamp(rpcResp.Group.CreatedAt),
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
