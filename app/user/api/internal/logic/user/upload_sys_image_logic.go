package user

import (
	"context"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"activity-platform/app/user/api/internal/svc"
	"activity-platform/app/user/api/internal/types"
	"activity-platform/app/user/rpc/client/uploadtoqiniu"
	"activity-platform/common/errorx"
	ctxUtils "activity-platform/common/utils/context"

	"github.com/zeromicro/go-zero/core/logx"
)

type UploadSysImageLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewUploadSysImageLogic(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *UploadSysImageLogic {
	return &UploadSysImageLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *UploadSysImageLogic) UploadSysImage(req *types.UploadSysImageReq) (resp *types.UploadSysImageResp, err error) {
	userId, err := ctxUtils.GetUserIdFromCtx(l.ctx)
	if err != nil {
		return nil, err
	}

	file, header, err := l.r.FormFile("file")
	if err != nil {
		return nil, errorx.New(errorx.CodeInvalidParams)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, errorx.New(errorx.CodeInvalidParams)
	}

	originName := header.Filename
	ext := filepath.Ext(originName)

	bizType := req.BizType
	bizType = strings.TrimSpace(bizType)
	if bizType == "" {
		return nil, errorx.New(errorx.CodeInvalidParams)
	}
	if bizType != "avatar" && bizType != "activity_cover" && bizType != "identity_auth" {
		return nil, errorx.New(errorx.CodeInvalidParams)
	}

	rpcResp, err := l.svcCtx.UploadToQiNiuRpc.UploadSysImage(l.ctx, &uploadtoqiniu.UploadSysImageReq{
		UserId:     userId,
		OriginName: originName,
		BizType:    bizType,
		FileSize:   int64(len(data)),
		MimeType:   http.DetectContentType(data),
		Extension:  ext,
		ImageData:  data,
	})
	if err != nil {
		return nil, err
	}

	return &types.UploadSysImageResp{
		Id: rpcResp.Id,
	}, nil
}
