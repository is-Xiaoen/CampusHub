// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package message

import (
	"net/http"

	"activity-platform/app/chat/api/internal/logic/message"
	"activity-platform/app/chat/api/internal/svc"
	"activity-platform/app/chat/api/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// GetMessageHistoryHandler 查询消息历史
func GetMessageHistoryHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetMessageHistoryReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := message.NewGetMessageHistoryLogic(r.Context(), svcCtx)
		resp, err := l.GetMessageHistory(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
