/**
 * @projectName: CampusHub
 * @package: verify
 * @className: ApplyVerifyHandler
 * @description: 提交学生认证申请 Handler（multipart/form-data）
 * @date: 2026-02-07
 * @version: 2.0
 */

package verify

import (
	"net/http"

	"activity-platform/app/user/api/internal/logic/verify"
	"activity-platform/app/user/api/internal/svc"
	"activity-platform/app/user/api/internal/types"
	"activity-platform/common/errorx"
	"activity-platform/common/response"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// ApplyVerifyHandler 提交学生认证申请
// 请求格式: multipart/form-data
//   - 文本字段: real_name, school_name, student_id, department, admission_year
//   - 图片URL: front_image_url, back_image_url
func ApplyVerifyHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := logx.WithContext(r.Context())

		// 解析参数（go-zero httpx.Parse 支持 multipart/form-data 中的文本字段）
		var req types.ApplyVerifyReq
		if err := httpx.Parse(r, &req); err != nil {
			logger.Errorf("[ApplyVerifyHandler] 参数解析失败: %v", err)
			response.Fail(w, errorx.ErrInvalidParams("参数解析失败"))
			return
		}

		logger.Infof("[ApplyVerifyHandler] 收到请求: realName=%s, schoolName=%s, studentId=%s",
			req.RealName, req.SchoolName, req.StudentId)

		// 调用 Logic 层
		l := verify.NewApplyVerifyLogic(r.Context(), svcCtx)
		resp, err := l.ApplyVerify(&req)
		if err != nil {
			response.Fail(w, err)
		} else {
			response.Success(w, resp)
		}
	}
}
