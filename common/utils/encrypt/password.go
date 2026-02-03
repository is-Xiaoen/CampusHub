package encrypt

import (
	"crypto/sha256"
	"encoding/hex"
)

func EncryptPassword(password string) string {
	sum := sha256.Sum256([]byte(password))
	return hex.EncodeToString(sum[:])
}

func ComparePassword(rawPassword, encryptedPassword string) bool {
	return EncryptPassword(rawPassword) == encryptedPassword
}
