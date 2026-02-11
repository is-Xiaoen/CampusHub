package userbasicservicelogic

import (
	"context"

	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type UpdateSysImageRefCountLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateSysImageRefCountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateSysImageRefCountLogic {
	return &UpdateSysImageRefCountLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 更新图片引用计数
func (l *UpdateSysImageRefCountLogic) UpdateSysImageRefCount(in *pb.UpdateSysImageRefCountReq) (*pb.UpdateSysImageRefCountResp, error) {
	// 参数校验：delta 仅允许 -1, 0, 1
	if in.Delta != -1 && in.Delta != 0 && in.Delta != 1 {
		return nil, errorx.NewWithMessage(errorx.CodeInvalidParams, "delta 必须为 -1/0/1")
	}

	// 查询图片记录
	image, err := l.svcCtx.SysImageModel.FindByID(l.ctx, in.ImageId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errorx.NewWithMessage(errorx.CodeNotFound, "图片不存在")
		}
		l.Errorf("UpdateSysImageRefCount 查询图片失败: id=%d, err=%v", in.ImageId, err)
		return nil, errorx.FromError(err)
	}

	// delta=0：仅查询当前引用计数
	if in.Delta == 0 {
		return &pb.UpdateSysImageRefCountResp{
			NewRefCount: image.RefCount,
		}, nil
	}

	// 按 delta 更新：1 增加，-1 减少（不低于 0）
	newRef := image.RefCount + int64(in.Delta)
	if newRef < 0 {
		newRef = 0
	}
	image.RefCount = newRef

	// 持久化更新
	if err := l.svcCtx.SysImageModel.Update(l.ctx, image); err != nil {
		l.Errorf("UpdateSysImageRefCount 更新失败: id=%d, err=%v", in.ImageId, err)
		return nil, errorx.NewWithMessage(errorx.CodeDBError, "更新引用计数失败")
	}

	return &pb.UpdateSysImageRefCountResp{
		NewRefCount: image.RefCount,
	}, nil
}
