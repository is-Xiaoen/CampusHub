// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"context"
	"net/http"
	"strings"

	"activity-platform/app/user/api/internal/svc"
	"activity-platform/app/user/rpc/client/userbasicservice"

	"github.com/zeromicro/go-zero/core/logx"
)

type LogoutLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

// 退出登录
func NewLogoutLogic(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *LogoutLogic {
	return &LogoutLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *LogoutLogic) Logout() error {
	// 获取 Authorization Header
	authHeader := l.r.Header.Get("Authorization")
	if authHeader == "" {
		return nil // 没有 token，视为已登出
	}

	// 去除 "Bearer " 前缀
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return nil // 格式不对，忽略
	}
	token := parts[1]

	// 调用 RPC 登出
	_, err := l.svcCtx.UserBasicServiceRpc.Logout(l.ctx, &userbasicservice.LogoutReq{
		AccessToken: token,
	})
	if err != nil {
		l.Logger.Errorf("RPC Logout failed: %v", err)
		return err
	}

	return nil
}
