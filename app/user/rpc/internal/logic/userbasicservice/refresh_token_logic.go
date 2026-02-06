package userbasicservicelogic

import (
	"context"
	"fmt"

	"activity-platform/app/user/rpc/internal/svc"
	"activity-platform/app/user/rpc/pb/pb"
	"activity-platform/common/errorx"
	"activity-platform/common/utils/jwt"

	"github.com/google/uuid"
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
		return nil, errorx.NewDefaultError("无效的刷新令牌")
	}

	// 2. 从redis校验refresh_jwtid是否存在
	// key: token:refresh:{refreshJwtId}
	refreshTokenKey := fmt.Sprintf("token:refresh:%s", claims.RefreshJwtId)
	_, err = l.svcCtx.Redis.Get(l.ctx, refreshTokenKey).Result()
	if err != nil {
		return nil, errorx.NewDefaultError("刷新令牌已过期或不存在")
	}

	// 3. 生成新的短token (AccessJwtId使用新UUID，RefreshJwtId沿用旧的)
	newAccessId := uuid.New().String()
	shortToken, err := jwt.GenerateShortToken(claims.UserId, claims.Role, jwt.AuthConfig{
		Secret: l.svcCtx.Config.JWT.AccessSecret,
		Expire: l.svcCtx.Config.JWT.AccessExpire,
	}, newAccessId, claims.RefreshJwtId)
	if err != nil {
		return nil, errorx.NewSystemError("Token生成失败")
	}

	return &pb.RefreshResponse{
		AccessToken: shortToken.Token,
	}, nil
}
