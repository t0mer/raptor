// Package auth provides authentication primitives for Raptor: password hashing,
// API-key generation/hashing, and session-token generation.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// APIKeyPrefix marks Raptor API keys for easy identification.
const APIKeyPrefix = "rpt_"

// HashPassword returns a bcrypt hash of a plaintext password.
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", fmt.Errorf("password must not be empty")
	}
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(h), nil
}

// CheckPassword reports whether password matches the bcrypt hash.
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// GenerateAPIKey returns a new API key: the plaintext (shown to the user once,
// prefixed rpt_) and its SHA-256 hash (stored for lookup). API keys are random
// 256-bit tokens, so a fast hash is appropriate (unlike low-entropy passwords).
func GenerateAPIKey() (plaintext, hash string, err error) {
	raw := make([]byte, 32)
	if _, err = io.ReadFull(rand.Reader, raw); err != nil {
		return "", "", err
	}
	plaintext = APIKeyPrefix + base64.RawURLEncoding.EncodeToString(raw)
	return plaintext, HashAPIKey(plaintext), nil
}

// HashAPIKey returns the SHA-256 hex hash of an API key for storage/lookup.
func HashAPIKey(key string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(key)))
	return hex.EncodeToString(sum[:])
}

// ConstantTimeEqual compares two strings without leaking length-independent
// timing (used for token comparison).
func ConstantTimeEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// GenerateToken returns a URL-safe random token of n bytes of entropy.
func GenerateToken(n int) (string, error) {
	if n <= 0 {
		n = 32
	}
	raw := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
