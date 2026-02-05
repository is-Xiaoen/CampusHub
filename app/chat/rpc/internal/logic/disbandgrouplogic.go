package logic

import (
	"context"

	"activity-platform/app/chat/rpc/chat"
	"activity-platform/app/chat/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type DisbandGroupLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDisbandGroupLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DisbandGroupLogic {
	return &DisbandGroupLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// DisbandGroup 解散群聊
func (l *DisbandGroupLogic) DisbandGroup(in *chat.DisbandGroupReq) (*chat.DisbandGroupResp, error) {
	// 1. 参数验证
	if in.GroupId == "" {
		return nil, status.Error(codes.InvalidArgument, "群聊ID不能为空")
	}
	if in.OperatorId == "" {
		return nil, status.Error(codes.InvalidArgument, "操作者ID不能为空")
	}

	// 2. 检查群聊是否存在
	group, err := l.svcCtx.GroupModel.FindOne(l.ctx, in.GroupId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Error(codes.NotFound, "群聊不存在")
		}
		l.Errorf("查询群聊失败: %v", err)
		return nil, status.Error(codes.Internal, "查询群聊失败")
	}

	// 3. 检查操作者是否为群主
	if group.OwnerID != in.OperatorId {
		return nil, status.Error(codes.PermissionDenied, "只有群主可以解散群聊")
	}

	// 4. 检查群聊是否已解散
	if group.Status == 2 {
		return nil, status.Error(codes.FailedPrecondition, "群聊已解散")
	}

	// 5. 更新群聊状态为已解散
	if err := l.svcCtx.GroupModel.UpdateStatus(l.ctx, in.GroupId, 2); err != nil {
		l.Errorf("解散群聊失败: %v", err)
		return nil, status.Error(codes.Internal, "解散群聊失败")
	}

	// 6. 返回结果
	return &chat.DisbandGroupResp{
		Success: true,
	}, nil
}
