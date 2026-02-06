package uploadtoqiniulogic

import (
	"context"
	"net/url"
	"strings"

	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"

	"github.com/qiniu/go-sdk/v7/auth/qbox"
	"github.com/qiniu/go-sdk/v7/storage"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type DeleteFileLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteFileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteFileLogic {
	return &DeleteFileLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DeleteFileLogic) DeleteFile(in *pb.DeleteFileReq) (*pb.DeleteFileResponse, error) {
	// 1. 基础校验
	if in.FileUrl == "" {
		return nil, status.Error(codes.InvalidArgument, "file url is empty")
	}

	// 2. 从 URL 中提取 Key (文件名)
	// 假设 URL 格式: http://domain.com/key
	u, err := url.Parse(in.FileUrl)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid file url")
	}
	// 去掉开头的 "/"
	key := strings.TrimPrefix(u.Path, "/")
	if key == "" {
		return nil, status.Error(codes.InvalidArgument, "file key not found in url")
	}

	// 3. 准备七牛云配置
	qConfig := l.svcCtx.Config.Qiniu
	if qConfig.AccessKey == "" || qConfig.SecretKey == "" {
		return nil, status.Error(codes.Internal, "qiniu config is missing")
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
		l.Logger.Errorf("Failed to delete file from qiniu: %v", err)
		// 如果是文件不存在，也可以视为成功，或者返回特定错误
		// 这里简单处理：只要 SDK 报错就返回错误
		return nil, status.Error(codes.Internal, "delete failed")
	}

	return &pb.DeleteFileResponse{
		Success: true,
	}, nil
}
