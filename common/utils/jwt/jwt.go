// ============================================================================
// JWT 双 Token 工具
// ============================================================================
//
// 负责人：杨春路（B同学 - 用户服务）

package jwt

import "errors"

// ==================== 类型定义（存根，供编译通过） ====================

// JwtConfig JWT 配置
// TODO(杨春路): 完善配置项
type JwtConfig struct {
	AccessSecret  string
	RefreshSecret string
	AccessExpire  int64
	RefreshExpire int64
}

// Claims 自定义 JWT Claims
// TODO(杨春路): 完善 Claims 结构
type Claims struct {
	UserID int64
	Phone  string
}

// TokenPair Token 对
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    int64
}

// ==================== 错误定义 ====================

var ErrTokenExpired = errors.New("token expired")

// ==================== 待实现方法（存根） ====================

// GenerateTokenPair 生成 Token 对
// TODO(杨春路): 实现 JWT 双 Token 生成
func GenerateTokenPair(config *JwtConfig, userID int64, phone string) (*TokenPair, error) {
	panic("TODO: 杨春路实现")
}

// ParseAccessToken 解析 Access Token
// TODO(杨春路): 实现 Access Token 解析
func ParseAccessToken(config *JwtConfig, tokenString string) (*Claims, error) {
	panic("TODO: 杨春路实现")
}

// ParseRefreshToken 解析 Refresh Token
// TODO(杨春路): 实现 Refresh Token 解析
func ParseRefreshToken(config *JwtConfig, tokenString string) (*Claims, error) {
	panic("TODO: 杨春路实现")
}

// RefreshTokenPair 刷新 Token 对
// TODO(杨春路): 实现 Token 刷新
func RefreshTokenPair(config *JwtConfig, refreshTokenString string) (*TokenPair, error) {
	panic("TODO: 杨春路实现")
}

// IsTokenExpired 判断是否是过期错误
// TODO(杨春路): 实现过期判断
func IsTokenExpired(err error) bool {
	return errors.Is(err, ErrTokenExpired)
}
