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

type SearchActivityLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 搜索活动
func NewSearchActivityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SearchActivityLogic {
	return &SearchActivityLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SearchActivityLogic) SearchActivity(req *types.SearchActivityReq) (resp *types.SearchActivityResp, err error) {
	// 1. 参数校验
	keywordLen := len([]rune(req.Keyword))
	if keywordLen < 2 {
		return nil, errorx.ErrInvalidParams("搜索关键词至少2个字符")
	}
	if keywordLen > 50 {
		return nil, errorx.ErrInvalidParams("搜索关键词不能超过50个字符")
	}

	// 分页参数校验
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	if req.PageSize > 50 {
		req.PageSize = 50
	}

	// 2. 调用 RPC 服务
	rpcResp, err := l.svcCtx.ActivityRpc.SearchActivities(l.ctx, &activityservice.SearchActivitiesReq{
		Keyword:    req.Keyword,
		CategoryId: req.CategoryId,
		Page:       req.Page,
		PageSize:   req.PageSize,
		Sort:       req.Sort,
	})
	if err != nil {
		l.Errorf("RPC SearchActivities failed: keyword=%s, err=%v", req.Keyword, err)
		return nil, errorx.FromError(err)
	}

	// 3. 转换响应类型
	return &types.SearchActivityResp{
		List:        logic.ConvertRpcActivityListItemsToApi(rpcResp.List),
		Total:       rpcResp.Total,
		QueryTimeMs: rpcResp.QueryTimeMs,
	}, nil
}
