package public

import (
	"context"

	"activity-platform/app/activity/api/internal/logic"
	"activity-platform/app/activity/api/internal/svc"
	"activity-platform/app/activity/api/internal/types"
	"activity-platform/app/activity/rpc/activityservice"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListCategoryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 分类列表
func NewListCategoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListCategoryLogic {
	return &ListCategoryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListCategoryLogic) ListCategory(req *types.ListCategoryReq) (resp *types.ListCategoryResp, err error) {
	// 调用 RPC 服务
	rpcResp, err := l.svcCtx.ActivityRpc.ListCategories(l.ctx, &activityservice.ListCategoriesReq{})
	if err != nil {
		l.Errorf("RPC ListCategories failed: %v", err)
		return nil, errorx.FromError(err)
	}

	// 转换响应类型
	return &types.ListCategoryResp{
		List: logic.ConvertRpcCategoriesToApiCategories(rpcResp.List),
	}, nil
}
