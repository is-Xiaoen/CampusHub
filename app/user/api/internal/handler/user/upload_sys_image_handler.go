// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"net/http"

	"activity-platform/app/user/api/internal/logic/user"
	"activity-platform/app/user/api/internal/svc"
	"activity-platform/app/user/api/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 上传通用系统图片
func UploadSysImageHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.UploadSysImageReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		l := user.NewUploadSysImageLogic(r.Context(), svcCtx, r)
		resp, err := l.UploadSysImage(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
