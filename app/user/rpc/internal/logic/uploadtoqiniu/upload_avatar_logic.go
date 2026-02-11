package uploadtoqiniulogic

import (
	"context"

	"activity-platform/app/user/model"
	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type UploadAvatarLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUploadAvatarLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UploadAvatarLogic {
	return &UploadAvatarLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 直接上传用户头像（删除旧图并更新DB）
func (l *UploadAvatarLogic) UploadAvatar(in *pb.UploadAvatarReq) (*pb.UploadAvatarResp, error) {
	// 查询用户
	user, err := l.svcCtx.UserModel.FindByUserID(l.ctx, in.UserId)
	if err != nil {
		// gorm ErrRecordNotFound 时 user=nil，允许后续返回用户不存在
		if err != gorm.ErrRecordNotFound {
			l.Logger.Errorf("FindByUserID failed, userId=%d, err=%v", in.UserId, err)
			return nil, errorx.ErrDBError(err)
		}
	}
	if user == nil {
		return nil, errorx.New(errorx.CodeUserNotFound)
	}

	// 删除旧头像（逻辑保留但需适配新模式，暂时跳过或通过SysImage清理）
	// if user.AvatarID > 0 { ... }

	// 上传新头像 (通过 UploadSysImageLogic)
	uploadLogic := NewUploadSysImageLogic(l.ctx, l.svcCtx)
	uploadResp, err := uploadLogic.UploadSysImage(&pb.UploadSysImageReq{
		UserId:     in.UserId,
		ImageData:  in.FileData,
		OriginName: in.FileName,
		BizType:    model.SysImageBizTypeAvatar,
	})
	if err != nil {
		return nil, err
	}

	// 更新数据库
	user.AvatarID = uploadResp.Id
	if err := l.svcCtx.UserModel.Update(l.ctx, user); err != nil {
		l.Logger.Errorf("Update user avatar failed, userId=%d, err=%v", in.UserId, err)
		return nil, errorx.New(errorx.CodeUserUpdateFailed)
	}

	// 获取图片URL用于返回
	sysImage, err := l.svcCtx.SysImageModel.FindByID(l.ctx, uploadResp.Id)
	var avatarUrl string
	if err == nil {
		avatarUrl = sysImage.URL
	}

	return &pb.UploadAvatarResp{
		AvatarUrl: avatarUrl,
	}, nil
}
