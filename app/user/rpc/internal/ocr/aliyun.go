/**
 * @projectName: CampusHub
 * @package: ocr
 * @className: aliyun
 * @author: lijunqi
 * @description: 阿里云OCR提供商实现（使用通用票证抽取API - RecognizeGeneralStructure）
 * @date: 2026-02-02
 * @version: 2.0
 *
 * API说明：
 * 使用阿里云"通用票证抽取"API（RecognizeGeneralStructure）
 * 该API结合读光OCR和通义千问大模型的能力，提供关键KV信息抽取
 *
 * 特性：
 * 1. 支持指定需要抽取的Key字段（最多30个）
 * 2. 返回结构化的KV信息，无需手动解析文本
 * 3. 支持多种图片格式：PNG、JPG、JPEG、PDF、BMP、GIF、TIFF、WebP
 *
 * 响应结构：
 * Response.Data.SubImages[].KvInfo.Data -> map[string]string
 *
 *	例如: {"姓名": "张三", "学号": "2024001234", "学校": "北京大学"}
 */
package ocr

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"activity-platform/common/errorx"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	ocrapi "github.com/alibabacloud-go/ocr-api-20210707/v3/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/zeromicro/go-zero/core/logx"
)

// ============================================================================
// 阿里云OCR配置
// ============================================================================

// AliyunConfig 阿里云OCR配置
type AliyunConfig struct {
	// AccessKeyId 访问密钥ID
	AccessKeyId string
	// AccessKeySecret 访问密钥Secret
	AccessKeySecret string
	// Endpoint 服务端点（如 ocr-api.cn-hangzhou.aliyuncs.com）
	Endpoint string
	// Timeout 超时时间（秒）
	Timeout int
	// Enabled 是否启用
	Enabled bool
}

// ============================================================================
// 阿里云OCR提供商
// ============================================================================

// AliyunProvider 阿里云OCR提供商
type AliyunProvider struct {
	config AliyunConfig
	client *ocrapi.Client
}

// 确保实现 Provider 接口
var _ Provider = (*AliyunProvider)(nil)

// NewAliyunProvider 创建阿里云OCR提供商
func NewAliyunProvider(config AliyunConfig) (*AliyunProvider, error) {
	if !config.Enabled {
		return &AliyunProvider{config: config}, nil
	}

	// 设置默认端点
	endpoint := config.Endpoint
	if endpoint == "" {
		endpoint = "ocr-api.cn-hangzhou.aliyuncs.com"
	}

	// 创建配置
	openApiConfig := &openapi.Config{
		AccessKeyId:     tea.String(config.AccessKeyId),
		AccessKeySecret: tea.String(config.AccessKeySecret),
		Endpoint:        tea.String(endpoint),
	}

	// 创建OCR客户端
	client, err := ocrapi.NewClient(openApiConfig)
	if err != nil {
		return nil, errorx.ErrOcrConfigInvalid()
	}

	return &AliyunProvider{
		config: config,
		client: client,
	}, nil
}

// ============================================================================
// Provider 接口实现
// ============================================================================

// Name 返回提供商名称
func (p *AliyunProvider) Name() string {
	return ProviderNameAliyun
}

// IsAvailable 检查提供商是否可用
func (p *AliyunProvider) IsAvailable(ctx context.Context) bool {
	return p.config.Enabled && p.client != nil
}

// Recognize 执行OCR识别
// 使用阿里云通用票证抽取API（RecognizeGeneralStructure）识别学生证
// 识别策略：
//   - 正面图片：识别 姓名、学号、学校、学院
//   - 背面图片：仅当正面未识别出学校时，补充识别学校名称
func (p *AliyunProvider) Recognize(
	ctx context.Context,
	frontImageURL, backImageURL string,
) (*OcrResult, error) {
	if !p.IsAvailable(ctx) {
		return nil, errorx.ErrOcrServiceUnavailable()
	}

	startTime := time.Now()
	logx.WithContext(ctx).Infof("阿里云OCR开始识别: front=%s", frontImageURL)

	// 1. 识别正面图片（包含姓名、学校、学号、学院等主要信息）
	frontResult, frontRaw, err := p.recognizeImage(ctx, frontImageURL)
	if err != nil {
		return nil, err
	}

	// 2. 如果正面未识别出学校名称，尝试识别背面补充
	if frontResult.SchoolName == "" && backImageURL != "" {
		logx.WithContext(ctx).Infof("阿里云OCR正面未识别出学校，尝试识别背面")
		backResult, _, err := p.recognizeImage(ctx, backImageURL)
		if err == nil && backResult.SchoolName != "" {
			frontResult.SchoolName = backResult.SchoolName
			logx.WithContext(ctx).Infof("阿里云OCR从背面补充学校: %s", backResult.SchoolName)
		}
	}

	// 3. 设置元数据
	frontResult.Platform = p.Name()
	frontResult.RawResponse = frontRaw

	elapsed := time.Since(startTime)
	logx.WithContext(ctx).Infof("阿里云OCR识别完成: elapsed=%v, school=%s, name=%s, studentId=%s, dept=%s",
		elapsed, frontResult.SchoolName, frontResult.RealName, frontResult.StudentID, frontResult.Department)

	return frontResult, nil
}

// ============================================================================
// 内部方法
// ============================================================================

// recognizeImage 识别单张图片
// 使用通用票证抽取API（RecognizeGeneralStructure）
func (p *AliyunProvider) recognizeImage(
	ctx context.Context,
	imageURL string,
) (*OcrResult, string, error) {
	// 创建请求 - 使用通用票证抽取API
	request := &ocrapi.RecognizeGeneralStructureRequest{
		Url: tea.String(imageURL),
		// 指定需要提取的4个字段（根据实际返回的KEY）
		Keys: tea.StringSlice([]string{
			"姓名",   // → RealName
			"学校名称", // → SchoolName
			"学院",   // → Department
			"学号",   // → StudentID
		}),
	}

	// 调用API（注意：此接口30秒超时）
	response, err := p.client.RecognizeGeneralStructure(request)
	if err != nil {
		return nil, "", p.handleError(err)
	}

	// 解析响应
	result, err := p.parseResponse(response)
	if err != nil {
		return nil, "", err
	}

	// 获取原始响应JSON
	rawJSON, _ := json.Marshal(response.Body)

	return result, string(rawJSON), nil
}

// ============================================================================
// 响应解析
// ============================================================================

// AliyunStructureResponse 阿里云通用票证抽取响应结构
type AliyunStructureResponse struct {
	// Data 识别结果
	Data *AliyunStructureData `json:"Data"`
}

// AliyunStructureData 识别数据
type AliyunStructureData struct {
	// Height 原图高度
	Height int `json:"Height"`
	// Width 原图宽度
	Width int `json:"Width"`
	// SubImageCount 子图数量
	SubImageCount int `json:"SubImageCount"`
	// SubImages 子图信息列表
	SubImages []AliyunSubImage `json:"SubImages"`
}

// AliyunSubImage 子图信息
type AliyunSubImage struct {
	// SubImageId 子图ID
	SubImageId int `json:"SubImageId"`
	// Angle 旋转角度
	Angle int `json:"Angle"`
	// KvInfo 结构化信息
	KvInfo *AliyunKvInfo `json:"KvInfo"`
}

// AliyunKvInfo 结构化KV信息
type AliyunKvInfo struct {
	// KvCount 键值对数量
	KvCount int `json:"KvCount"`
	// Data 键值对数据（如 {"姓名": "张三", "学号": "2024001234"}）
	Data map[string]string `json:"Data"`
}

// parseResponse 解析阿里云OCR响应
func (p *AliyunProvider) parseResponse(response *ocrapi.RecognizeGeneralStructureResponse) (*OcrResult, error) {
	if response == nil || response.Body == nil || response.Body.Data == nil {
		return nil, errorx.ErrOcrEmptyResult()
	}

	// 解析Data字段（JSON对象）
	var structData AliyunStructureData
	dataBytes, err := json.Marshal(response.Body.Data)
	if err != nil {
		return nil, errorx.ErrOcrRecognizeFailed()
	}
	if err := json.Unmarshal(dataBytes, &structData); err != nil {
		return nil, errorx.ErrOcrRecognizeFailed()
	}

	result := &OcrResult{}

	// 遍历子图，提取KV信息
	for _, subImage := range structData.SubImages {
		if subImage.KvInfo == nil || subImage.KvInfo.Data == nil {
			continue
		}

		// 从KV数据中提取字段
		for key, value := range subImage.KvInfo.Data {
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			p.mapFieldByKey(result, key, value)
		}
	}

	// 如果没有提取到任何有效信息
	if result.RealName == "" && result.SchoolName == "" && result.StudentID == "" {
		return nil, errorx.ErrOcrEmptyResult()
	}

	return result, nil
}

// mapFieldByKey 根据键名映射到结果字段
// 字段映射规则（仅4个字段）：
//
//	阿里云返回KEY     →    OcrResult字段
//	─────────────────────────────────────
//	"姓名"           →    RealName
//	"学校名称"       →    SchoolName
//	"学院"           →    Department
//	"学号"           →    StudentID
func (p *AliyunProvider) mapFieldByKey(result *OcrResult, key, value string) {
	// 移除可能的冒号和空格
	key = strings.TrimSpace(key)

	switch key {
	// 姓名 → RealName
	case "姓名":
		if result.RealName == "" {
			result.RealName = value
		}

	// 学校名称 → SchoolName
	case "学校名称":
		if result.SchoolName == "" {
			result.SchoolName = value
		}

	// 学院 → Department
	case "学院":
		if result.Department == "" {
			result.Department = value
		}

	// 学号 → StudentID
	case "学号":
		if result.StudentID == "" {
			result.StudentID = value
		}
	}
}

// ============================================================================
// 错误处理
// ============================================================================

// handleError 处理阿里云API错误，直接返回 BizError
func (p *AliyunProvider) handleError(err error) *errorx.BizError {
	// 记录原始错误日志
	logx.Errorf("[%s] OCR API错误: %v", p.Name(), err)

	errMsg := err.Error()

	// 检查常见错误
	if strings.Contains(errMsg, "InvalidImage") || strings.Contains(errMsg, "ImageFormatError") {
		return errorx.ErrOcrImageInvalid()
	}

	if strings.Contains(errMsg, "NoBalance") || strings.Contains(errMsg, "Insufficient") {
		return errorx.ErrOcrInsufficientBalance()
	}

	if strings.Contains(errMsg, "Timeout") || strings.Contains(errMsg, "timeout") ||
		strings.Contains(errMsg, "LLMTimeout") {
		return errorx.ErrOcrNetworkTimeout()
	}

	if strings.Contains(errMsg, "Throttling") || strings.Contains(errMsg, "RateLimit") {
		return errorx.ErrOcrRecognizeFailed()
	}

	if strings.Contains(errMsg, "ExceededKeyNumber") {
		return errorx.ErrOcrRecognizeFailed()
	}

	return errorx.ErrOcrRecognizeFailed()
}
