package util

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// GenerateToken returns a random 32-byte hex string.
func GenerateToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// RandomSuffix returns a short random hex suffix.
func RandomSuffix() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// GenerateID returns prefix_YYYYMMDDHHMMSS_<suffix>.
func GenerateID(prefix string) string {
	return prefix + "_" + time.Now().Format("20060102150405") + "_" + RandomSuffix()
}
