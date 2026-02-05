// ============================================================================
// JWT 双 Token 工具
// ============================================================================
//
// 负责人：杨春路（B同学 - 用户服务）

package jwt

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v4"
)

type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

var (
	ErrInvalidRole = errors.New("invalid role")
)

type AuthConfig struct {
	Secret string
	Expire int64
}

type Claims struct {
	UserId       int64  `json:"userId"`
	Role         Role   `json:"role"`
	AccessJwtId  string `json:"accessJwtId"`
	RefreshJwtId string `json:"refreshJwtId"`
	jwt.RegisteredClaims
}

type TokenResult struct {
	Token    string
	ExpireAt int64
}

func GenerateShortToken(userId int64, role Role, cfg AuthConfig, accessId, refreshId string) (TokenResult, error) {
	return generateToken(userId, role, cfg, time.Now(), accessId, refreshId)
}

func GenerateLongToken(userId int64, role Role, cfg AuthConfig, accessId, refreshId string) (TokenResult, error) {
	return generateToken(userId, role, cfg, time.Now(), accessId, refreshId)
}

func IsAdmin(ctx context.Context) bool {
	role, ok := GetRoleFromContext(ctx)
	return ok && role == RoleAdmin
}

func IsUser(ctx context.Context) bool {
	role, ok := GetRoleFromContext(ctx)
	return ok && role == RoleUser
}

func GetRoleFromContext(ctx context.Context) (Role, bool) {
	if ctx == nil {
		return "", false
	}
	value := ctx.Value("role")
	switch v := value.(type) {
	case string:
		return Role(v), v != ""
	case []byte:
		if len(v) == 0 {
			return "", false
		}
		return Role(string(v)), true
	default:
		if value == nil {
			return "", false
		}
		role := fmt.Sprint(value)
		if role == "" {
			return "", false
		}
		return Role(role), true
	}
}

func ValidateRole(role Role) error {
	if role != RoleUser && role != RoleAdmin {
		return ErrInvalidRole
	}
	return nil
}

func generateToken(userId int64, role Role, cfg AuthConfig, now time.Time, accessId, refreshId string) (TokenResult, error) {
	if err := ValidateRole(role); err != nil {
		return TokenResult{}, err
	}
	if cfg.Secret == "" || cfg.Expire <= 0 {
		return TokenResult{}, errors.New("invalid auth config")
	}

	expireAt := now.Add(time.Duration(cfg.Expire) * time.Second)
	claims := Claims{
		UserId:       userId,
		Role:         role,
		AccessJwtId:  accessId,
		RefreshJwtId: refreshId,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expireAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(cfg.Secret))
	if err != nil {
		return TokenResult{}, err
	}

	return TokenResult{
		Token:    signed,
		ExpireAt: claims.ExpiresAt.Unix(),
	}, nil
}

func ParseToken(tokenString string, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid token")
}

func CheckTokenBlacklist(ctx context.Context, rdb *redis.Client, tokenStr string, secret string) (bool, error) {
	claims, err := ParseToken(tokenStr, secret)
	if err != nil {
		return false, err
	}

	key := fmt.Sprintf("token:blacklist:access:%s", claims.AccessJwtId)
	exists, err := rdb.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}
