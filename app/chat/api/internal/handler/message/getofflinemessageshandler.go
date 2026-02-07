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

// 获取离线消息
func GetOfflineMessagesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetOfflineMessagesReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := message.NewGetOfflineMessagesLogic(r.Context(), svcCtx)
		resp, err := l.GetOfflineMessages(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
