// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package ticket

import (
	"net/http"

	"activity-platform/app/activity/api/internal/logic/ticket"
	"activity-platform/app/activity/api/internal/svc"
	"activity-platform/app/activity/api/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 获取个人票券列表
func GetTicketListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetTicketListRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := ticket.NewGetTicketListLogic(r.Context(), svcCtx)
		resp, err := l.GetTicketList(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
