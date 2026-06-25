package credential

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEncryptDecryptRoundtrip(t *testing.T) {
	key := make([]byte, 32) // 32-byte key for AES-256
	for i := range key {
		key[i] = byte(i)
	}

	plaintext := []byte(`{"providers":{"openai":{"api_key":"**********","added_at":"2026-06-25T12:00:00Z"}}}`)

	ciphertext, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt error: %v", err)
	}

	decrypted, err := Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("Decrypt error: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("roundtrip mismatch: got %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptProducesDifferentCiphertext(t *testing.T) {
	key := make([]byte, 32)
	plaintext := []byte("test data")

	ct1, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt 1: %v", err)
	}
	ct2, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt 2: %v", err)
	}

	// GCM uses a random nonce, so same plaintext produces different ciphertext
	if string(ct1) == string(ct2) {
		t.Error("same plaintext should produce different ciphertext due to random nonce")
	}
}

func TestDecryptWithWrongKeyFails(t *testing.T) {
	key1 := make([]byte, 32)
	key1[0] = 1
	key2 := make([]byte, 32)
	key2[0] = 2

	plaintext := []byte("secret")
	ciphertext, err := Encrypt(plaintext, key1)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	_, err = Decrypt(ciphertext, key2)
	if err == nil {
		t.Fatal("Decrypt with wrong key should fail")
	}
}

func TestEnsureSaltCreatesAndPersists(t *testing.T) {
	saltPath := filepath.Join(t.TempDir(), ".salt")

	salt1, err := EnsureSalt(saltPath)
	if err != nil {
		t.Fatalf("EnsureSalt 1: %v", err)
	}
	if len(salt1) != 32 {
		t.Errorf("salt length = %d, want 32", len(salt1))
	}

	// Second call should read the same salt from disk
	salt2, err := EnsureSalt(saltPath)
	if err != nil {
		t.Fatalf("EnsureSalt 2: %v", err)
	}
	if string(salt1) != string(salt2) {
		t.Error("salt should be deterministic after first creation")
	}

	// Verify file permissions
	info, err := os.Stat(saltPath)
	if err != nil {
		t.Fatalf("stat salt: %v", err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Errorf("salt file mode = %o, want 0600", got)
	}
}

func TestDeriveKeyDeterministicForSameInputs(t *testing.T) {
	dir := t.TempDir()
	saltPath1 := filepath.Join(dir, "salt1")
	saltPath2 := filepath.Join(dir, "salt2")

	// Same salt -> same key (on same machine, hostname+username are constant)
	salt, err := EnsureSalt(saltPath1)
	if err != nil {
		t.Fatalf("EnsureSalt: %v", err)
	}
	key1, err := DeriveKey(saltPath1)
	if err != nil {
		t.Fatalf("DeriveKey 1: %v", err)
	}

	// Manually write the same salt to a different path
	if err := os.WriteFile(saltPath2, salt, 0600); err != nil {
		t.Fatalf("write salt2: %v", err)
	}
	key2, err := DeriveKey(saltPath2)
	if err != nil {
		t.Fatalf("DeriveKey 2: %v", err)
	}

	if string(key1) != string(key2) {
		t.Error("keys should be identical for same salt on same machine")
	}
	if len(key1) != 32 {
		t.Errorf("key length = %d, want 32", len(key1))
	}
}

func TestDeriveKeyDifferentSaltProducesDifferentKey(t *testing.T) {
	dir := t.TempDir()
	saltPath1 := filepath.Join(dir, "salt1")
	saltPath2 := filepath.Join(dir, "salt2")

	key1, err := DeriveKey(saltPath1)
	if err != nil {
		t.Fatalf("DeriveKey 1: %v", err)
	}
	key2, err := DeriveKey(saltPath2)
	if err != nil {
		t.Fatalf("DeriveKey 2: %v", err)
	}

	if string(key1) == string(key2) {
		t.Error("different salts should produce different keys")
	}
}
