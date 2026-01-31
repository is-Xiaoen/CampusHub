// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package credit

import (
	"net/http"

	"activity-platform/app/user/api/internal/logic/credit"
	"activity-platform/app/user/api/internal/svc"
	"activity-platform/app/user/api/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 查询信用变更记录
func GetCreditLogsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetCreditLogsReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := credit.NewGetCreditLogsLogic(r.Context(), svcCtx)
		resp, err := l.GetCreditLogs(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
