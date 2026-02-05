package svc

import (
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v4"
)

var (
	ErrInvalidToken = errors.New("Token 无效")
	ErrExpiredToken = errors.New("Token 已过期")
)

// JwtAuth JWT 认证工具
type JwtAuth struct {
	secret []byte
}

// NewJwtAuth 创建 JWT 认证工具
func NewJwtAuth(secret string) *JwtAuth {
	return &JwtAuth{
		secret: []byte(secret),
	}
}

// ParseToken 解析 JWT token，返回 userId (string)
func (j *JwtAuth) ParseToken(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// 验证签名算法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.secret, nil
	})

	if err != nil {
		if ve, ok := err.(*jwt.ValidationError); ok {
			if ve.Errors&jwt.ValidationErrorExpired != 0 {
				return "", ErrExpiredToken
			}
		}
		return "", ErrInvalidToken
	}

	if !token.Valid {
		return "", ErrInvalidToken
	}

	// 提取 claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", ErrInvalidToken
	}

	// 提取 userId
	userID, ok := claims["userId"]
	if !ok {
		return "", errors.New("Token 中未找到 userId")
	}

	// 转换为 string
	return fmt.Sprintf("%v", userID), nil
}
