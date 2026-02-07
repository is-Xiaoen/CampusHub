package uploadtoqiniulogic

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"
	"activity-platform/common/errorx"

	"github.com/google/uuid"
	"github.com/qiniu/go-sdk/v7/auth/qbox"
	"github.com/qiniu/go-sdk/v7/storage"
	"github.com/zeromicro/go-zero/core/logx"
)

type UploadFileLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUploadFileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UploadFileLogic {
	return &UploadFileLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UploadFileLogic) UploadFile(in *pb.UploadFileReq) (*pb.UploadFileResponse, error) {
	// 1. 大小限制 (5MB)
	const MaxFileSize = 5 * 1024 * 1024
	if len(in.FileData) > MaxFileSize {
		return nil, errorx.New(errorx.CodeFileTooLarge)
	}

	// 2. 格式检查 (仅允许图片)
	contentType := http.DetectContentType(in.FileData)
	if !strings.HasPrefix(contentType, "image/") {
		return nil, errorx.New(errorx.CodeFileTypeInvalid)
	}

	// 3. 准备七牛云配置
	qConfig := l.svcCtx.Config.Qiniu
	if qConfig.AccessKey == "" || qConfig.SecretKey == "" {
		return nil, errorx.New(errorx.CodeFileConfigError)
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
	ext := filepath.Ext(in.FileName)
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
	dataLen := int64(len(in.FileData))
	dataReader := bytes.NewReader(in.FileData)
	err := formUploader.Put(l.ctx, &ret, upToken, key, dataReader, dataLen, nil)
	if err != nil {
		l.Logger.Errorf("Failed to upload to qiniu: %v", err)
		return nil, errorx.New(errorx.CodeFileUploadFailed)
	}

	// 8. 拼接返回 URL
	domain := qConfig.Domain
	if !strings.HasPrefix(domain, "http") {
		domain = "http://" + domain
	}
	fileUrl := fmt.Sprintf("%s/%s", strings.TrimRight(domain, "/"), ret.Key)

	return &pb.UploadFileResponse{
		FileUrl: fileUrl,
	}, nil
}
