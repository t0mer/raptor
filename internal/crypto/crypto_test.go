package crypto

import (
	"path/filepath"
	"testing"
)

func newCipher(t *testing.T) *Cipher {
	t.Helper()
	key, err := LoadOrCreateKey(filepath.Join(t.TempDir(), "secret.key"))
	if err != nil {
		t.Fatalf("LoadOrCreateKey: %v", err)
	}
	c, err := New(key)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func TestRoundTrip(t *testing.T) {
	c := newCipher(t)
	for _, s := range []string{"", "slack://tok@chan", "a longer secret with spaces & symbols ✓"} {
		enc, err := c.Encrypt(s)
		if err != nil {
			t.Fatalf("Encrypt(%q): %v", s, err)
		}
		if s != "" && enc == s {
			t.Errorf("ciphertext equals plaintext for %q", s)
		}
		dec, err := c.Decrypt(enc)
		if err != nil {
			t.Fatalf("Decrypt: %v", err)
		}
		if dec != s {
			t.Errorf("round-trip = %q, want %q", dec, s)
		}
	}
}

func TestKeyPersistsAndIsStable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret.key")
	k1, err := LoadOrCreateKey(path)
	if err != nil {
		t.Fatal(err)
	}
	k2, err := LoadOrCreateKey(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(k1) != string(k2) {
		t.Error("key changed across loads")
	}
	if len(k1) != KeySize {
		t.Errorf("key size = %d, want %d", len(k1), KeySize)
	}
}

func TestDecryptTampered(t *testing.T) {
	c := newCipher(t)
	enc, _ := c.Encrypt("secret")
	// Flip a character in the base64 to corrupt the ciphertext.
	bad := "A" + enc[1:]
	if _, err := c.Decrypt(bad); err == nil {
		t.Error("expected error decrypting tampered ciphertext")
	}
}
