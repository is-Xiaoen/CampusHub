package sensitivedata

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"activity-platform/app/user/model"
)

const (
	// CipherPrefix 密文前缀，用于识别受支持的密文版本
	CipherPrefix = "enc:v1:"

	fieldRealName  = "real_name"
	fieldStudentID = "student_id"
)

// Codec 提供敏感字段的加解密与学号哈希能力。
type Codec struct {
	gcm     cipher.AEAD
	hashKey []byte
}

// New 创建敏感数据编解码器。
// aesKeyBase64 与 hashKeyBase64 必须是 base64 编码后的 32 字节密钥。
func New(aesKeyBase64, hashKeyBase64 string) (*Codec, error) {
	aesKey, err := decodeBase64Key("SensitiveData.AesKey", aesKeyBase64)
	if err != nil {
		return nil, err
	}
	hashKey, err := decodeBase64Key("SensitiveData.HashKey", hashKeyBase64)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("init aes cipher failed: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("init gcm failed: %w", err)
	}

	return &Codec{
		gcm:     gcm,
		hashKey: hashKey,
	}, nil
}

// IsEncrypted 判断值是否为受支持的密文格式。
func (c *Codec) IsEncrypted(value string) bool {
	return strings.HasPrefix(value, CipherPrefix)
}

// Encrypt 对字段进行 AES-256-GCM 加密。
func (c *Codec) Encrypt(field, plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	nonce := make([]byte, c.gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generate nonce failed: %w", err)
	}

	ciphertext := c.gcm.Seal(nil, nonce, []byte(plaintext), fieldAAD(field))
	raw := append(nonce, ciphertext...)

	return CipherPrefix + base64.StdEncoding.EncodeToString(raw), nil
}

// Decrypt 对字段进行 AES-256-GCM 解密。
func (c *Codec) Decrypt(field, value string) (string, error) {
	if value == "" {
		return "", nil
	}
	if !c.IsEncrypted(value) {
		return "", fmt.Errorf("invalid ciphertext format")
	}

	encoded := strings.TrimPrefix(value, CipherPrefix)
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("decode ciphertext failed: %w", err)
	}

	nonceSize := c.gcm.NonceSize()
	if len(raw) <= nonceSize {
		return "", fmt.Errorf("invalid ciphertext payload")
	}

	nonce := raw[:nonceSize]
	ciphertext := raw[nonceSize:]

	plain, err := c.gcm.Open(nil, nonce, ciphertext, fieldAAD(field))
	if err != nil {
		return "", fmt.Errorf("gcm decrypt failed: %w", err)
	}

	return string(plain), nil
}

// HashStudentID 使用 HMAC-SHA256 生成学号哈希索引值。
func (c *Codec) HashStudentID(schoolName, studentID string) (string, error) {
	school := strings.TrimSpace(schoolName)
	student := strings.TrimSpace(studentID)
	if student == "" {
		return "", fmt.Errorf("student_id is empty")
	}

	mac := hmac.New(sha256.New, c.hashKey)
	_, _ = mac.Write([]byte(school + "\n" + student))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

func decodeBase64Key(name, value string) ([]byte, error) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return nil, fmt.Errorf("%s is required", name)
	}

	key, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		key, err = base64.RawStdEncoding.DecodeString(raw)
		if err != nil {
			return nil, fmt.Errorf("%s must be valid base64: %w", name, err)
		}
	}

	if len(key) != 32 {
		return nil, fmt.Errorf("%s must decode to 32 bytes, got %d", name, len(key))
	}
	return key, nil
}

func fieldAAD(field string) []byte {
	switch strings.TrimSpace(field) {
	case fieldRealName:
		return []byte("verify:real_name:v1")
	case fieldStudentID:
		return []byte("verify:student_id:v1")
	default:
		return []byte("verify:unknown:v1")
	}
}

var _ model.SensitiveDataCodec = (*Codec)(nil)
