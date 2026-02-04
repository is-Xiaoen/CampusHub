package userbasicservicelogic

import (
	"context"
	"fmt"
	"time"

	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"
	"activity-platform/common/utils/jwt"

	"github.com/zeromicro/go-zero/core/logx"
)

type LogoutLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLogoutLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LogoutLogic {
	return &LogoutLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 用户登出
func (l *LogoutLogic) Logout(in *pb.LogoutReq) (*pb.LogoutResponse, error) {
	// 1. 解析 Access Token
	claims, err := jwt.ParseToken(in.AccessToken, l.svcCtx.Config.Auth.AccessSecret)
	if err != nil {
		l.Logger.Errorf("Parse token failed: %v", err)
		// 即使解析失败，也尝试继续处理（可能只是为了登出），或者直接返回成功
		// 但如果无法解析，就拿不到 ID，所以这里如果失败，可能意味着已经是无效 token
		return &pb.LogoutResponse{}, nil
	}

	// 2. 将 AccessJwtId 加入黑名单
	// key: token:blacklist:access:{accessJwtId}
	accessBlacklistKey := fmt.Sprintf("token:blacklist:access:%s", claims.AccessJwtId)
	// 计算剩余有效期
	remain := time.Until(claims.ExpiresAt.Time)
	if remain > 0 {
		err := l.svcCtx.Redis.Set(l.ctx, accessBlacklistKey, "1", remain).Err()
		if err != nil {
			l.Logger.Errorf("Set access token blacklist failed: %v", err)
		}
	}

	// 3. 删除 RefreshJwtId
	// key: token:refresh:{refreshJwtId}
	refreshKey := fmt.Sprintf("token:refresh:%s", claims.RefreshJwtId)
	err = l.svcCtx.Redis.Del(l.ctx, refreshKey).Err()
	if err != nil {
		l.Logger.Errorf("Del refresh token failed: %v", err)
	}

	return &pb.LogoutResponse{}, nil
}
