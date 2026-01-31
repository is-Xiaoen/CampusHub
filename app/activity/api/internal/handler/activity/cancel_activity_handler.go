// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package activity

import (
	"net/http"

	"activity-platform/app/activity/api/internal/logic/activity"
	"activity-platform/app/activity/api/internal/svc"
	"activity-platform/app/activity/api/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 取消活动
func CancelActivityHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.CancelActivityReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := activity.NewCancelActivityLogic(r.Context(), svcCtx)
		resp, err := l.CancelActivity(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
