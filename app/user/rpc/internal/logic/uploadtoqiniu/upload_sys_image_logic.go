package uploadtoqiniulogic

import (
	"context"

	"activity-platform/app/user/model"
	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type UploadSysImageLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUploadSysImageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UploadSysImageLogic {
	return &UploadSysImageLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UploadSysImageLogic) UploadSysImage(in *pb.UploadSysImageReq) (*pb.UploadSysImageResp, error) {
	if in.UserId == 0 || len(in.ImageData) == 0 {
		return nil, errorx.New(errorx.CodeInvalidParams)
	}

	url, err := uploadToQiniu(l.ctx, l.svcCtx, in.ImageData, in.OriginName)
	if err != nil {
		return nil, err
	}

	img := &model.SysImage{
		URL:        url,
		OriginName: in.OriginName,
		BizType:    in.BizType,
		FileSize:   in.FileSize,
		MimeType:   in.MimeType,
		Extension:  in.Extension,
		RefCount:   0,
		UploaderID: in.UserId,
		Status:     1,
	}

	if err := l.svcCtx.SysImageModel.Create(l.ctx, img); err != nil {
		return nil, errorx.New(errorx.CodeDBError)
	}

	return &pb.UploadSysImageResp{
		Id: img.ID,
	}, nil
}
