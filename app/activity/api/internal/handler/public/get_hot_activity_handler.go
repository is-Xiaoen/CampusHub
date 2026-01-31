// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package public

import (
	"net/http"

	"activity-platform/app/activity/api/internal/logic/public"
	"activity-platform/app/activity/api/internal/svc"
	"activity-platform/app/activity/api/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 热门活动
func GetHotActivityHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetHotActivityReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := public.NewGetHotActivityLogic(r.Context(), svcCtx)
		resp, err := l.GetHotActivity(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
