// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package group

import (
	"net/http"

	"activity-platform/app/chat/api/internal/logic/group"
	"activity-platform/app/chat/api/internal/svc"
	"activity-platform/app/chat/api/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 获取用户的群聊列表
func GetUserGroupsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetUserGroupsReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := group.NewGetUserGroupsLogic(r.Context(), svcCtx)
		resp, err := l.GetUserGroups(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
