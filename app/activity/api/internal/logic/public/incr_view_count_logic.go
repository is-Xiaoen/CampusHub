package public

import (
	"context"
	"net/http"

	"activity-platform/app/activity/api/internal/svc"
	"activity-platform/app/activity/api/internal/types"
	"activity-platform/app/activity/rpc/activityservice"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

type IncrViewCountLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

// 增加浏览量
func NewIncrViewCountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *IncrViewCountLogic {
	return &IncrViewCountLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// WithRequest 注入 HTTP 请求（用于获取客户端 IP）
func (l *IncrViewCountLogic) WithRequest(r *http.Request) *IncrViewCountLogic {
	l.r = r
	return l
}

func (l *IncrViewCountLogic) IncrViewCount(req *types.IncrViewCountReq) (resp *types.IncrViewCountResp, err error) {
	// 1. 参数校验
	if req.Id <= 0 {
		return nil, errorx.ErrInvalidParams("活动ID无效")
	}

	// 2. 获取客户端 IP（用于防刷）
	clientIP := ""
	if l.r != nil {
		clientIP = httpx.GetRemoteAddr(l.r)
	}

	// 3. 调用 RPC 服务
	// 公开接口 user_id = 0，RPC 层会使用 IP 进行防刷
	rpcResp, err := l.svcCtx.ActivityRpc.IncrViewCount(l.ctx, &activityservice.IncrViewCountReq{
		Id:       req.Id,
		UserId:   0, // 公开接口无用户身份
		ClientIp: clientIP,
	})
	if err != nil {
		l.Errorf("RPC IncrViewCount failed: id=%d, err=%v", req.Id, err)
		return nil, errorx.FromError(err)
	}

	// 4. 返回响应
	return &types.IncrViewCountResp{
		ViewCount: rpcResp.ViewCount,
	}, nil
}
