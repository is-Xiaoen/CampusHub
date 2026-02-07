// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package message

import (
	"context"

	"activity-platform/app/chat/api/internal/svc"
	"activity-platform/app/chat/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMessageHistoryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 查询消息历史
func NewGetMessageHistoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMessageHistoryLogic {
	return &GetMessageHistoryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetMessageHistoryLogic) GetMessageHistory(req *types.GetMessageHistoryReq) (resp *types.GetMessageHistoryResp, err error) {
	// todo: add your logic here and delete this line

	return
}
