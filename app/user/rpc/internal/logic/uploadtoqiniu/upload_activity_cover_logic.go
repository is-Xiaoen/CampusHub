package uploadtoqiniulogic

import (
	"context"

	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type UploadActivityCoverLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUploadActivityCoverLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UploadActivityCoverLogic {
	return &UploadActivityCoverLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// Activity defines the model for updating cover_url
type Activity struct {
	ID       int64  `gorm:"primaryKey;column:id"`
	CoverUrl string `gorm:"column:cover_url"`
}

func (Activity) TableName() string {
	return "campushub_main.activities"
}

// 上传活动封面图片（同时处理旧图删除和DB更新）
func (l *UploadActivityCoverLogic) UploadActivityCover(in *pb.UploadActivityCoverReq) (*pb.UploadActivityCoverResp, error) {
	if in.ActivityId == 0 {
		return nil, errorx.New(errorx.CodeInvalidParams)
	}
	if len(in.FileData) == 0 {
		return nil, errorx.New(errorx.CodeInvalidParams)
	}

	// 1. 查询当前活动信息
	var activity Activity
	err := l.svcCtx.DB.Select("id, cover_url").Where("id = ?", in.ActivityId).First(&activity).Error
	if err != nil {
		l.Errorf("查询活动 %d 失败: %v", in.ActivityId, err)
		return nil, errorx.New(errorx.CodeActivityNotFound)
	}

	// 2. 如果已有封面，先删除
	if activity.CoverUrl != "" {
		if err := deleteFromQiniu(l.ctx, l.svcCtx, activity.CoverUrl); err != nil {
			// 删除失败记录日志，但不阻断流程
			l.Errorf("删除活动 %d 的旧封面失败: %v", in.ActivityId, err)
		}
	}

	// 3. 上传新图片
	url, err := uploadToQiniu(l.ctx, l.svcCtx, in.FileData, in.FileName)
	if err != nil {
		return nil, err
	}

	// 4. 更新数据库
	err = l.svcCtx.DB.Model(&activity).Update("cover_url", url).Error
	if err != nil {
		l.Errorf("更新活动 %d cover_url 失败: %v", in.ActivityId, err)
		return nil, errorx.New(errorx.CodeDBError)
	}

	return &pb.UploadActivityCoverResp{
		CoverUrl: url,
	}, nil
}
