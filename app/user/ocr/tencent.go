/**
 * @projectName: CampusHub
 * @package: ocr
 * @className: tencent
 * @author: lijunqi
 * @description: 腾讯云OCR提供商实现（使用文档抽取基础版API - ExtractDocBasic）
 * @date: 2026-02-02
 * @version: 2.0
 *
 * API说明：
 * 使用腾讯云"文档抽取基础版"API（Action: ExtractDocBasic，别名: SmartStructuralOCRV2）
 * 接口请求域名：ocr.tencentcloudapi.com
 *
 * V2特性：
 * 1. 支持PDF识别（IsPdf、PdfPageNumber参数）
 * 2. 支持自定义字段提取（ItemNames参数）
 * 3. 支持印章识别（EnableSealRecognize参数）
 * 4. 支持全文字段识别（ReturnFullText参数）
 * 5. 支持多种配置模板（ConfigId参数）
 *
 * 响应结构（SDK类型链）：
 * ExtractDocBasicResponse.Response.StructuralList
 *   └── []*GroupInfo
 *       └── Groups ([]*LineInfo)
 *           └── Lines ([]*ItemInfo)
 *               ├── Key (*Key) -> AutoName/ConfigName
 *               └── Value (*Value) -> AutoContent
 *
 * 注意：V2 API 不返回字段置信度（Confidence），与V1 API不同
 */

package ocr

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"activity-platform/common/errorx"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	ocrSdk "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/ocr/v20181119"
	"github.com/zeromicro/go-zero/core/logx"
)

// ============================================================================
// 腾讯云OCR配置
// ============================================================================

// TencentConfig 腾讯云OCR配置
type TencentConfig struct {
	// SecretId 密钥ID
	SecretId string
	// SecretKey 密钥Key
	SecretKey string
	// Region 地域（如 ap-guangzhou）
	Region string
	// Endpoint 服务端点（如 ocr.tencentcloudapi.com）
	Endpoint string
	// Timeout 超时时间（秒）
	Timeout int
	// Enabled 是否启用
	Enabled bool
}

// ============================================================================
// 腾讯云OCR提供商
// ============================================================================

// TencentProvider 腾讯云OCR提供商
type TencentProvider struct {
	config TencentConfig
	client *ocrSdk.Client
}

// 确保实现 Provider 接口
var _ Provider = (*TencentProvider)(nil)

// NewTencentProvider 创建腾讯云OCR提供商
func NewTencentProvider(config TencentConfig) (*TencentProvider, error) {
	if !config.Enabled {
		return &TencentProvider{config: config}, nil
	}

	// 创建认证对象
	credential := common.NewCredential(config.SecretId, config.SecretKey)

	// 创建客户端配置
	cpf := profile.NewClientProfile()
	// 使用配置中的端点，如果未配置则使用默认值
	endpoint := config.Endpoint
	if endpoint == "" {
		endpoint = "ocr.tencentcloudapi.com"
	}
	cpf.HttpProfile.Endpoint = endpoint
	if config.Timeout > 0 {
		cpf.HttpProfile.ReqTimeout = config.Timeout
	} else {
		cpf.HttpProfile.ReqTimeout = 30
	}

	// 创建OCR客户端
	client, err := ocrSdk.NewClient(credential, config.Region, cpf)
	if err != nil {
		return nil, errorx.ErrOcrConfigInvalid()
	}

	return &TencentProvider{
		config: config,
		client: client,
	}, nil
}

// ============================================================================
// Provider 接口实现
// ============================================================================

// Name 返回提供商名称
func (p *TencentProvider) Name() string {
	return ProviderNameTencent
}

// IsAvailable 检查提供商是否可用
func (p *TencentProvider) IsAvailable(ctx context.Context) bool {
	return p.config.Enabled && p.client != nil
}

// Recognize 执行OCR识别
// 使用腾讯云文档抽取基础版API（ExtractDocBasic / SmartStructuralOCRV2）识别学生证
// 识别策略：
//   - 正面图片：识别 机构(学校)、姓名、学号、学院
//   - 背面图片：仅当正面未识别出学校时，补充识别学校名称
func (p *TencentProvider) Recognize(
	ctx context.Context,
	frontImageURL, backImageURL string,
) (*OcrResult, error) {
	if !p.IsAvailable(ctx) {
		return nil, errorx.ErrOcrServiceUnavailable()
	}

	startTime := time.Now()
	logx.WithContext(ctx).Infof("腾讯云OCR开始识别: front=%s", frontImageURL)

	// 1. 识别正面图片（包含姓名、学校、学号、学院等主要信息）
	frontResult, frontRaw, err := p.recognizeImage(ctx, frontImageURL)
	if err != nil {
		return nil, err
	}

	// 2. 如果正面未识别出学校名称（机构），尝试识别背面补充
	if frontResult.SchoolName == "" && backImageURL != "" {
		logx.WithContext(ctx).Infof("腾讯云OCR正面未识别出机构，尝试识别背面")
		backResult, err := p.recognizeBackImage(ctx, backImageURL)
		if err == nil && backResult.SchoolName != "" {
			frontResult.SchoolName = backResult.SchoolName
			logx.WithContext(ctx).Infof("腾讯云OCR从背面补充机构: %s", backResult.SchoolName)
		}
	}

	// 3. 设置元数据
	frontResult.Platform = p.Name()
	frontResult.RawResponse = frontRaw

	elapsed := time.Since(startTime)
	logx.WithContext(ctx).Infof("腾讯云OCR识别完成: elapsed=%v, school=%s, name=%s, studentId=%s, dept=%s",
		elapsed, frontResult.SchoolName, frontResult.RealName, frontResult.StudentID, frontResult.Department)

	return frontResult, nil
}

// ============================================================================
// 内部方法
// ============================================================================

// recognizeImage 识别单张图片（正面）
// 使用文档抽取基础版API（ExtractDocBasic / SmartStructuralOCRV2）
// 提取字段：机构、姓名、学号、学院
func (p *TencentProvider) recognizeImage(
	ctx context.Context,
	imageURL string,
) (*OcrResult, string, error) {
	// 创建请求 - 使用文档抽取基础版API
	request := ocrSdk.NewExtractDocBasicRequest()
	request.ImageUrl = common.StringPtr(imageURL)

	// 指定需要提取的4个字段：机构、姓名、学号、学院
	request.ItemNames = common.StringPtrs([]string{
		"机构", "姓名", "学号", "学院",
	})

	// 关闭全文识别，仅返回结构化字段
	request.ReturnFullText = common.BoolPtr(false)

	// 调用API（Action: ExtractDocBasic）
	response, err := p.client.ExtractDocBasic(request)
	if err != nil {
		return nil, "", p.handleError(err)
	}

	// 解析响应
	result, err := p.parseExtractDocBasicResponse(response)
	if err != nil {
		return nil, "", err
	}

	// 获取原始响应JSON
	rawJSON, _ := json.Marshal(response.Response)

	return result, string(rawJSON), nil
}

// recognizeBackImage 识别背面图片
// 仅提取机构（学校名称）字段
func (p *TencentProvider) recognizeBackImage(ctx context.Context, imageURL string) (*OcrResult, error) {
	// 创建请求
	request := ocrSdk.NewExtractDocBasicRequest()
	request.ImageUrl = common.StringPtr(imageURL)

	// 背面只提取机构字段
	request.ItemNames = common.StringPtrs([]string{"机构"})
	request.ReturnFullText = common.BoolPtr(false)

	// 调用API
	response, err := p.client.ExtractDocBasic(request)
	if err != nil {
		return nil, p.handleError(err)
	}

	// 解析响应
	return p.parseExtractDocBasicResponse(response)
}

// parseExtractDocBasicResponse 解析文档抽取基础版响应
// 字段映射（根据学生证实际返回）：
//   - 机构 → SchoolName（学校名称）
//   - 姓名 → RealName（真实姓名）
//   - 学号 → StudentID（学号）
//   - 学院 → Department（院系）
//
// 响应结构遍历路径（SDK类型链）：
// Response.StructuralList -> []*GroupInfo
//
// Response
// └── StructuralList (数组)
//
//	└── GroupInfo
//	    └── Groups (数组)
//	        └── LineInfo
//	            └── Lines (数组)
//	                └── ItemInfo
//	                    ├── Key (字段名)
//	                    │   ├── ConfigName: "机构"
//	                    │   └── AutoName: "机构"
//	                    └── Value (字段值)
//	                        └── AutoContent: "北京大学"
//
// 注意：V2 API不返回字段置信度（Confidence）
func (p *TencentProvider) parseExtractDocBasicResponse(response *ocrSdk.ExtractDocBasicResponse) (*OcrResult, error) {
	if response == nil || response.Response == nil {
		return nil, errorx.ErrOcrEmptyResult()
	}

	result := &OcrResult{}

	// 遍历结构化识别结果
	// StructuralList -> []*GroupInfo
	if response.Response.StructuralList == nil {
		return nil, errorx.ErrOcrEmptyResult()
	}

	for _, groupInfo := range response.Response.StructuralList {
		if groupInfo == nil || groupInfo.Groups == nil {
			continue
		}

		// GroupInfo.Groups -> []*LineInfo
		for _, lineInfo := range groupInfo.Groups {
			if lineInfo == nil || lineInfo.Lines == nil {
				continue
			}

			// LineInfo.Lines -> []*ItemInfo
			for _, itemInfo := range lineInfo.Lines {
				if itemInfo == nil || itemInfo.Key == nil || itemInfo.Value == nil {
					continue
				}

				// 获取字段名称：优先使用ConfigName（用户指定），其次使用AutoName（自动识别）
				key := ""
				if itemInfo.Key.ConfigName != nil && *itemInfo.Key.ConfigName != "" {
					key = strings.TrimSpace(*itemInfo.Key.ConfigName)
				} else if itemInfo.Key.AutoName != nil {
					key = strings.TrimSpace(*itemInfo.Key.AutoName)
				}

				// 获取字段值
				value := ""
				if itemInfo.Value.AutoContent != nil {
					value = strings.TrimSpace(*itemInfo.Value.AutoContent)
				}

				if key == "" || value == "" {
					continue
				}

				// 根据键名映射到结果字段
				p.mapFieldByKey(result, key, value)
			}
		}
	}

	// V2 API不返回置信度，默认设置为0
	// 如需置信度功能，请使用V1 API（SmartStructuralOCR）
	result.Confidence = 0

	return result, nil
}

// mapFieldByKey 根据键名映射到结果字段
// 字段映射规则（仅4个字段）：
//   - "机构" → SchoolName（学校名称）
//   - "姓名" → RealName（真实姓名）
//   - "学号" → StudentID（学号）
//   - "学院" → Department（院系）
func (p *TencentProvider) mapFieldByKey(result *OcrResult, key, value string) {
	// 移除可能的冒号和空格
	key = strings.TrimSpace(key)

	switch key {
	// 机构 → SchoolName
	case "机构":
		if result.SchoolName == "" {
			result.SchoolName = value
		}

	// 姓名 → RealName
	case "姓名":
		if result.RealName == "" {
			result.RealName = value
		}

	// 学号 → StudentID
	case "学号":
		if result.StudentID == "" {
			result.StudentID = value
		}

	// 学院 → Department
	case "学院":
		if result.Department == "" {
			result.Department = value
		}
	}
}

// handleError 处理腾讯云API错误，直接返回 BizError
func (p *TencentProvider) handleError(err error) *errorx.BizError {
	// 记录原始错误日志
	logx.Errorf("[%s] OCR API错误: %v", p.Name(), err)

	if sdkErr, ok := err.(*errors.TencentCloudSDKError); ok {
		switch sdkErr.Code {
		case "FailedOperation.ImageDecodeFailed",
			"FailedOperation.ImageNoText",
			"InvalidParameter.ImageStringError":
			return errorx.ErrOcrImageInvalid()

		case "ResourceUnavailable.InArrears":
			return errorx.ErrOcrInsufficientBalance()

		case "RequestLimitExceeded":
			return errorx.ErrOcrRecognizeFailed()

		default:
			return errorx.ErrOcrRecognizeFailed()
		}
	}

	// 网络超时
	if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "Timeout") {
		return errorx.ErrOcrNetworkTimeout()
	}

	return errorx.ErrOcrRecognizeFailed()
}
