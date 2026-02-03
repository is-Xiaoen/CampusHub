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
	AccessSecret string
	AccessExpire int64
}

type Claims struct {
	UserId int64 `json:"userId"`
	Role   Role  `json:"role"`
	jwt.RegisteredClaims
}

type TokenResult struct {
	Token    string
	ExpireAt int64
}

func GenerateShortToken(userId int64, role Role, cfg AuthConfig) (TokenResult, error) {
	return generateToken(userId, role, cfg, time.Now())
}

func GenerateLongToken(userId int64, role Role, cfg AuthConfig) (TokenResult, error) {
	return generateToken(userId, role, cfg, time.Now())
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

func generateToken(userId int64, role Role, cfg AuthConfig, now time.Time) (TokenResult, error) {
	if err := ValidateRole(role); err != nil {
		return TokenResult{}, err
	}
	if cfg.AccessSecret == "" || cfg.AccessExpire <= 0 {
		return TokenResult{}, errors.New("invalid auth config")
	}

	expireAt := now.Add(time.Duration(cfg.AccessExpire) * time.Second)
	claims := Claims{
		UserId: userId,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expireAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(cfg.AccessSecret))
	if err != nil {
		return TokenResult{}, err
	}

	return TokenResult{
		Token:    signed,
		ExpireAt: claims.ExpiresAt.Unix(),
	}, nil
}
