// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package verify

import (
	"net/http"

	"activity-platform/app/user/api/internal/logic/verify"
	"activity-platform/app/user/api/internal/svc"
	"activity-platform/app/user/api/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 提交学生认证申请
func ApplyVerifyHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ApplyVerifyReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := verify.NewApplyVerifyLogic(r.Context(), svcCtx)
		resp, err := l.ApplyVerify(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
