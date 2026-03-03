package encrypt

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/crypto/argon2"
)

func EncryptPassword(password string) string {
	salt := make([]byte, 16)
	_, _ = rand.Read(salt)
	time := uint32(3)
	memory := uint32(64 * 1024)
	threads := uint8(2)
	keyLen := uint32(32)
	hash := argon2.IDKey([]byte(password), salt, time, memory, threads, keyLen)
	saltB64 := base64.RawStdEncoding.EncodeToString(salt)
	hashB64 := base64.RawStdEncoding.EncodeToString(hash)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", memory, time, threads, saltB64, hashB64)
}

func ComparePassword(rawPassword, encryptedPassword string) bool {
	parts := strings.Split(encryptedPassword, "$")
	if len(parts) != 6 {
		return false
	}
	if parts[1] != "argon2id" {
		return false
	}
	params := strings.TrimPrefix(parts[3], "m=")
	ps := strings.Split(params, ",")
	if len(ps) != 3 {
		return false
	}
	mStr := strings.TrimPrefix(ps[0], "")
	tStr := strings.TrimPrefix(ps[1], "t=")
	pStr := strings.TrimPrefix(ps[2], "p=")
	memory, err1 := strconv.Atoi(strings.TrimPrefix(mStr, ""))
	time, err2 := strconv.Atoi(tStr)
	threadsInt, err3 := strconv.Atoi(pStr)
	if err1 != nil || err2 != nil || err3 != nil {
		return false
	}
	saltB64 := parts[4]
	hashB64 := parts[5]
	salt, err := base64.RawStdEncoding.DecodeString(saltB64)
	if err != nil {
		return false
	}
	storedHash, err := base64.RawStdEncoding.DecodeString(hashB64)
	if err != nil {
		return false
	}
	keyLen := uint32(len(storedHash))
	computed := argon2.IDKey([]byte(rawPassword), salt, uint32(time), uint32(memory), uint8(threadsInt), keyLen)
	return bytes.Equal(computed, storedHash)
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
