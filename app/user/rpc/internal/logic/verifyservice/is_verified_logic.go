package verifyservicelogic

import (
	"context"

	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type IsVerifiedLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewIsVerifiedLogic(ctx context.Context, svcCtx *svc.ServiceContext) *IsVerifiedLogic {
	return &IsVerifiedLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// IsVerified 查询用户是否已完成学生认证
func (l *IsVerifiedLogic) IsVerified(in *pb.IsVerifiedReq) (*pb.IsVerifiedResp, error) {
	// todo: add your logic here and delete this line

	return &pb.IsVerifiedResp{}, nil
}
