package auth

import (
	"strings"
	"testing"
)

func TestPasswordHashing(t *testing.T) {
	hash, err := HashPassword("s3cret-pass")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == "s3cret-pass" {
		t.Error("password stored in plaintext")
	}
	if !CheckPassword(hash, "s3cret-pass") {
		t.Error("correct password rejected")
	}
	if CheckPassword(hash, "wrong") {
		t.Error("wrong password accepted")
	}
	if _, err := HashPassword(""); err == nil {
		t.Error("empty password should error")
	}
}

func TestAPIKeyGeneration(t *testing.T) {
	plain, hash, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey: %v", err)
	}
	if !strings.HasPrefix(plain, APIKeyPrefix) {
		t.Errorf("key missing prefix: %q", plain)
	}
	if hash == plain || len(hash) != 64 {
		t.Errorf("hash looks wrong: %q", hash)
	}
	if HashAPIKey(plain) != hash {
		t.Error("HashAPIKey not stable for the same key")
	}
	// Distinct keys each call.
	plain2, _, _ := GenerateAPIKey()
	if plain2 == plain {
		t.Error("API keys are not unique")
	}
}

func TestGenerateTokenUnique(t *testing.T) {
	a, _ := GenerateToken(32)
	b, _ := GenerateToken(32)
	if a == b || a == "" {
		t.Errorf("tokens not unique/non-empty: %q %q", a, b)
	}
}
