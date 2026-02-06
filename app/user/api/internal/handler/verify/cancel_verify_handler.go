// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package verify

import (
	"net/http"

	"activity-platform/app/user/api/internal/logic/verify"
	"activity-platform/app/user/api/internal/svc"
	"activity-platform/app/user/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// CancelVerifyHandler 取消认证申请
func CancelVerifyHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.CancelVerifyReq
		if err := httpx.Parse(r, &req); err != nil {
			logx.WithContext(r.Context()).Errorf("[CancelVerifyHandler] 参数解析失败: err=%v", err)
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		logx.WithContext(r.Context()).Infof("[CancelVerifyHandler] 收到请求: verifyId=%d", req.VerifyId)

		l := verify.NewCancelVerifyLogic(r.Context(), svcCtx)
		resp, err := l.CancelVerify(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
