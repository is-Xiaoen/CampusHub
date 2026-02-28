/**
 * @projectName: CampusHub
 * @package: verify
 * @className: ApplyVerifyHandler
 * @description: 提交学生认证申请 Handler（application/json）
 * @date: 2026-02-07
 * @version: 2.0
 */

package verify

import (
	"net/http"

	"activity-platform/app/user/api/internal/logic/verify"
	"activity-platform/app/user/api/internal/svc"
	"activity-platform/app/user/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// ApplyVerifyHandler 提交学生认证申请
// 请求格式: application/json
//   - 字段: real_name, school_name, student_id, department, admission_year, front_image_url, back_image_url
func ApplyVerifyHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := logx.WithContext(r.Context())

		// 解析参数（JSON）
		var req types.ApplyVerifyReq
		if err := httpx.Parse(r, &req); err != nil {
			logger.Errorf("[ApplyVerifyHandler] 参数解析失败: %v", err)
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		logger.Infof("[ApplyVerifyHandler] 收到请求: realName=%s, schoolName=%s, studentId=%s",
			req.RealName, req.SchoolName, req.StudentId)

		// 调用 Logic 层
		l := verify.NewApplyVerifyLogic(r.Context(), svcCtx)
		resp, err := l.ApplyVerify(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
