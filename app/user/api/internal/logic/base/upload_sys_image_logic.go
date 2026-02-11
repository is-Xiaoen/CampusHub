// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package base

import (
	"context"

	"activity-platform/app/user/api/internal/svc"
	"activity-platform/app/user/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UploadSysImageLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 上传系统图片
func NewUploadSysImageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UploadSysImageLogic {
	return &UploadSysImageLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UploadSysImageLogic) UploadSysImage(req *types.UploadSysImageReq) (resp *types.UploadSysImageResp, err error) {
	// todo: add your logic here and delete this line

	return
}
