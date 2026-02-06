/**
 * @projectName: CampusHub
 * @package: credit
 * @className: GetCreditLogsLogic
 * @author: lijunqi
 * @description: 查询信用变更记录业务逻辑
 * @date: 2026-01-30
 * @version: 1.0
 */

package credit

import (
	"context"

	"activity-platform/app/user/api/internal/svc"
	"activity-platform/app/user/api/internal/types"
	"activity-platform/app/user/rpc/client/creditservice"
	"activity-platform/common/ctxdata"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

// GetCreditLogsLogic 查询信用变更记录逻辑
type GetCreditLogsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewGetCreditLogsLogic 创建查询信用变更记录逻辑实例
func NewGetCreditLogsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCreditLogsLogic {
	return &GetCreditLogsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// GetCreditLogs 查询信用变更记录
func (l *GetCreditLogsLogic) GetCreditLogs(req *types.GetCreditLogsReq) (resp *types.GetCreditLogsResp, err error) {
	// 1. 从 JWT 中获取当前用户ID
	userId := ctxdata.GetUserIDFromCtx(l.ctx)

	// 1.1 参数校验
	if userId <= 0 {
		l.Errorf("GetCreditLogs 参数错误: userId=%d", userId)
		return nil, errorx.ErrUnauthorized()
	}

	// 2. 调用 RPC 查询信用变更记录
	rpcResp, err := l.svcCtx.CreditServiceRpc.GetCreditLogs(l.ctx, &creditservice.GetCreditLogsReq{
		UserId:     userId,
		ChangeType: req.ChangeType,
		StartTime:  req.StartTime,
		EndTime:    req.EndTime,
		Page:       req.Page,
		PageSize:   req.PageSize,
	})
	if err != nil {
		l.Errorf("调用 CreditServiceRpc.GetCreditLogs 失败: userId=%d, err=%v", userId, err)
		return nil, errorx.FromError(err)
	}

	// 3. 转换 RPC 响应为 API 响应
	list := make([]types.CreditLogItem, 0, len(rpcResp.List))
	for _, item := range rpcResp.List {
		list = append(list, types.CreditLogItem{
			Id:             item.Id,
			UserId:         item.UserId,
			ChangeType:     item.ChangeType,
			ChangeTypeName: item.ChangeTypeName,
			Delta:          item.Delta,
			SourceId:       item.SourceId,
			Reason:         item.Reason,
			CreatedAt:      item.CreatedAt,
		})
	}

	l.Infof("GetCreditLogs 查询成功: userId=%d, total=%d", userId, rpcResp.Total)

	return &types.GetCreditLogsResp{
		List:     list,
		Total:    rpcResp.Total,
		Page:     rpcResp.Page,
		PageSize: rpcResp.PageSize,
	}, nil
}
