/**
 * @projectName: CampusHub
 * @package: verify
 * @className: GetVerifyCurrentLogic
 * @author: lijunqi
 * @description: 获取当前认证进度业务逻辑
 * @date: 2026-02-02
 * @version: 1.0
 */

package verify

import (
	"context"
	"time"

	"activity-platform/app/user/api/internal/svc"
	"activity-platform/app/user/api/internal/types"
	"activity-platform/app/user/rpc/client/verifyservice"
	"activity-platform/common/ctxdata"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

// GetVerifyCurrentLogic 获取当前认证进度逻辑处理器
type GetVerifyCurrentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewGetVerifyCurrentLogic 创建获取当前认证进度逻辑实例
func NewGetVerifyCurrentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetVerifyCurrentLogic {
	return &GetVerifyCurrentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// GetVerifyCurrent 获取当前认证进度
func (l *GetVerifyCurrentLogic) GetVerifyCurrent() (resp *types.GetVerifyCurrentResp, err error) {
	// 1. 从 JWT 中获取当前用户ID
	userId := ctxdata.GetUserIDFromCtx(l.ctx)
	if userId <= 0 {
		l.Errorf("GetVerifyCurrent 获取用户ID失败")
		return nil, errorx.ErrUnauthorized()
	}

	// 2. 调用 RPC 获取当前认证进度
	rpcResp, err := l.svcCtx.VerifyServiceRpc.GetVerifyCurrent(l.ctx, &verifyservice.GetVerifyCurrentReq{
		UserId: userId,
	})
	if err != nil {
		l.Errorf("GetVerifyCurrent 调用 RPC 失败: userId=%d, err=%v", userId, err)
		return nil, errorx.FromError(err)
	}

	// 3. 转换 RPC 响应为 API 响应
	resp = &types.GetVerifyCurrentResp{
		HasRecord:    rpcResp.HasRecord,
		VerifyId:     rpcResp.VerifyId,
		Status:       rpcResp.Status,
		StatusDesc:   rpcResp.StatusDesc,
		CanApply:     rpcResp.CanApply,
		CanConfirm:   rpcResp.CanConfirm,
		CanCancel:    rpcResp.CanCancel,
		NeedAction:   rpcResp.NeedAction,
		RejectReason: rpcResp.RejectReason,
	}

	// 4. 转换时间戳为字符串格式
	if rpcResp.CreatedAt > 0 {
		resp.CreatedAt = time.Unix(rpcResp.CreatedAt, 0).Format(time.RFC3339)
	}
	if rpcResp.UpdatedAt > 0 {
		resp.UpdatedAt = time.Unix(rpcResp.UpdatedAt, 0).Format(time.RFC3339)
	}

	// 5. 转换 OCR 识别数据
	if rpcResp.VerifyData != nil {
		resp.VerifyData = &types.VerifyData{
			RealName:      rpcResp.VerifyData.RealName,
			SchoolName:    rpcResp.VerifyData.SchoolName,
			StudentId:     rpcResp.VerifyData.StudentId,
			Department:    rpcResp.VerifyData.Department,
			AdmissionYear: rpcResp.VerifyData.AdmissionYear,
		}
		// 状态为"已通过"(4)时，UpdatedAt 即为认证通过时间
		if rpcResp.Status == 4 && rpcResp.UpdatedAt > 0 {
			resp.VerifyData.VerifiedAt = time.Unix(rpcResp.UpdatedAt, 0).Format(time.RFC3339)
		}
	}

	l.Infof("GetVerifyCurrent 查询成功: userId=%d, hasRecord=%v, status=%d, needAction=%s",
		userId, resp.HasRecord, resp.Status, resp.NeedAction)

	return resp, nil
}
