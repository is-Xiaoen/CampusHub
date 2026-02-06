/**
 * @projectName: CampusHub
 * @package: ocr
 * @className: interface
 * @author: lijunqi
 * @description: OCR模块核心接口定义，提供统一的OCR识别能力抽象
 * @date: 2026-01-31
 * @version: 1.0
 */

package ocr

import "context"

// ============================================================================
// OCR识别结果
// ============================================================================

// OcrResult OCR识别结果（统一结构体）
// 屏蔽不同OCR厂商的字段差异（腾讯叫Name，阿里叫RealName等）
// 所有敏感字段在这里是明文，业务层负责在入库前加密
type OcrResult struct {
	// ==================== 识别出的学生信息 ====================

	// 真实姓名（明文，入库前需加密）
	RealName string `json:"real_name"`
	// 学校名称
	SchoolName string `json:"school_name"`
	// 学号（明文，入库前需加密）
	StudentID string `json:"student_id"`
	// 院系
	Department string `json:"department"`
	// 入学年份
	AdmissionYear string `json:"admission_year"`

	// ==================== OCR元数据 ====================

	// OCR平台：tencent / aliyun
	Platform string `json:"platform"`
	// 识别置信度（0-100）
	Confidence float64 `json:"confidence"`
	// 原始响应JSON（用于审计追溯）
	RawResponse string `json:"raw_response"`
}

// ============================================================================
// OCR提供商接口
// ============================================================================

// Provider OCR提供商接口
// 任何接入的OCR厂商都必须实现这个接口
type Provider interface {
	// Recognize 执行OCR识别
	// 参数:
	//   - ctx: 上下文（用于超时控制、链路追踪）
	//   - frontImageURL: 学生证正面图片URL
	//   - backImageURL: 学生证详情面图片URL
	// 返回:
	//   - *OcrResult: 统一格式的识别结果
	//   - error: 错误信息（网络错误、识别失败、余额不足等）
	Recognize(ctx context.Context, frontImageURL, backImageURL string) (*OcrResult, error)

	// Name 返回提供商名称（用于日志记录和熔断器Key）
	// 返回值示例: "tencent", "aliyun"
	Name() string

	// IsAvailable 检查提供商是否可用
	// 用于判断：余额是否充足、是否被熔断、服务是否正常
	IsAvailable(ctx context.Context) bool
}

// ============================================================================
// 提供商名称常量
// ============================================================================

const (
	// ProviderNameTencent 腾讯云
	ProviderNameTencent = "tencent"
	// ProviderNameAliyun 阿里云
	ProviderNameAliyun = "aliyun"
)
