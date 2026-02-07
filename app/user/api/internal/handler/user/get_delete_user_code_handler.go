// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"net/http"

	"activity-platform/app/user/api/internal/logic/user"
	"activity-platform/app/user/api/internal/svc"
	"activity-platform/common/response"
)

// 获取注销用户QQ验证码
func GetDeleteUserCodeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := user.NewGetDeleteUserCodeLogic(r.Context(), svcCtx)
		err := l.GetDeleteUserCode()
		if err != nil {
			response.Fail(w, err)
		} else {
			response.Success(w, nil)
		}
	}
}
