package userbasicservicelogic

import (
	"context"
	"fmt"

	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"
	"activity-platform/common/errorx"
	"activity-platform/common/utils/jwt"

	"github.com/zeromicro/go-zero/core/logx"
)

type RefreshTokenLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRefreshTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RefreshTokenLogic {
	return &RefreshTokenLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 刷新短token
func (l *RefreshTokenLogic) RefreshToken(in *pb.RefreshReq) (*pb.RefreshResponse, error) {
	// 1. 解析长token
	claims, err := jwt.ParseToken(in.RefreshToken, l.svcCtx.Config.JWT.RefreshSecret)
	if err != nil {
		return nil, errorx.New(errorx.CodeRefreshTokenInvalid)
	}

	// 2. 从redis校验refresh_token是否存在且匹配
	// key: token:refresh:user:{userId}
	refreshTokenKey := fmt.Sprintf("token:refresh:user:%d", claims.UserId)
	storedToken, err := l.svcCtx.Redis.Get(l.ctx, refreshTokenKey).Result()
	if err != nil || storedToken != in.RefreshToken {
		return nil, errorx.New(errorx.CodeRefreshTokenExpired)
	}

	// 3. 生成新的短token
	shortToken, err := jwt.GenerateShortToken(claims.UserId, claims.Role, jwt.AuthConfig{
		Secret: l.svcCtx.Config.JWT.AccessSecret,
		Expire: l.svcCtx.Config.JWT.AccessExpire,
	})
	if err != nil {
		return nil, errorx.New(errorx.CodeTokenGenerateFailed)
	}

	return &pb.RefreshResponse{
		AccessToken: shortToken.Token,
	}, nil
}
