package forum

import (
	"crypto/sha256"
	"encoding/hex"
)

func Hash(password string) string {
	hash := sha256.New()
	hash.Write([]byte(password))
	hashInBytes := hash.Sum(nil)
	return hex.EncodeToString(hashInBytes)
}
