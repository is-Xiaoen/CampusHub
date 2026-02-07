package uploadtoqiniulogic

import (
	"context"
	"errors"

	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type UploadStudentCardImagesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUploadStudentCardImagesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UploadStudentCardImagesLogic {
	return &UploadStudentCardImagesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// UploadStudentCardImages 上传学生认证图片（同时处理旧图删除和DB更新）
func (l *UploadStudentCardImagesLogic) UploadStudentCardImages(in *pb.UploadStudentCardImagesReq) (*pb.UploadStudentCardImagesResp, error) {
	// 1. 查询用户认证记录
	// 我们需要知道之前是否上传过图片，如果有，需要删除旧图
	verification, err := l.svcCtx.StudentVerificationModel.FindByUserID(l.ctx, in.UserId)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		l.Logger.Errorf("Failed to find verification record: %v", err)
		return nil, err
	}

	// 2. 如果存在记录，尝试删除旧图片
	if verification != nil {
		if verification.FrontImageURL != "" {
			if err := deleteFromQiniu(l.ctx, l.svcCtx, verification.FrontImageURL); err != nil {
				l.Logger.Errorf("Failed to delete old front image: %v", err)
				// 继续执行，不阻塞上传流程
			}
		}
		if verification.BackImageURL != "" {
			if err := deleteFromQiniu(l.ctx, l.svcCtx, verification.BackImageURL); err != nil {
				l.Logger.Errorf("Failed to delete old back image: %v", err)
				// 继续执行
			}
		}
	}

	// 3. 上传新图片
	frontUrl, err := uploadToQiniu(l.ctx, l.svcCtx, in.FrontImageData, in.FrontImageName)
	if err != nil {
		return nil, err
	}

	backUrl, err := uploadToQiniu(l.ctx, l.svcCtx, in.BackImageData, in.BackImageName)
	if err != nil {
		// 如果第二张上传失败，尝试清理第一张（尽力而为）
		_ = deleteFromQiniu(l.ctx, l.svcCtx, frontUrl)
		return nil, err
	}

	// 4. 如果记录存在，更新数据库中的URL
	// 这一步是为了确保下次调用时能找到并删除这些新上传的图片（作为旧图片）
	// 如果记录不存在（首次申请），通常由后续的 ApplyStudentVerify 接口创建记录并保存URL
	if verification != nil {
		verification.FrontImageURL = frontUrl
		verification.BackImageURL = backUrl
		if err := l.svcCtx.StudentVerificationModel.Update(l.ctx, verification); err != nil {
			l.Logger.Errorf("Failed to update verification record with new images: %v", err)
			return nil, err
		}
	}

	return &pb.UploadStudentCardImagesResp{
		FrontImageUrl: frontUrl,
		BackImageUrl:  backUrl,
	}, nil
}
