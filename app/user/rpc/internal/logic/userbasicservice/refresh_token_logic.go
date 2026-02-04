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
	claims, err := jwt.ParseToken(in.RefreshToken, l.svcCtx.Config.RefreshAuth.AccessSecret)
	if err != nil {
		return nil, errorx.NewDefaultError("无效的刷新令牌")
	}

	// 2. 从redis获取存储的长token并比较
	refreshTokenKey := fmt.Sprintf("refresh_token:%d", claims.UserId)
	storedToken, err := l.svcCtx.Redis.Get(l.ctx, refreshTokenKey).Result()
	if err != nil {
		return nil, errorx.NewDefaultError("刷新令牌已过期或不存在")
	}

	if storedToken != in.RefreshToken {
		return nil, errorx.NewDefaultError("刷新令牌无效")
	}

	// 3. 生成新的短token
	shortToken, err := jwt.GenerateShortToken(claims.UserId, claims.Role, jwt.AuthConfig(l.svcCtx.Config.Auth))
	if err != nil {
		return nil, errorx.NewSystemError("Token生成失败")
	}

	return &pb.RefreshResponse{
		AccessToken: shortToken.Token,
	}, nil
}
