package userbasicservicelogic

import (
	"context"

	"activity-platform/app/user/model"
	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type GetSysImageLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetSysImageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSysImageLogic {
	return &GetSysImageLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取系统图片
func (l *GetSysImageLogic) GetSysImage(in *pb.GetSysImageReq) (*pb.GetSysImageResp, error) {
	// 1. 查询图片记录
	image, err := l.svcCtx.SysImageModel.FindByID(l.ctx, in.ImageId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errorx.NewWithMessage(errorx.CodeNotFound, "图片不存在")
		}
		l.Errorf("GetSysImage 查询图片失败: id=%d, err=%v", in.ImageId, err)
		return nil, errorx.FromError(err)
	}

	// 2. 校验状态
	if image.Status != model.SysImageStatusNormal {
		return nil, errorx.NewWithMessage(errorx.CodeForbidden, "图片状态异常")
	}

	// 3. 校验业务类型
	// 如果是身份认证图片，必须是本人才能查看
	if image.BizType == model.SysImageBizTypeIdentityAuth {
		if image.UploaderID != in.UserId {
			return nil, errorx.NewWithMessage(errorx.CodeForbidden, "无权访问该图片")
		}
	}

	// 4. 返回URL
	return &pb.GetSysImageResp{
		Url: image.URL,
	}, nil
}
