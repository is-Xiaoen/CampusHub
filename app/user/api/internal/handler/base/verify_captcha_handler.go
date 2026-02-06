// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package base

import (
	"net/http"

	"activity-platform/app/user/api/internal/logic/base"
	"activity-platform/app/user/api/internal/svc"
	"activity-platform/app/user/api/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 校验验证码
func VerifyCaptchaHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.VerifyCaptchaReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := base.NewVerifyCaptchaLogic(r.Context(), svcCtx)
		resp, err := l.VerifyCaptcha(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
