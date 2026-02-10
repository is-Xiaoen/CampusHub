// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package base

import (
	"context"

	"activity-platform/app/user/api/internal/svc"
	"activity-platform/app/user/api/internal/types"
	"activity-platform/app/user/rpc/client/qqemail"
	"activity-platform/app/user/rpc/client/userbasicservice"
	"activity-platform/common/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetRegisterCodeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取注册QQ验证码
func NewGetRegisterCodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetRegisterCodeLogic {
	return &GetRegisterCodeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetRegisterCodeLogic) GetRegisterCode(req *types.GetRegisterCodeReq) (resp *types.GetCodeResp, err error) {
	// 1. 检查用户是否已存在
	existsResp, err := l.svcCtx.UserBasicServiceRpc.CheckUserExists(l.ctx, &userbasicservice.CheckUserExistsReq{
		QqEmail: req.QqEmail,
	})
	if err != nil {
		return nil, err
	}
	if existsResp.Exists {
		// 如果用户已存在，返回错误码 CodeUserEmailAlreadyExists (2016)
		return nil, errorx.New(errorx.CodeUserEmailAlreadyExists)
	}

	// 2. 发送验证码
	_, err = l.svcCtx.QQEmailRpc.SendQQEmail(l.ctx, &qqemail.SendQQEmailReq{
		QqEmail: req.QqEmail,
		Scene:   "register",
	})
	if err != nil {
		return nil, err
	}

	return &types.GetCodeResp{}, nil
}
