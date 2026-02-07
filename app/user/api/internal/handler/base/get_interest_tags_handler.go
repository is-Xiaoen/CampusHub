// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package base

import (
	"net/http"

	"activity-platform/app/user/api/internal/logic/base"
	"activity-platform/app/user/api/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 获取所有的兴趣标签
func GetInterestTagsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := base.NewGetInterestTagsLogic(r.Context(), svcCtx)
		resp, err := l.GetInterestTags()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
