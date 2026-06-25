package credential

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
)

// EnsureSalt loads the salt from saltPath, or generates and persists a new
// 32-byte salt if the file does not exist. The salt file is created with
// 0600 permissions.
func EnsureSalt(saltPath string) ([]byte, error) {
	data, err := os.ReadFile(saltPath)
	if err == nil {
		if len(data) != 32 {
			return nil, fmt.Errorf("salt file has wrong size: %d, want 32", len(data))
		}
		return data, nil
	}
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read salt: %w", err)
	}

	salt := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("generate salt: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(saltPath), 0700); err != nil {
		return nil, fmt.Errorf("create salt dir: %w", err)
	}
	if err := os.WriteFile(saltPath, salt, 0600); err != nil {
		return nil, fmt.Errorf("write salt: %w", err)
	}
	return salt, nil
}

// DeriveKey produces a 32-byte AES-256 key from the machine identity
// (hostname + username) and the salt at saltPath. The salt is created if
// it does not exist. The key is deterministic for the same machine + salt.
func DeriveKey(saltPath string) ([]byte, error) {
	salt, err := EnsureSalt(saltPath)
	if err != nil {
		return nil, err
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("get hostname: %w", err)
	}

	currentUser, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("get current user: %w", err)
	}
	username := currentUser.Username

	// SHA-256(hostname + username + salt) -> 32-byte key
	h := sha256.New()
	h.Write([]byte(hostname))
	h.Write([]byte(username))
	h.Write(salt)
	return h.Sum(nil), nil
}

// Encrypt encrypts plaintext using AES-256-GCM with the provided 32-byte key.
// Returns ciphertext with a random nonce prepended.
func Encrypt(plaintext []byte, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("key must be 32 bytes for AES-256, got %d", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
	return append(nonce, ciphertext...), nil
}

// Decrypt decrypts ciphertext (nonce-prepended AES-256-GCM) using the
// provided 32-byte key.
func Decrypt(ciphertext []byte, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("key must be 32 bytes for AES-256, got %d", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short: %d bytes, need at least %d", len(ciphertext), nonceSize)
	}

	nonce, encrypted := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	return plaintext, nil
}
