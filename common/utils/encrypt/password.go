package encrypt

import (
	"crypto/sha256"
	"encoding/hex"
	"unicode"
)

func EncryptPassword(password string) string {
	sum := sha256.Sum256([]byte(password))
	return hex.EncodeToString(sum[:])
}

func ComparePassword(rawPassword, encryptedPassword string) bool {
	return EncryptPassword(rawPassword) == encryptedPassword
}

// ValidatePassword 校验密码格式
// 长度：8 ~ 20 位
// 字符类型：必须包含 至少 3 种 类型（大写、小写、数字、特殊符号）
func ValidatePassword(password string) bool {
	if len(password) < 8 || len(password) > 20 {
		return false
	}

	var (
		hasUpper   bool
		hasLower   bool
		hasNumber  bool
		hasSpecial bool
	)

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	typesCount := 0
	if hasUpper {
		typesCount++
	}
	if hasLower {
		typesCount++
	}
	if hasNumber {
		typesCount++
	}
	if hasSpecial {
		typesCount++
	}

	return typesCount >= 3
}
