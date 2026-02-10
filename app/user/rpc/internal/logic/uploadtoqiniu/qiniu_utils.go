package uploadtoqiniulogic

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/common/errorx"

	"github.com/google/uuid"
	"github.com/qiniu/go-sdk/v7/auth/qbox"
	"github.com/qiniu/go-sdk/v7/storage"
	"github.com/zeromicro/go-zero/core/logx"
)

// uploadToQiniu 上传文件到七牛云
func uploadToQiniu(ctx context.Context, svcCtx *svc.ServiceContext, data []byte, fileName string) (string, error) {
	logger := logx.WithContext(ctx)

	// 1. 大小限制 (5MB)
	const MaxFileSize = 5 * 1024 * 1024
	if len(data) > MaxFileSize {
		return "", errorx.New(errorx.CodeFileTooLarge)
	}

	// 2. 格式检查 (仅允许图片)
	contentType := http.DetectContentType(data)
	if !strings.HasPrefix(contentType, "image/") {
		return "", errorx.New(errorx.CodeFileTypeInvalid)
	}

	// 3. 准备七牛云配置
	qConfig := svcCtx.Config.Qiniu
	if qConfig.AccessKey == "" || qConfig.SecretKey == "" {
		return "", errorx.New(errorx.CodeFileConfigError)
	}

	// 4. 生成上传凭证
	mac := qbox.NewMac(qConfig.AccessKey, qConfig.SecretKey)
	putPolicy := storage.PutPolicy{
		Scope: qConfig.Bucket,
	}
	upToken := putPolicy.UploadToken(mac)

	// 5. 配置上传管理器
	cfg := storage.Config{}
	// 根据 Zone 配置空间对应的机房
	switch qConfig.Zone {
	case "Huadong", "z0":
		cfg.Zone = &storage.ZoneHuadong
	case "Huabei", "z1":
		cfg.Zone = &storage.ZoneHuabei
	case "Huanan", "z2":
		cfg.Zone = &storage.ZoneHuanan
	case "Beimei", "na0":
		cfg.Zone = &storage.ZoneBeimei
	case "Xinjiapo", "as0":
		cfg.Zone = &storage.ZoneXinjiapo
	default:
		// 默认使用华南（根据用户配置）或让 SDK 自动探测
		cfg.Zone = &storage.ZoneHuanan
	}
	cfg.UseHTTPS = false
	cfg.UseCdnDomains = false

	formUploader := storage.NewFormUploader(&cfg)
	ret := storage.PutRet{}

	// 6. 生成唯一文件名
	ext := filepath.Ext(fileName)
	if ext == "" {
		// 尝试根据 contentType 补充后缀
		switch contentType {
		case "image/jpeg":
			ext = ".jpg"
		case "image/png":
			ext = ".png"
		case "image/gif":
			ext = ".gif"
		case "image/webp":
			ext = ".webp"
		default:
			ext = ".jpg"
		}
	}
	key := fmt.Sprintf("%s%s", uuid.NewString(), ext)

	// 7. 执行上传
	dataLen := int64(len(data))
	dataReader := bytes.NewReader(data)
	err := formUploader.Put(ctx, &ret, upToken, key, dataReader, dataLen, nil)
	if err != nil {
		logger.Errorf("Failed to upload to qiniu: %v", err)
		return "", errorx.New(errorx.CodeFileUploadFailed)
	}

	// 8. 拼接返回 URL
	domain := qConfig.Domain
	if !strings.HasPrefix(domain, "http") {
		domain = "http://" + domain
	}
	fileUrl := fmt.Sprintf("%s/%s", strings.TrimRight(domain, "/"), ret.Key)

	return fileUrl, nil
}

// deleteFromQiniu 从七牛云删除文件
func deleteFromQiniu(ctx context.Context, svcCtx *svc.ServiceContext, fileUrl string) error {
	logger := logx.WithContext(ctx)

	// 1. 基础校验
	if fileUrl == "" {
		return nil // URL为空视为成功（无需删除）
	}

	// 2. 从 URL 中提取 Key (文件名)
	u, err := url.Parse(fileUrl)
	if err != nil {
		logger.Errorf("Invalid file url format: %s", fileUrl)
		return errorx.ErrInvalidParams("文件URL格式错误")
	}
	// 去掉开头的 "/"
	key := strings.TrimPrefix(u.Path, "/")
	if key == "" {
		logger.Errorf("Cannot parse key from url: %s", fileUrl)
		return errorx.ErrInvalidParams("无法解析文件Key")
	}

	// 3. 准备七牛云配置
	qConfig := svcCtx.Config.Qiniu
	if qConfig.AccessKey == "" || qConfig.SecretKey == "" {
		return errorx.New(errorx.CodeFileConfigError)
	}

	// 4. 配置 Bucket Manager
	mac := qbox.NewMac(qConfig.AccessKey, qConfig.SecretKey)
	cfg := storage.Config{}
	// 根据 Zone 配置
	switch qConfig.Zone {
	case "Huadong", "z0":
		cfg.Zone = &storage.ZoneHuadong
	case "Huabei", "z1":
		cfg.Zone = &storage.ZoneHuabei
	case "Huanan", "z2":
		cfg.Zone = &storage.ZoneHuanan
	case "Beimei", "na0":
		cfg.Zone = &storage.ZoneBeimei
	case "Xinjiapo", "as0":
		cfg.Zone = &storage.ZoneXinjiapo
	default:
		cfg.Zone = &storage.ZoneHuanan
	}
	cfg.UseHTTPS = false
	bucketManager := storage.NewBucketManager(mac, &cfg)

	// 5. 执行删除
	err = bucketManager.Delete(qConfig.Bucket, key)
	if err != nil {
		logger.Errorf("Failed to delete file from qiniu: %v", err)
		// 即使删除失败，通常也允许继续上传流程，记录日志即可
		return errorx.New(errorx.CodeFileDeleteFailed)
	}

	return nil
}
