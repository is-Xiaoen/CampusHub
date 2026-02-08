package userbasicservicelogic

import (
	"context"

	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type CheckUserExistsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCheckUserExistsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CheckUserExistsLogic {
	return &CheckUserExistsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 检查用户是否存在（通过邮箱）
func (l *CheckUserExistsLogic) CheckUserExists(in *pb.CheckUserExistsReq) (*pb.CheckUserExistsResponse, error) {
	exists, err := l.svcCtx.UserModel.ExistsByQQEmail(l.ctx, in.QqEmail)
	if err != nil {
		l.Logger.Errorf("CheckUserExists error: %v, email: %s", err, in.QqEmail)
		return nil, err
	}

	return &pb.CheckUserExistsResponse{
		Exists: exists,
	}, nil
}
