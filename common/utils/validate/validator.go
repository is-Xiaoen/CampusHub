package validate

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// 预编译正则表达式，提升性能
var (
	phoneRegex    = regexp.MustCompile(`^1[3-9]\d{9}$`)
	emailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_\p{Han}]{2,20}$`)
	idCardRegex   = regexp.MustCompile(`^[1-9]\d{5}(19|20)\d{2}(0[1-9]|1[0-2])(0[1-9]|[12]\d|3[01])\d{3}[\dXx]$`)
)

// IsValidPhone 验证手机号（中国大陆）

// IsValidEmail 验证邮箱格式

// IsValidUsername 验证用户名

// IsValidPassword 验证密码强度

// IsValidIDCard 验证身份证号（18位）
func IsValidIDCard(idCard string) bool {
	if !idCardRegex.MatchString(idCard) {
		return false
	}

	// 校验位验证
	return verifyIDCardChecksum(idCard)
}

// verifyIDCardChecksum 身份证校验位验证
func verifyIDCardChecksum(idCard string) bool {
	weights := []int{7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2}
	checkCodes := []byte{'1', '0', 'X', '9', '8', '7', '6', '5', '4', '3', '2'}

	sum := 0
	for i := 0; i < 17; i++ {
		digit := int(idCard[i] - '0')
		sum += digit * weights[i]
	}

	checkCode := checkCodes[sum%11]
	lastChar := idCard[17]
	if lastChar == 'x' {
		lastChar = 'X'
	}

	return checkCode == lastChar
}

// IsNotBlank 判断字符串不为空白
func IsNotBlank(s string) bool {
	return strings.TrimSpace(s) != ""
}

// IsBlank 判断字符串为空白
func IsBlank(s string) bool {
	return strings.TrimSpace(s) == ""
}

// LengthBetween 判断字符串长度在范围内（按字符数，非字节数）
func LengthBetween(s string, min, max int) bool {
	length := utf8.RuneCountInString(s)
	return length >= min && length <= max
}

// MaxLength 判断字符串长度不超过最大值
func MaxLength(s string, max int) bool {
	return utf8.RuneCountInString(s) <= max
}

// MinLength 判断字符串长度不少于最小值
func MinLength(s string, min int) bool {
	return utf8.RuneCountInString(s) >= min
}

// InRange 判断整数在范围内
func InRange(n, min, max int) bool {
	return n >= min && n <= max
}

// InRange64 判断 int64 在范围内
func InRange64(n, min, max int64) bool {
	return n >= min && n <= max
}

// IsPositive 判断是否为正数
func IsPositive(n int64) bool {
	return n > 0
}

// IsNonNegative 判断是否为非负数
func IsNonNegative(n int64) bool {
	return n >= 0
}

// Contains 判断字符串是否在列表中
func Contains(s string, list []string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}
	return false
}

// ContainsInt 判断整数是否在列表中
func ContainsInt(n int, list []int) bool {
	for _, item := range list {
		if item == n {
			return true
		}
	}
	return false
}
