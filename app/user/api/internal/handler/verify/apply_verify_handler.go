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
	"io"
	"net/http"
	"strings"

	"activity-platform/app/user/api/internal/logic/verify"
	"activity-platform/app/user/api/internal/svc"
	"activity-platform/app/user/api/internal/types"
	"activity-platform/common/errorx"
	"activity-platform/common/response"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

const (
	maxUploadSize = 10 << 20 // 整个表单限制 10MB
	maxFileSize   = 5 << 20  // 单个文件限制 5MB
)

// allowedImageTypes 允许的图片 MIME 类型
var allowedImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/jpg":  true,
	"image/png":  true,
}

// ApplyVerifyHandler 提交学生认证申请
// 请求格式: multipart/form-data
//   - 文本字段: real_name, school_name, student_id, department, admission_year
//   - 文件字段: front_image（正面照片）, back_image（详情面照片）
func ApplyVerifyHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := logx.WithContext(r.Context())

		// 1. 解析 multipart 表单（限制总大小 10MB）
		if err := r.ParseMultipartForm(maxUploadSize); err != nil {
			logger.Errorf("[ApplyVerifyHandler] ParseMultipartForm 失败: %v", err)
			response.Fail(w, errorx.ErrInvalidParams("请求格式错误，请使用 multipart/form-data"))
			return
		}

		// 2. 解析文本字段到 req 结构体（go-zero 支持 form tag）
		var req types.ApplyVerifyReq
		if err := httpx.Parse(r, &req); err != nil {
			logger.Errorf("[ApplyVerifyHandler] 文本字段解析失败: %v", err)
			response.Fail(w, errorx.ErrInvalidParams("参数解析失败"))
			return
		}

		logger.Infof("[ApplyVerifyHandler] 收到请求: realName=%s, schoolName=%s, studentId=%s",
			req.RealName, req.SchoolName, req.StudentId)

		// 3. 读取正面照片文件
		frontData, frontName, err := readAndValidateFile(r, "front_image")
		if err != nil {
			logger.Errorf("[ApplyVerifyHandler] 正面照片校验失败: %v", err)
			response.Fail(w, err)
			return
		}

		// 4. 读取详情面照片文件
		backData, backName, err := readAndValidateFile(r, "back_image")
		if err != nil {
			logger.Errorf("[ApplyVerifyHandler] 详情面照片校验失败: %v", err)
			response.Fail(w, err)
			return
		}

		// 5. 调用 Logic 层（文本字段 + 文件数据）
		l := verify.NewApplyVerifyLogic(r.Context(), svcCtx)
		resp, err := l.ApplyVerify(&req, frontData, frontName, backData, backName)
		if err != nil {
			response.Fail(w, err)
		} else {
			response.Success(w, resp)
		}
	}
}

// readAndValidateFile 从 multipart 表单中读取并校验文件
// 返回: 文件数据([]byte), 文件名(string), 错误(error)
func readAndValidateFile(r *http.Request, fieldName string) ([]byte, string, error) {
	file, header, err := r.FormFile(fieldName)
	if err != nil {
		if fieldName == "front_image" {
			return nil, "", errorx.New(errorx.CodeVerifyFrontImageMissing)
		}
		return nil, "", errorx.New(errorx.CodeVerifyBackImageMissing)
	}
	defer file.Close()

	// 校验文件大小
	if header.Size > maxFileSize {
		return nil, "", errorx.New(errorx.CodeVerifyImageTooLarge)
	}

	// 校验文件类型
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		// 如果 Header 没有 Content-Type，通过扩展名判断
		ext := strings.ToLower(header.Filename[strings.LastIndex(header.Filename, ".")+1:])
		switch ext {
		case "jpg", "jpeg":
			contentType = "image/jpeg"
		case "png":
			contentType = "image/png"
		}
	}
	if !allowedImageTypes[contentType] {
		return nil, "", errorx.New(errorx.CodeVerifyImageFormatError)
	}

	// 读取文件内容
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, "", errorx.NewSystemError("读取文件失败")
	}

	return data, header.Filename, nil
}
