// ============================================================================
// 测试 Token 生成脚本
// ============================================================================
//
// 用途：生成用于 Apifox 测试的 JWT Token
// 运行：go run scripts/gen_test_token.go
//
// ============================================================================

package main

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

func main() {
	// 与 deploy/docker/config/*.yaml 中的 AccessSecret 保持一致
	accessSecret := "your-access-secret-key-change-in-production-32chars"

	// 测试用户ID（需要与数据库中的用户ID一致）
	testUserID := int64(10001)

	// Token 有效期：2年（长期测试使用）
	expireAt := time.Now().Add(2 * 365 * 24 * time.Hour).Unix()

	// 创建 Token（包含 role 字段，与 common/utils/jwt/jwt.go 结构一致）
	claims := jwt.MapClaims{
		"userId": testUserID,
		"role":   "user", // 普通用户角色
		"exp":    expireAt,
		"iat":    time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(accessSecret))
	if err != nil {
		fmt.Printf("生成 Token 失败: %v\n", err)
		return
	}

	fmt.Println("============================================")
	fmt.Println("测试 JWT Token 生成成功！")
	fmt.Println("============================================")
	fmt.Printf("用户ID: %d\n", testUserID)
	fmt.Printf("角色: user\n")
	fmt.Printf("过期时间: %s\n", time.Unix(expireAt, 0).Format("2006-01-02 15:04:05"))
	fmt.Println("--------------------------------------------")
	fmt.Println("Token:")
	fmt.Println(tokenString)
	fmt.Println("--------------------------------------------")
	fmt.Println("Apifox Header 配置:")
	fmt.Printf("Authorization: Bearer %s\n", tokenString)
	fmt.Println("============================================")
}
