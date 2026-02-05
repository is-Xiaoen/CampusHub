// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package base

import (
	"net/http"

	"activity-platform/app/user/api/internal/logic/base"
	"activity-platform/app/user/api/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 获取注册QQ验证码
func GetRegisterCodeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := base.NewGetRegisterCodeLogic(r.Context(), svcCtx)
		err := l.GetRegisterCode()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.Ok(w)
		}
	}
}
