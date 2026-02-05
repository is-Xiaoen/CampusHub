package main

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// 从配置文件中获取的密钥
const AccessSecret = "k9#8G7&6F5%4D3$2S1@0P9*8O7!6N5^4M3+2L1=0"

// Claims JWT 声明
type Claims struct {
	UserID int64  `json:"userId"`
	Phone  string `json:"phone"`
	jwt.RegisteredClaims
}

func main() {
	// 测试用户信息
	userID := int64(1)
	phone := "13800138000"

	// 创建 claims
	now := time.Now()
	claims := Claims{
		UserID: userID,
		Phone:  phone,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(2 * time.Hour)), // 2小时过期
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	// 生成 token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(AccessSecret))
	if err != nil {
		fmt.Printf("生成 token 失败: %v\n", err)
		return
	}

	fmt.Println("=== 测试 Token 生成成功 ===")
	fmt.Printf("UserID: %d\n", userID)
	fmt.Printf("Phone: %s\n", phone)
	fmt.Printf("过期时间: %s\n", claims.ExpiresAt.Time.Format("2006-01-02 15:04:05"))
	fmt.Println("\nToken:")
	fmt.Println(tokenString)
	fmt.Println("\n使用方法:")
	fmt.Println("在请求头中添加: Authorization: Bearer " + tokenString)
}
