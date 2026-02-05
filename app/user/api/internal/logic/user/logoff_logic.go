// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"context"

	"activity-platform/app/user/api/internal/svc"
	"activity-platform/app/user/api/internal/types"
	"activity-platform/app/user/rpc/client/userbasicservice"
	ctxUtils "activity-platform/common/utils/context"

	"github.com/zeromicro/go-zero/core/logx"
)

type LogoffLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 注销用户
func NewLogoffLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LogoffLogic {
	return &LogoffLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LogoffLogic) Logoff(req *types.LogoffReq) (resp *types.LogoffResp, err error) {
	userId, err := ctxUtils.GetUserIdFromCtx(l.ctx)
	if err != nil {
		return nil, err
	}

	rpcResp, err := l.svcCtx.UserBasicServiceRpc.DeleteUser(l.ctx, &userbasicservice.DeleteUserReq{
		UserId: userId,
		QqCode: req.QqCode,
	})
	if err != nil {
		return nil, err
	}

	return &types.LogoffResp{
		Success: rpcResp.Success,
	}, nil
}
