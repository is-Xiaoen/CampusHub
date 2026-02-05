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

type ListTagLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 标签列表
func NewListTagLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListTagLogic {
	return &ListTagLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListTagLogic) ListTag(req *types.ListTagReq) (resp *types.ListTagResp, err error) {
	// 调用 RPC 服务
	// limit = 0 表示获取全部，limit > 0 表示获取热门 N 个
	rpcResp, err := l.svcCtx.ActivityRpc.ListTags(l.ctx, &activityservice.ListTagsReq{
		Limit: req.Limit,
	})
	if err != nil {
		l.Errorf("RPC ListTags failed: limit=%d, err=%v", req.Limit, err)
		return nil, errorx.FromError(err)
	}

	// 转换响应类型
	return &types.ListTagResp{
		List: logic.ConvertRpcTagsToApiTags(rpcResp.List),
	}, nil
}
