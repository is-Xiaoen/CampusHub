package activityservicelogic

import (
	"context"

	"activity-platform/app/activity/rpc/activity"
	"activity-platform/app/activity/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListCategoriesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListCategoriesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListCategoriesLogic {
	return &ListCategoriesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// ==================== 分类标签接口 ====================
func (l *ListCategoriesLogic) ListCategories(in *activity.ListCategoriesReq) (*activity.ListCategoriesResp, error) {
	// todo: add your logic here and delete this line

	return &activity.ListCategoriesResp{}, nil
}
