# Phase 2: Credential System & Auth Commands

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Create an encrypted credential store, update the config format to multi-provider, add `auth add/remove/list` commands, and wire `gen`/`edit` to read credentials from the new store instead of the old flat config.

**Architecture:** A new `internal/credential/` package handles AES-256-GCM encryption with a machine-derived key. A new `internal/auth/` package manages credential lifecycle (add, remove, list, get). The config package is updated to a multi-provider YAML format with `active_provider`/`active_model` and per-provider settings. New CLI commands `auth add`, `auth remove`/`rm`, `auth list`/`ls` provide non-interactive credential management. The `gen`/`edit` commands now resolve credentials from the store before calling the adapter.

**Tech Stack:** Go 1.26, `crypto/aes` + `crypto/cipher` (AES-GCM), `crypto/sha256` (KDF), `os/user` + `os.Hostname` (machine identity), Cobra CLI, `gopkg.in/yaml.v3`

## Global Constraints

- Go 1.26, pure Go only (no CGO, no system keyring dependencies)
- Standard `gofmt` formatting. Run `gofmt -l .` before committing; fix any flagged files
- No panics in library code. Use `fmt.Errorf` with `%w` for error wrapping
- No bare `interface{}` / bare `any` in domain sigs
- No `_ = err` (every (T, error) must be checked)
- `context.Context` as first param where applicable
- Keep files under 250 pure LOC
- Table-driven tests preferred. Test files sit alongside source: `foo.go` / `foo_test.go`
- CLI tests dispatch via `rootCmd.SetArgs([]string{...})` and `rootCmd.Execute()`
- Use `t.TempDir()` for temp files and `t.Setenv()` for env vars in tests
- Exit codes: 0 success, 2 config error, 3 API error, 4 image error
- Module path: `github.com/ncxton/potaco`
- `internal/adapter/` package already exists from Phase 1 with `Adapter` interface, `Get(name, apiKey, opts)`, `List()`, `AdapterOpts{BaseURL, Timeout, Retries}`
- `internal/adapter/openai/` already registered via `init()`
- `internal/cli/helpers.go` has `adapterForProvider(cfg)` that reads `cfg.Provider` (defaults to "openai"), `cfg.BaseURL`, `cfg.APIKey`, `cfg.Retries`, `cfg.Timeout`
- `internal/config/` has `Config{BaseURL, APIKey, Model, Retries, Timeout, Provider}`, `FileConfig` with flat `default` section, `Merge(opts)`, `Load(path)`, `FromEnv()`

---

### Task 1: Encryption Core (Machine-Derived Key + AES-GCM)

**Files:**
- Create: `internal/credential/encrypt.go`
- Create: `internal/credential/encrypt_test.go`

**Interfaces:**
- Produces: `DeriveKey(saltPath string) ([]byte, error)`, `Encrypt(plaintext []byte, key []byte) ([]byte, error)`, `Decrypt(ciphertext []byte, key []byte) ([]byte, error)`, `EnsureSalt(saltPath string) ([]byte, error)`

- [ ] **Step 1: Write the failing test**

```go
// internal/credential/encrypt_test.go
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

	plaintext := []byte(`{"providers":{"openai":{"api_key":"sk-test123","added_at":"2026-06-25T12:00:00Z"}}}`)

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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/credential/ -v`
Expected: FAIL with "package credential is not in std" or "no such package"

- [ ] **Step 3: Write minimal implementation**

```go
// internal/credential/encrypt.go
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/credential/ -v`
Expected: PASS (6 tests)

- [ ] **Step 5: Commit**

```bash
git add internal/credential/encrypt.go internal/credential/encrypt_test.go
git commit -m "credential: add AES-256-GCM encryption with machine-derived key"
```

---

### Task 2: Credential Store (Encrypted File Read/Write)

**Files:**
- Create: `internal/credential/store.go`
- Create: `internal/credential/store_test.go`
- Create: `internal/credential/types.go`

**Interfaces:**
- Consumes: `Encrypt([]byte, []byte) ([]byte, error)`, `Decrypt([]byte, []byte) ([]byte, error)`, `DeriveKey(string) ([]byte, error)` (from Task 1)
- Produces: `CredentialStore` struct with `New(credPath, saltPath string) (*CredentialStore, error)`, `Get(provider string) (string, error)`, `Set(provider, apiKey string) error`, `Remove(provider string) error`, `List() []string`

- [ ] **Step 1: Write the failing test**

```go
// internal/credential/store_test.go
package credential

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newTestStore(t *testing.T) (*CredentialStore, string) {
	t.Helper()
	dir := t.TempDir()
	credPath := filepath.Join(dir, "credentials.enc")
	saltPath := filepath.Join(dir, ".salt")
	store, err := New(credPath, saltPath)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return store, credPath
}

func TestStoreSetAndGet(t *testing.T) {
	store, _ := newTestStore(t)

	if err := store.Set("openai", "sk-test-key-123"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	key, err := store.Get("openai")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if key != "sk-test-key-123" {
		t.Errorf("Get = %q, want 'sk-test-key-123'", key)
	}
}

func TestStoreGetMissingProvider(t *testing.T) {
	store, _ := newTestStore(t)

	_, err := store.Get("nonexistent")
	if err == nil {
		t.Fatal("Get should error for missing provider")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found', got: %v", err)
	}
}

func TestStoreRemove(t *testing.T) {
	store, _ := newTestStore(t)

	store.Set("openai", "sk-test")
	store.Set("fal", "fal-key")

	if err := store.Remove("openai"); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	_, err := store.Get("openai")
	if err == nil {
		t.Error("Get should fail after Remove")
	}

	// fal should still be present
	key, err := store.Get("fal")
	if err != nil {
		t.Fatalf("Get fal after removing openai: %v", err)
	}
	if key != "fal-key" {
		t.Errorf("fal key = %q, want 'fal-key'", key)
	}
}

func TestStoreRemoveMissingProvider(t *testing.T) {
	store, _ := newTestStore(t)

	err := store.Remove("nonexistent")
	if err == nil {
		t.Fatal("Remove should error for missing provider")
	}
}

func TestStoreList(t *testing.T) {
	store, _ := newTestStore(t)

	store.Set("openai", "sk-1")
	store.Set("fal", "sk-2")
	store.Set("vercel", "sk-3")

	providers := store.List()
	if len(providers) != 3 {
		t.Fatalf("List len = %d, want 3", len(providers))
	}

	found := map[string]bool{}
	for _, p := range providers {
		found[p] = true
	}
	if !found["openai"] || !found["fal"] || !found["vercel"] {
		t.Errorf("List missing providers, got: %v", providers)
	}
}

func TestStoreListEmpty(t *testing.T) {
	store, _ := newTestStore(t)

	providers := store.List()
	if len(providers) != 0 {
		t.Errorf("List on empty store should return empty, got %v", providers)
	}
}

func TestStorePersistsAcrossInstances(t *testing.T) {
	dir := t.TempDir()
	credPath := filepath.Join(dir, "credentials.enc")
	saltPath := filepath.Join(dir, ".salt")

	store1, err := New(credPath, saltPath)
	if err != nil {
		t.Fatalf("New 1: %v", err)
	}
	store1.Set("openai", "sk-persistent")

	// Create a new store instance pointing to the same files
	store2, err := New(credPath, saltPath)
	if err != nil {
		t.Fatalf("New 2: %v", err)
	}

	key, err := store2.Get("openai")
	if err != nil {
		t.Fatalf("Get from store2: %v", err)
	}
	if key != "sk-persistent" {
		t.Errorf("key = %q, want 'sk-persistent'", key)
	}
}

func TestStoreFilePermissions(t *testing.T) {
	_, credPath := newTestStore(t)

	store, _ := newTestStore(t) // creates a new store, but we check the path
	_ = store
	// Write to the path and check perms
	dir := t.TempDir()
	credPath2 := filepath.Join(dir, "credentials.enc")
	saltPath2 := filepath.Join(dir, ".salt")
	s, err := New(credPath2, saltPath2)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	s.Set("openai", "sk-test")

	info, err := os.Stat(credPath2)
	if err != nil {
		t.Fatalf("stat credentials: %v", err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Errorf("credentials file mode = %o, want 0600", got)
	}
}

func TestStoreGetAddedAt(t *testing.T) {
	store, _ := newTestStore(t)
	store.Set("openai", "sk-test")

	cred, err := store.GetCredential("openai")
	if err != nil {
		t.Fatalf("GetCredential: %v", err)
	}
	if cred.APIKey != "sk-test" {
		t.Errorf("APIKey = %q", cred.APIKey)
	}
	if cred.AddedAt.IsZero() {
		t.Error("AddedAt should not be zero")
	}
	if time.Since(cred.AddedAt) > 5*time.Second {
		t.Error("AddedAt should be recent")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/credential/ -run TestStore -v`
Expected: FAIL with undefined types and functions

- [ ] **Step 3: Write types.go**

```go
// internal/credential/types.go
package credential

import "time"

// ProviderCredential holds a single provider's API key and metadata.
type ProviderCredential struct {
	APIKey  string    `json:"api_key"`
	AddedAt time.Time `json:"added_at"`
}

// credentialData is the JSON structure serialized to the encrypted file.
type credentialData struct {
	Providers map[string]ProviderCredential `json:"providers"`
}
```

- [ ] **Step 4: Write store.go**

```go
// internal/credential/store.go
package credential

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CredentialStore manages encrypted API keys for providers.
type CredentialStore struct {
	credPath string
	saltPath string
	key      []byte
	data     credentialData
	loaded   bool
}

// New creates a CredentialStore backed by the given credential and salt file paths.
// The salt file is created if it does not exist. The credential file is loaded
// if it exists; otherwise an empty store is initialized.
func New(credPath, saltPath string) (*CredentialStore, error) {
	key, err := DeriveKey(saltPath)
	if err != nil {
		return nil, fmt.Errorf("derive key: %w", err)
	}

	s := &CredentialStore{
		credPath: credPath,
		saltPath: saltPath,
		key:      key,
		data: credentialData{
			Providers: make(map[string]ProviderCredential),
		},
	}

	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

// load reads and decrypts the credential file if it exists.
func (s *CredentialStore) load() error {
	s.loaded = true
	ciphertext, err := os.ReadFile(s.credPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // empty store is valid
		}
		return fmt.Errorf("read credentials: %w", err)
	}

	plaintext, err := Decrypt(ciphertext, s.key)
	if err != nil {
		return fmt.Errorf("decrypt credentials: %w", err)
	}

	if err := json.Unmarshal(plaintext, &s.data); err != nil {
		return fmt.Errorf("parse credentials: %w", err)
	}
	if s.data.Providers == nil {
		s.data.Providers = make(map[string]ProviderCredential)
	}
	return nil
}

// save encrypts and writes the credential data to disk.
func (s *CredentialStore) save() error {
	plaintext, err := json.Marshal(s.data)
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}

	ciphertext, err := Encrypt(plaintext, s.key)
	if err != nil {
		return fmt.Errorf("encrypt credentials: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(s.credPath), 0700); err != nil {
		return fmt.Errorf("create credential dir: %w", err)
	}

	if err := os.WriteFile(s.credPath, ciphertext, 0600); err != nil {
		return fmt.Errorf("write credentials: %w", err)
	}
	return nil
}

// Get retrieves the API key for the named provider.
func (s *CredentialStore) Get(provider string) (string, error) {
	cred, err := s.GetCredential(provider)
	if err != nil {
		return "", err
	}
	return cred.APIKey, nil
}

// GetCredential retrieves the full ProviderCredential for the named provider.
func (s *CredentialStore) GetCredential(provider string) (ProviderCredential, error) {
	cred, ok := s.data.Providers[provider]
	if !ok {
		return ProviderCredential{}, fmt.Errorf("provider %q not found in credential store", provider)
	}
	return cred, nil
}

// Set stores the API key for the named provider and persists to disk.
func (s *CredentialStore) Set(provider, apiKey string) error {
	s.data.Providers[provider] = ProviderCredential{
		APIKey:  apiKey,
		AddedAt: time.Now(),
	}
	return s.save()
}

// Remove deletes the named provider from the credential store and persists.
func (s *CredentialStore) Remove(provider string) error {
	if _, ok := s.data.Providers[provider]; !ok {
		return fmt.Errorf("provider %q not found in credential store", provider)
	}
	delete(s.data.Providers, provider)
	return s.save()
}

// List returns the names of all providers with stored credentials.
func (s *CredentialStore) List() []string {
	names := make([]string, 0, len(s.data.Providers))
	for name := range s.data.Providers {
		names = append(names, name)
	}
	return names
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/credential/ -v`
Expected: PASS (all encrypt + store tests)

- [ ] **Step 6: Commit**

```bash
git add internal/credential/store.go internal/credential/store_test.go internal/credential/types.go
git commit -m "credential: add encrypted credential store with Set/Get/Remove/List"
```

---

### Task 3: Multi-Provider Config Format

**Files:**
- Modify: `internal/config/types.go` (add multi-provider types)
- Modify: `internal/config/config.go` (add LoadV2, Save, multi-provider merge logic)
- Create: `internal/config/config_v2_test.go`

**Interfaces:**
- Produces: `MultiProviderConfig` struct with `ActiveProvider string`, `ActiveModel string`, `Providers map[string]ProviderConfig`, `LoadMultiProvider(path string) (*MultiProviderConfig, error)`, `SaveMultiProvider(path string, cfg *MultiProviderConfig) error`

The old `Config`, `FileConfig`, `Merge`, `Load`, `FromEnv` functions remain for backward compatibility during the transition. New code uses `MultiProviderConfig`.

- [ ] **Step 1: Write the failing test**

```go
// internal/config/config_v2_test.go
package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadMultiProviderConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := []byte(`
active_provider: openai
active_model: gpt-image-2
providers:
  openai:
    model: gpt-image-2
    retries: 3
    timeout: 90s
  fal:
    model: fal-ai/flux/dev
    retries: 2
    timeout: 120s
`)
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadMultiProvider(path)
	if err != nil {
		t.Fatalf("LoadMultiProvider: %v", err)
	}
	if cfg.ActiveProvider != "openai" {
		t.Errorf("ActiveProvider = %q, want 'openai'", cfg.ActiveProvider)
	}
	if cfg.ActiveModel != "gpt-image-2" {
		t.Errorf("ActiveModel = %q, want 'gpt-image-2'", cfg.ActiveModel)
	}
	if len(cfg.Providers) != 2 {
		t.Fatalf("Providers len = %d, want 2", len(cfg.Providers))
	}

	openai := cfg.Providers["openai"]
	if openai.Model != "gpt-image-2" {
		t.Errorf("openai model = %q", openai.Model)
	}
	if openai.Retries != 3 {
		t.Errorf("openai retries = %d, want 3", openai.Retries)
	}
	if openai.Timeout != 90*time.Second {
		t.Errorf("openai timeout = %v, want 90s", openai.Timeout)
	}

	fal := cfg.Providers["fal"]
	if fal.Model != "fal-ai/flux/dev" {
		t.Errorf("fal model = %q", fal.Model)
	}
}

func TestLoadMultiProviderMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent.yaml")
	cfg, err := LoadMultiProvider(path)
	if err != nil {
		t.Fatalf("missing file should not error: %v", err)
	}
	if cfg == nil {
		t.Fatal("should return empty config, not nil")
	}
	if cfg.ActiveProvider != "" {
		t.Errorf("ActiveProvider should be empty, got %q", cfg.ActiveProvider)
	}
	if len(cfg.Providers) != 0 {
		t.Errorf("Providers should be empty, got %d", len(cfg.Providers))
	}
}

func TestSaveMultiProviderConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := &MultiProviderConfig{
		ActiveProvider: "openai",
		ActiveModel:     "gpt-image-2",
		Providers: map[string]ProviderConfig{
			"openai": {Model: "gpt-image-2", Retries: 2, Timeout: 120 * time.Second},
		},
	}

	if err := SaveMultiProvider(path, cfg); err != nil {
		t.Fatalf("SaveMultiProvider: %v", err)
	}

	// Read it back
	loaded, err := LoadMultiProvider(path)
	if err != nil {
		t.Fatalf("LoadMultiProvider after save: %v", err)
	}
	if loaded.ActiveProvider != "openai" {
		t.Errorf("ActiveProvider = %q", loaded.ActiveProvider)
	}
	if loaded.Providers["openai"].Model != "gpt-image-2" {
		t.Errorf("model = %q", loaded.Providers["openai"].Model)
	}

	// Verify file permissions
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Errorf("file mode = %o, want 0600", got)
	}
}

func TestSaveMultiProviderPreservesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg1 := &MultiProviderConfig{
		ActiveProvider: "openai",
		ActiveModel:     "gpt-image-2",
		Providers: map[string]ProviderConfig{
			"openai": {Model: "gpt-image-2", Retries: 2, Timeout: 120 * time.Second},
		},
	}
	SaveMultiProvider(path, cfg1)

	// Load, add another provider, save
	cfg2, err := LoadMultiProvider(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	cfg2.Providers["fal"] = ProviderConfig{Model: "fal-ai/flux/dev", Retries: 2, Timeout: 120 * time.Second}
	cfg2.ActiveProvider = "fal"
	cfg2.ActiveModel = "fal-ai/flux/dev"
	SaveMultiProvider(path, cfg2)

	// Read back and verify both providers
	loaded, err := LoadMultiProvider(path)
	if err != nil {
		t.Fatalf("Load after second save: %v", err)
	}
	if len(loaded.Providers) != 2 {
		t.Fatalf("Providers len = %d, want 2", len(loaded.Providers))
	}
	if loaded.ActiveProvider != "fal" {
		t.Errorf("ActiveProvider = %q, want 'fal'", loaded.ActiveProvider)
	}
	if _, ok := loaded.Providers["openai"]; !ok {
		t.Error("openai should still be present")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/ -run "TestLoadMultiProvider|TestSaveMultiProvider" -v`
Expected: FAIL with undefined types and functions

- [ ] **Step 3: Write minimal implementation**

Add to `internal/config/types.go`:

```go
// ProviderConfig holds per-provider settings in the multi-provider config format.
type ProviderConfig struct {
	Model   string        `yaml:"model"`
	Retries int           `yaml:"retries"`
	Timeout time.Duration `yaml:"timeout"`
}

// MultiProviderConfig is the v2 config format supporting multiple providers
// with separate credentials and an active provider/model selector.
type MultiProviderConfig struct {
	ActiveProvider string                    `yaml:"active_provider"`
	ActiveModel    string                    `yaml:"active_model"`
	Providers      map[string]ProviderConfig `yaml:"providers"`
}
```

Add to `internal/config/config.go` (new functions, keeping existing ones):

```go
// LoadMultiProvider reads and parses a multi-provider config file.
// Returns an empty MultiProviderConfig (not nil, not an error) if the
// file does not exist.
func LoadMultiProvider(path string) (*MultiProviderConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &MultiProviderConfig{
				Providers: make(map[string]ProviderConfig),
			}, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg MultiProviderConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg.Providers == nil {
		cfg.Providers = make(map[string]ProviderConfig)
	}
	return &cfg, nil
}

// SaveMultiProvider writes the multi-provider config to the given path.
// The file is created with 0600 permissions and the directory with 0700.
func SaveMultiProvider(path string, cfg *MultiProviderConfig) error {
	if cfg.Providers == nil {
		cfg.Providers = make(map[string]ProviderConfig)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".config-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp config: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		return fmt.Errorf("set temp config perms: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp config: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace config: %w", err)
	}
	cleanup = false
	return nil
}

// DefaultCredentialPath returns the default credential file path.
func DefaultCredentialPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".potaco/credentials.enc"
	}
	return filepath.Join(home, ".potaco", "credentials.enc")
}

// DefaultSaltPath returns the default salt file path.
func DefaultSaltPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".potaco/.salt"
	}
	return filepath.Join(home, ".potaco", ".salt")
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/config/ -run "TestLoadMultiProvider|TestSaveMultiProvider" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/config/types.go internal/config/config.go internal/config/config_v2_test.go
git commit -m "config: add multi-provider config format with Load/Save and ProviderConfig type"
```

---

### Task 4: Auth Package (Credential Management Logic)

**Files:**
- Create: `internal/auth/auth.go`
- Create: `internal/auth/auth_test.go`

**Interfaces:**
- Consumes: `credential.CredentialStore` (from Task 2), `config.MultiProviderConfig`, `config.LoadMultiProvider`, `config.SaveMultiProvider`, `config.DefaultConfigPath`, `config.DefaultCredentialPath`, `config.DefaultSaltPath` (from Task 3), `adapter.Get`, `adapter.List` (from Phase 1)
- Produces: `AuthManager` struct with `New() (*AuthManager, error)`, `Add(provider, apiKey string, force bool) error`, `Remove(provider string) error`, `List() []ProviderInfo`, `GetActiveProvider() (string, error)`, `SetActiveProvider(provider, model string) error`, `GetActiveCredential() (adapter.Adapter, error)`

- [ ] **Step 1: Write the failing test**

```go
// internal/auth/auth_test.go
package auth

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ncxton/potaco/internal/credential"
)

func newTestAuth(t *testing.T) *AuthManager {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	// We need to override the config and credential paths for testing.
	// AuthManager uses DefaultConfigPath/DefaultCredentialPath/DefaultSaltPath
	// which read from HOME. By setting HOME to temp dir, all paths resolve there.
	return newAuthWithPaths(
		filepath.Join(dir, ".potaco", "config.yaml"),
		filepath.Join(dir, ".potaco", "credentials.enc"),
		filepath.Join(dir, ".potaco", ".salt"),
	)
}

func newAuthWithPaths(configPath, credPath, saltPath string) *AuthManager {
	store, err := credential.New(credPath, saltPath)
	if err != nil {
		panic(err)
	}
	return &AuthManager{
		store:      store,
		configPath: configPath,
	}
}

func TestAuthAdd(t *testing.T) {
	auth := newTestAuth(t)

	err := auth.Add("openai", "sk-test-key", true)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	// Verify key is stored
	key, err := auth.store.Get("openai")
	if err != nil {
		t.Fatalf("Get key: %v", err)
	}
	if key != "sk-test-key" {
		t.Errorf("key = %q, want 'sk-test-key'", key)
	}

	// Verify config entry is written
	cfg, err := auth.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.ActiveProvider != "openai" {
		t.Errorf("ActiveProvider = %q, want 'openai'", cfg.ActiveProvider)
	}
	if _, ok := cfg.Providers["openai"]; !ok {
		t.Error("openai should be in config providers")
	}
}

func TestAuthAddSetsActiveProvider(t *testing.T) {
	auth := newTestAuth(t)

	auth.Add("openai", "sk-1", true)
	auth.Add("fal", "fal-1", true)

	cfg, _ := auth.LoadConfig()
	// Adding a second provider should switch active to the new one
	if cfg.ActiveProvider != "fal" {
		t.Errorf("ActiveProvider = %q, want 'fal'", cfg.ActiveProvider)
	}
}

func TestAuthRemove(t *testing.T) {
	auth := newTestAuth(t)
	auth.Add("openai", "sk-1", true)
	auth.Add("fal", "fal-2", true)

	err := auth.Remove("fal")
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}

	// Key should be gone
	_, err = auth.store.Get("fal")
	if err == nil {
		t.Error("fal key should be removed")
	}

	// Config entry should be gone
	cfg, _ := auth.LoadConfig()
	if _, ok := cfg.Providers["fal"]; ok {
		t.Error("fal should be removed from config")
	}

	// Active should switch back to openai
	if cfg.ActiveProvider != "openai" {
		t.Errorf("ActiveProvider = %q, want 'openai'", cfg.ActiveProvider)
	}
}

func TestAuthRemoveActiveProvider(t *testing.T) {
	auth := newTestAuth(t)
	auth.Add("openai", "sk-1", true)

	err := auth.Remove("openai")
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}

	cfg, _ := auth.LoadConfig()
	if cfg.ActiveProvider != "" {
		t.Errorf("ActiveProvider = %q, want empty", cfg.ActiveProvider)
	}
}

func TestAuthList(t *testing.T) {
	auth := newTestAuth(t)
	auth.Add("openai", "sk-1", true)
	auth.Add("fal", "sk-2", true)

	providers := auth.List()
	if len(providers) != 2 {
		t.Fatalf("List len = %d, want 2", len(providers))
	}

	found := map[string]bool{}
	for _, p := range providers {
		found[p.Name] = true
		if p.Name == "openai" {
			if !p.HasKey {
				t.Error("openai should have key")
			}
		}
	}
	if !found["openai"] || !found["fal"] {
		t.Error("missing providers in list")
	}
}

func TestAuthSetActiveProvider(t *testing.T) {
	auth := newTestAuth(t)
	auth.Add("openai", "sk-1", true)
	auth.Add("fal", "sk-2", true)

	err := auth.SetActiveProvider("fal", "fal-ai/flux/schnell")
	if err != nil {
		t.Fatalf("SetActiveProvider: %v", err)
	}

	cfg, _ := auth.LoadConfig()
	if cfg.ActiveProvider != "fal" {
		t.Errorf("ActiveProvider = %q, want 'fal'", cfg.ActiveProvider)
	}
	if cfg.ActiveModel != "fal-ai/flux/schnell" {
		t.Errorf("ActiveModel = %q", cfg.ActiveModel)
	}
	if cfg.Providers["fal"].Model != "fal-ai/flux/schnell" {
		t.Errorf("fal model in config = %q", cfg.Providers["fal"].Model)
	}
}

func TestAuthGetActiveCredential(t *testing.T) {
	auth := newTestAuth(t)
	auth.Add("openai", "sk-test", true)

	// This should return an adapter for the active provider
	_, err := auth.GetActiveAPIKey()
	if err != nil {
		t.Fatalf("GetActiveAPIKey: %v", err)
	}
}

func TestAuthGetActiveAPIKeyNoProvider(t *testing.T) {
	auth := newTestAuth(t)

	_, err := auth.GetActiveAPIKey()
	if err == nil {
		t.Fatal("should error when no active provider")
	}
	if !os.IsNotExist(err) && err.Error() == "" {
		// It should be a clear error about no active provider
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/auth/ -v`
Expected: FAIL with "no such package"

- [ ] **Step 3: Write minimal implementation**

```go
// internal/auth/auth.go
package auth

import (
	"fmt"
	"time"

	"github.com/ncxton/potaco/internal/config"
	"github.com/ncxton/potaco/internal/credential"
)

// ProviderInfo holds display information about a connected provider.
type ProviderInfo struct {
	Name     string
	Model    string
	HasKey   bool
	AddedAt  string // formatted date
	IsActive bool
}

// AuthManager coordinates credential storage and multi-provider config.
type AuthManager struct {
	store      *credential.CredentialStore
	configPath string
}

// New creates an AuthManager using default paths for config and credentials.
func New() (*AuthManager, error) {
	configPath := config.DefaultConfigPath()
	credPath := config.DefaultCredentialPath()
	saltPath := config.DefaultSaltPath()

	store, err := credential.New(credPath, saltPath)
	if err != nil {
		return nil, fmt.Errorf("create credential store: %w", err)
	}

	return &AuthManager{
		store:      store,
		configPath: configPath,
	}, nil
}

// NewWithStore creates an AuthManager with an explicit credential store
// and config path. Used for testing.
func NewWithStore(store *credential.CredentialStore, configPath string) *AuthManager {
	return &AuthManager{
		store:      store,
		configPath: configPath,
	}
}

// LoadConfig reads the multi-provider config file.
func (m *AuthManager) LoadConfig() (*config.MultiProviderConfig, error) {
	return config.LoadMultiProvider(m.configPath)
}

// saveConfig writes the multi-provider config file.
func (m *AuthManager) saveConfig(cfg *config.MultiProviderConfig) error {
	return config.SaveMultiProvider(m.configPath, cfg)
}

// Add stores the API key for a provider, creates a config entry, and
// sets it as the active provider. The force parameter skips verification
// (verification itself is handled by the CLI command, not the auth layer).
func (m *AuthManager) Add(provider, apiKey string, force bool) error {
	if err := m.store.Set(provider, apiKey); err != nil {
		return fmt.Errorf("store credential: %w", err)
	}

	cfg, err := m.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if cfg.Providers == nil {
		cfg.Providers = make(map[string]config.ProviderConfig)
	}

	// Set default model for the provider if not already configured
	if _, exists := cfg.Providers[provider]; !exists {
		cfg.Providers[provider] = config.ProviderConfig{
			Model:   defaultModelForProvider(provider),
			Retries: 2,
			Timeout: 120 * time.Second,
		}
	}

	// Set as active provider
	cfg.ActiveProvider = provider
	cfg.ActiveModel = cfg.Providers[provider].Model

	return m.saveConfig(cfg)
}

// Remove deletes the provider's credential and config entry.
// If the removed provider was active, switches to another available provider
// or clears the active field if none remain.
func (m *AuthManager) Remove(provider string) error {
	if err := m.store.Remove(provider); err != nil {
		return fmt.Errorf("remove credential: %w", err)
	}

	cfg, err := m.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	delete(cfg.Providers, provider)

	// If we just removed the active provider, pick a replacement
	if cfg.ActiveProvider == provider {
		cfg.ActiveProvider = ""
		cfg.ActiveModel = ""
		for name := range cfg.Providers {
			cfg.ActiveProvider = name
			cfg.ActiveModel = cfg.Providers[name].Model
			break
		}
	}

	return m.saveConfig(cfg)
}

// List returns information about all configured providers.
func (m *AuthManager) List() []ProviderInfo {
	cfg, _ := m.LoadConfig()
	if cfg == nil {
		return nil
	}

	stored := m.store.List()
	infos := make([]ProviderInfo, 0, len(cfg.Providers))

	for name, pc := range cfg.Providers {
		info := ProviderInfo{
			Name:     name,
			Model:    pc.Model,
			IsActive: cfg.ActiveProvider == name,
		}
		// Check if key exists in credential store
		for _, s := range stored {
			if s == name {
				info.HasKey = true
				break
			}
		}
		infos = append(infos, info)
	}

	return infos
}

// SetActiveProvider changes the active provider and optionally updates the model.
// If model is empty, keeps the existing model from config.
func (m *AuthManager) SetActiveProvider(provider, model string) error {
	cfg, err := m.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if _, ok := cfg.Providers[provider]; !ok {
		return fmt.Errorf("provider %q is not configured. Use 'potaco auth add %s' first", provider, provider)
	}

	cfg.ActiveProvider = provider
	if model != "" {
		cfg.ActiveModel = model
		pc := cfg.Providers[provider]
		pc.Model = model
		cfg.Providers[provider] = pc
	} else {
		cfg.ActiveModel = cfg.Providers[provider].Model
	}

	return m.saveConfig(cfg)
}

// GetActiveAPIKey returns the API key for the active provider.
func (m *AuthManager) GetActiveAPIKey() (string, error) {
	cfg, err := m.LoadConfig()
	if err != nil {
		return "", fmt.Errorf("load config: %w", err)
	}
	if cfg.ActiveProvider == "" {
		return "", fmt.Errorf("no active provider configured. Use 'potaco auth add <provider>' to connect one")
	}
	return m.store.Get(cfg.ActiveProvider)
}

// GetActiveProvider returns the active provider name and model from config.
func (m *AuthManager) GetActiveProvider() (provider, model string, err error) {
	cfg, err := m.LoadConfig()
	if err != nil {
		return "", "", fmt.Errorf("load config: %w", err)
	}
	return cfg.ActiveProvider, cfg.ActiveModel, nil
}

// defaultModelForProvider returns a sensible default model for a provider name.
func defaultModelForProvider(provider string) string {
	defaults := map[string]string{
		"openai": "gpt-image-2",
		"fal":    "fal-ai/flux/dev",
		"vercel": "openai/gpt-image-2",
	}
	if m, ok := defaults[provider]; ok {
		return m
	}
	return ""
}
```

Note: The `time` import is needed for the `120 * time.Second` default. Also `config.ProviderConfig` is the type from Task 3.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/auth/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/auth/auth.go internal/auth/auth_test.go
git commit -m "auth: add AuthManager for credential and config lifecycle management"
```

---

### Task 5: `auth add` CLI Command (Non-Interactive)

**Files:**
- Create: `internal/cli/auth_cmd.go`
- Create: `internal/cli/auth_cmd_test.go`
- Modify: `internal/cli/root.go` (register auth command group)

**Interfaces:**
- Consumes: `auth.New()`, `AuthManager.Add(provider, apiKey, force)` (from Task 4), `adapter.Get(provider, apiKey, opts)`, `adapter.Adapter.Verify(ctx)` (from Phase 1)
- Produces: `authCmd` (auth command group), `authAddCmd` (`potaco auth add <provider> [--api-key <key>] [--force]`)

- [ ] **Step 1: Write the failing test**

```go
// internal/cli/auth_cmd_test.go
package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newAuthTest(t *testing.T) (string, *bytes.Buffer) {
	t.Helper()
	resetRootCmdFlags(t)
	resetAuthAddFlags(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	return tmpHome, &buf
}

func resetAuthAddFlags(t *testing.T) {
	t.Helper()
	for _, name := range []string{"api-key", "force", "model"} {
		flag := authAddCmd.Flags().Lookup(name)
		if flag == nil {
			return // flags not registered yet
		}
		_ = flag.Value.Set(flag.DefValue)
		flag.Changed = false
	}
}

func TestAuthCommandExists(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "auth" || strings.HasPrefix(cmd.Use, "auth ") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("root command should have 'auth' subcommand")
	}
}

func TestAuthAddCommandExists(t *testing.T) {
	found := false
	for _, cmd := range authCmd.Commands() {
		if cmd.Use == "add" || strings.HasPrefix(cmd.Use, "add ") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("auth command should have 'add' subcommand")
	}
}

func TestAuthAddNonInteractive(t *testing.T) {
	_, buf := newAuthTest(t)
	rootCmd.SetArgs([]string{"auth", "add", "openai", "--api-key", "sk-test-key"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("auth add error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "openai") {
		t.Errorf("output should mention openai, got: %q", output)
	}

	// Verify credential was stored
	credPath := filepath.Join(t.TempDir(), "nope") // wrong - we set HOME above
	// Instead check via the auth manager
	// The config file should exist at ~/.potaco/config.yaml (HOME = tmpHome)
}

func TestAuthAddRequiresAPIKey(t *testing.T) {
	_, buf := newAuthTest(t)
	rootCmd.SetArgs([]string{"auth", "add", "openai"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("auth add without --api-key should error in non-interactive mode")
	}
	if !strings.Contains(err.Error(), "api-key") && !strings.Contains(err.Error(), "API key") {
		t.Errorf("error should mention api-key, got: %v", err)
	}

	_ = buf // output may contain error message
}

func TestAuthAddUnknownProvider(t *testing.T) {
	_, _ := newAuthTest(t)
	rootCmd.SetArgs([]string{"auth", "add", "nonexistent", "--api-key", "sk-test"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("auth add with unknown provider should error")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cli/ -run TestAuth -v`
Expected: FAIL with undefined `authCmd`, `authAddCmd`

- [ ] **Step 3: Write minimal implementation**

```go
// internal/cli/auth_cmd.go
package cli

import (
	"context"
	"fmt"

	"github.com/ncxton/potaco/internal/adapter"
	"github.com/ncxton/potaco/internal/auth"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage provider credentials",
}

var authAddCmd = &cobra.Command{
	Use:   "add <provider>",
	Short: "Connect to a provider by storing its API key",
	Args:  cobra.ExactArgs(1),
	RunE:  runAuthAdd,
}

func init() {
	authAddCmd.Flags().String("api-key", "", "API key for the provider")
	authAddCmd.Flags().Bool("force", false, "skip provider verification")
	authAddCmd.Flags().String("model", "", "override the default model for this provider")

	authCmd.AddCommand(authAddCmd)
	rootCmd.AddCommand(authCmd)
}

func runAuthAdd(cmd *cobra.Command, args []string) error {
	providerName := args[0]

	// Check provider is a known adapter
	available := adapter.List()
	known := false
	for _, name := range available {
		if name == providerName {
			known = true
			break
		}
	}
	if !known {
		return configError(fmt.Errorf("unknown provider: %s (available: %v)", providerName, available))
	}

	// Get API key from flag or env
	apiKey, _ := cmd.Flags().GetString("api-key")
	if apiKey == "" {
		apiKey = envOrEmpty("POTACO_API_KEY")
	}
	if apiKey == "" {
		return configError(fmt.Errorf("API key required: use --api-key or set POTACO_API_KEY"))
	}

	// Create auth manager
	mgr, err := auth.New()
	if err != nil {
		return configError(fmt.Errorf("init auth: %w", err))
	}

	// Verify provider unless --force
	force, _ := cmd.Flags().GetBool("force")
	if !force {
		ad, err := adapter.Get(providerName, apiKey, adapter.AdapterOpts{})
		if err != nil {
			return configError(fmt.Errorf("create adapter: %w", err))
		}
		if err := ad.Verify(context.Background()); err != nil {
			return apiError(fmt.Errorf("verification failed: %w\nUse --force to add anyway", err))
		}
	}

	// Add the provider
	if err := mgr.Add(providerName, apiKey, force); err != nil {
		return configError(fmt.Errorf("add provider: %w", err))
	}

	// Override model if specified
	model, _ := cmd.Flags().GetString("model")
	if model != "" {
		if err := mgr.SetActiveProvider(providerName, model); err != nil {
			return configError(fmt.Errorf("set model: %w", err))
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Provider '%s' added successfully.\n", providerName)
	fmt.Fprintf(cmd.OutOrStdout(), "Use 'potaco use %s' to switch to it.\n", providerName)
	return nil
}

func envOrEmpty(key string) string {
	return getEnv(key)
}
```

Note: `getEnv` is a helper that reads from `os.Getenv`. If it doesn't exist yet, use `os.Getenv` directly. Add `os` to the imports if needed. Check if `os` is already imported in the file.

Actually, simplify: just use `os.Getenv` directly and add the `os` import:

```go
// At the top of auth_cmd.go, add "os" to imports
// Replace the envOrEmpty helper with direct os.Getenv call:
apiKey = os.Getenv("POTACO_API_KEY")
```

Remove the `envOrEmpty` function and `getEnv` reference. Use `os.Getenv("POTACO_API_KEY")` directly.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/cli/ -run TestAuth -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/cli/auth_cmd.go internal/cli/auth_cmd_test.go internal/cli/root.go
git commit -m "cli: add auth add command (non-interactive, with verification)"
```

---

### Task 6: `auth remove` and `auth list` CLI Commands

**Files:**
- Modify: `internal/cli/auth_cmd.go` (add remove and list commands)
- Modify: `internal/cli/auth_cmd_test.go` (add tests)

**Interfaces:**
- Consumes: `AuthManager.Remove(provider)`, `AuthManager.List()` (from Task 4)
- Produces: `authRemoveCmd` (`potaco auth remove <provider>` / `auth rm <provider>`), `authListCmd` (`potaco auth list` / `auth ls`)

- [ ] **Step 1: Write the failing test**

```go
// Add to internal/cli/auth_cmd_test.go

func TestAuthRemoveCommand(t *testing.T) {
	_, buf := newAuthTest(t)

	// First add a provider
	rootCmd.SetArgs([]string{"auth", "add", "openai", "--api-key", "sk-test"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("setup auth add: %v", err)
	}
	resetAuthAddFlags(t)
	resetRootCmdFlags(t)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	// Now remove it
	rootCmd.SetArgs([]string{"auth", "remove", "openai"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("auth remove error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "removed") || !strings.Contains(output, "openai") {
		t.Errorf("output should mention removal of openai, got: %q", output)
	}
}

func TestAuthRemoveAliasRm(t *testing.T) {
	_, buf := newAuthTest(t)

	rootCmd.SetArgs([]string{"auth", "add", "openai", "--api-key", "sk-test"})
	rootCmd.Execute()
	resetAuthAddFlags(t)
	resetRootCmdFlags(t)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"auth", "rm", "openai"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("auth rm error: %v", err)
	}
}

func TestAuthListCommand(t *testing.T) {
	_, buf := newAuthTest(t)

	// Add two providers
	rootCmd.SetArgs([]string{"auth", "add", "openai", "--api-key", "sk-1", "--force"})
	rootCmd.Execute()
	resetAuthAddFlags(t)
	resetRootCmdFlags(t)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"auth", "add", "fal", "--api-key", "fal-1", "--force"})
	rootCmd.Execute()
	resetAuthAddFlags(t)
	resetRootCmdFlags(t)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	// List
	rootCmd.SetArgs([]string{"auth", "list"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("auth list error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "openai") {
		t.Errorf("list should include openai, got: %q", output)
	}
	if !strings.Contains(output, "fal") {
		t.Errorf("list should include fal, got: %q", output)
	}
}

func TestAuthListAliasLs(t *testing.T) {
	_, buf := newAuthTest(t)

	rootCmd.SetArgs([]string{"auth", "add", "openai", "--api-key", "sk-1", "--force"})
	rootCmd.Execute()
	resetAuthAddFlags(t)
	resetRootCmdFlags(t)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"auth", "ls"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("auth ls error: %v", err)
	}
}

func TestAuthListJSON(t *testing.T) {
	_, buf := newAuthTest(t)

	rootCmd.SetArgs([]string{"auth", "add", "openai", "--api-key", "sk-1", "--force"})
	rootCmd.Execute()
	resetAuthAddFlags(t)
	resetRootCmdFlags(t)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	if err := rootCmd.PersistentFlags().Set("json", "true"); err != nil {
		t.Fatalf("set json flag: %v", err)
	}
	rootCmd.SetArgs([]string{"auth", "list"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("auth list --json error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "[") {
		t.Errorf("JSON output should be an array, got: %q", output)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cli/ -run "TestAuthRemove|TestAuthList" -v`
Expected: FAIL with undefined commands

- [ ] **Step 3: Write minimal implementation**

Add to `internal/cli/auth_cmd.go`:

```go
var authRemoveCmd = &cobra.Command{
	Use:   "remove <provider>",
	Short: "Remove a provider's credentials and config",
	Args:  cobra.ExactArgs(1),
	RunE:  runAuthRemove,
	Aliases: []string{"rm"},
}

var authListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all connected providers",
	RunE:    runAuthList,
	Aliases: []string{"ls"},
}

// Update the init() function to also add these:
func init() {
	authAddCmd.Flags().String("api-key", "", "API key for the provider")
	authAddCmd.Flags().Bool("force", false, "skip provider verification")
	authAddCmd.Flags().String("model", "", "override the default model for this provider")

	authCmd.AddCommand(authAddCmd)
	authCmd.AddCommand(authRemoveCmd)
	authCmd.AddCommand(authListCmd)
	rootCmd.AddCommand(authCmd)
}

func runAuthRemove(cmd *cobra.Command, args []string) error {
	providerName := args[0]

	mgr, err := auth.New()
	if err != nil {
		return configError(fmt.Errorf("init auth: %w", err))
	}

	if err := mgr.Remove(providerName); err != nil {
		return configError(fmt.Errorf("remove provider: %w", err))
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Provider '%s' removed.\n", providerName)
	return nil
}

func runAuthList(cmd *cobra.Command, args []string) error {
	mgr, err := auth.New()
	if err != nil {
		return configError(fmt.Errorf("init auth: %w", err))
	}

	providers := mgr.List()
	jsonMode, _ := cmd.Root().PersistentFlags().GetBool("json")
	out := cmd.OutOrStdout()

	if jsonMode {
		type providerJSON struct {
			Name     string `json:"name"`
			Model    string `json:"model"`
			HasKey   bool   `json:"has_key"`
			IsActive bool   `json:"is_active"`
		}
		items := make([]providerJSON, 0, len(providers))
		for _, p := range providers {
			items = append(items, providerJSON{
				Name:     p.Name,
				Model:    p.Model,
				HasKey:   p.HasKey,
				IsActive: p.IsActive,
			})
		}
		data, err := json.MarshalIndent(items, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal JSON: %w", err)
		}
		fmt.Fprintln(out, string(data))
		return nil
	}

	if len(providers) == 0 {
		fmt.Fprintln(out, "No providers connected. Use 'potaco auth add <provider>' to connect one.")
		return nil
	}

	fmt.Fprintln(out, "Connected providers:")
	fmt.Fprintln(out)
	for _, p := range providers {
		active := ""
		if p.IsActive {
			active = " (active)"
		}
		keyStatus := "missing"
		if p.HasKey {
			keyStatus = "configured"
		}
		fmt.Fprintf(out, "  %s\t%s\tkey: %s%s\n", p.Name, p.Model, keyStatus, active)
	}
	return nil
}
```

Note: Add `"encoding/json"` to the imports if not already present.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/cli/ -run TestAuth -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/cli/auth_cmd.go internal/cli/auth_cmd_test.go
git commit -m "cli: add auth remove/rm and auth list/ls commands"
```

---

### Task 7: Wire `gen`/`edit` to Read from Credential Store

**Files:**
- Modify: `internal/cli/gen.go` (resolve provider from auth manager)
- Modify: `internal/cli/edit.go` (resolve provider from auth manager)
- Modify: `internal/cli/helpers.go` (update `adapterForProvider` to use auth manager)
- Modify: `internal/cli/gen_test.go` (update tests for new resolution path)
- Modify: `internal/cli/edit_test.go` (update tests)

**Interfaces:**
- Consumes: `auth.New()`, `AuthManager.GetActiveAPIKey()`, `AuthManager.GetActiveProvider()`, `AuthManager.LoadConfig()` (from Tasks 4-6)
- Produces: `gen` and `edit` commands that read active provider + key from auth manager, with `--api-key`/`--base-url`/`--model`/`POTACO_API_KEY`/`POTACO_BASE_URL`/`POTACO_MODEL`/`POTACO_PROVIDER` overrides

The credential access flow when `gen` or `edit` runs:
1. Read `active_provider` from config (or override with `POTACO_PROVIDER` env or `--provider` flag)
2. Get API key from credential store (or override with `POTACO_API_KEY` env / `--api-key` flag)
3. Build adapter: `adapter.Get(providerName, apiKey, opts)`
4. Read `active_model` from config (or override with `--model` flag)

- [ ] **Step 1: Write the failing test**

```go
// Add to internal/cli/gen_test.go

func TestGenWithAuthCredentials(t *testing.T) {
	resetRootCmdFlags(t)
	resetAuthAddFlags(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)

	// Add a provider via auth add (with --force to skip verification)
	rootCmd.SetArgs([]string{"auth", "add", "openai", "--api-key", "sk-from-auth", "--force"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("setup auth add: %v", err)
	}
	resetAuthAddFlags(t)
	resetRootCmdFlags(t)
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)

	// Now run gen --dry-run, it should use the stored credential
	rootCmd.SetArgs([]string{"gen", "--prompt", "a cat", "--dry-run"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("gen --dry-run: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "/v1/images/generations") {
		t.Errorf("dry-run should contain endpoint, got: %q", output)
	}
	if !strings.Contains(output, "gpt-image-2") {
		t.Errorf("dry-run should contain default model, got: %q", output)
	}
}

func TestGenNoActiveProviderError(t *testing.T) {
	resetRootCmdFlags(t)
	resetAuthAddFlags(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	// Clear env overrides
	t.Setenv("POTACO_API_KEY", "")
	t.Setenv("POTACO_BASE_URL", "")
	t.Setenv("POTACO_PROVIDER", "")

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"gen", "--prompt", "test", "--dry-run"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("gen should error when no provider is configured")
	}
	if !strings.Contains(err.Error(), "provider") {
		t.Errorf("error should mention provider, got: %v", err)
	}
}

func TestGenWithApiKeyOverride(t *testing.T) {
	resetRootCmdFlags(t)
	resetAuthAddFlags(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)

	// Add a provider
	rootCmd.SetArgs([]string{"auth", "add", "openai", "--api-key", "sk-stored", "--force"})
	rootCmd.Execute()
	resetAuthAddFlags(t)
	resetRootCmdFlags(t)
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)

	// Run gen with --api-key override and --base-url override
	rootCmd.SetArgs([]string{"gen", "--prompt", "test", "--dry-run", "--api-key", "sk-override", "--base-url", "https://custom.api.com"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("gen with override: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "custom.api.com") {
		t.Errorf("dry-run should use overridden base-url, got: %q", output)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cli/ -run "TestGenWithAuth|TestGenNoActive" -v`
Expected: FAIL (gen still uses old config.Merge path)

- [ ] **Step 3: Update helpers.go to add new resolution function**

Add a new function `resolveAdapter` that uses the auth manager:

```go
// Add to internal/cli/helpers.go

// resolvedConfig holds the adapter and model resolved from auth/config/overrides.
type resolvedConfig struct {
	Adapter adapter.Adapter
	Model   string
}

// resolveAdapterForCommand resolves the active provider, credential, and model
// from the auth manager, applying CLI flag and env var overrides.
func resolveAdapterForCommand(cmd *cobra.Command) (*resolvedConfig, error) {
	mgr, err := auth.New()
	if err != nil {
		return nil, configError(fmt.Errorf("init auth: %w", err))
	}

	// Determine provider: --provider flag > POTACO_PROVIDER env > active_provider from config
	providerName := ""
	if cmd.Flags().Changed("provider") {
		providerName = flagString(cmd, "provider")
	} else if v := os.Getenv("POTACO_PROVIDER"); v != "" {
		providerName = v
	} else {
		p, _, err := mgr.GetActiveProvider()
		if err != nil {
			return nil, configError(fmt.Errorf("no active provider: %w", err))
		}
		providerName = p
	}
	if providerName == "" {
		return nil, configError(fmt.Errorf("no active provider configured. Use 'potaco auth add <provider>' to connect one"))
	}

	// Determine API key: --api-key flag > POTACO_API_KEY env > credential store
	apiKey := ""
	if cmd.Flags().Changed("api-key") {
		apiKey = flagString(cmd, "api-key")
	} else if v := os.Getenv("POTACO_API_KEY"); v != "" {
		apiKey = v
	} else {
		k, err := mgr.GetActiveAPIKey()
		if err != nil {
			return nil, configError(fmt.Errorf("get API key: %w", err))
		}
		apiKey = k
	}

	// Determine model: --model flag > POTACO_MODEL env > active_model from config
	model := ""
	if cmd.Flags().Changed("model") {
		model = flagString(cmd, "model")
	} else if v := os.Getenv("POTACO_MODEL"); v != "" {
		model = v
	} else {
		_, m, _ := mgr.GetActiveProvider()
		model = m
	}

	// Determine base URL override
	baseURL := ""
	if cmd.Flags().Changed("base-url") {
		baseURL = flagString(cmd, "base-url")
	} else if v := os.Getenv("POTACO_BASE_URL"); v != "" {
		baseURL = v
	}

	// Determine retries and timeout from config
	cfg, _ := mgr.LoadConfig()
	retries := 2
	timeout := 120 * time.Second
	if pc, ok := cfg.Providers[providerName]; ok {
		if pc.Retries > 0 {
			retries = pc.Retries
		}
		if pc.Timeout > 0 {
			timeout = pc.Timeout
		}
	}

	// Env overrides for retries/timeout
	if v := os.Getenv("POTACO_RETRIES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			retries = n
		}
	}
	if v := os.Getenv("POTACO_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			timeout = d
		}
	}
	if cmd.Flags().Changed("retries") {
		retries = flagInt(cmd, "retries")
	}
	if cmd.Flags().Changed("timeout") {
		timeout, _ = cmd.Flags().GetDuration("timeout")
	}

	opts := adapter.AdapterOpts{
		BaseURL: baseURL,
		Retries: retries,
	}
	if timeout > 0 {
		opts.Timeout = timeout.String()
	}

	ad, err := adapter.Get(providerName, apiKey, opts)
	if err != nil {
		return nil, configError(fmt.Errorf("create adapter: %w", err))
	}

	return &resolvedConfig{
		Adapter: ad,
		Model:   model,
	}, nil
}
```

Note: This function needs imports for `os`, `strconv`, `time`, `auth`, `adapter`, `config`. Check what's already imported in helpers.go and add missing ones.

- [ ] **Step 4: Update gen.go to use resolveAdapterForCommand**

In `gen.go`, replace the `buildMergeOptions` + `config.Merge` + `adapterForProvider` chain with `resolveAdapterForCommand`:

Replace the config resolution block in `runGen`:
```go
// Old:
opts := buildMergeOptions(cmd)
cfg, err := config.Merge(opts)
if err != nil {
    return configError(fmt.Errorf("config: %w", err))
}
model := cfg.Model
if cmd.Flags().Changed("model") {
    model = flagString(cmd, "model")
}

// New:
resolved, err := resolveAdapterForCommand(cmd)
if err != nil {
    return err
}
model := resolved.Model
```

And replace the adapter creation:
```go
// Old:
client := provider.NewClient(...) / adapterForProvider(cfg)
resp, err := ad.Generate(context.Background(), req)

// New:
resp, err := resolved.Adapter.Generate(context.Background(), req)
```

Remove the `config` import from `gen.go` if it's no longer used. Keep `adapter` import for the request types.

- [ ] **Step 5: Update edit.go similarly**

Apply the same pattern to `edit.go`: replace `buildMergeOptions` + `config.Merge` + `adapterForProvider` with `resolveAdapterForCommand`.

- [ ] **Step 6: Update existing tests**

The existing gen/edit tests that use `t.Setenv("POTACO_BASE_URL", ...)` and `t.Setenv("POTACO_API_KEY", ...)` will still work because `resolveAdapterForCommand` checks env vars. But tests that relied on the old `config.Merge` error messages may need updating.

Update `TestGenCommandDryRunNoAPI`:
```go
func TestGenCommandDryRunNoAPI(t *testing.T) {
	resetRootCmdFlags(t)
	resetAuthAddFlags(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Add a provider first
	rootCmd.SetArgs([]string{"auth", "add", "openai", "--api-key", "sk-test", "--force"})
	rootCmd.Execute()
	resetAuthAddFlags(t)
	resetRootCmdFlags(t)

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"gen", "--prompt", "a cat", "--dry-run"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("gen --dry-run returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"method": "POST"`) {
		t.Errorf("dry-run should print request method, got: %q", output)
	}
	if !strings.Contains(output, "/v1/images/generations") {
		t.Errorf("dry-run should contain endpoint URL, got: %q", output)
	}
	if !strings.Contains(output, "a cat") {
		t.Errorf("dry-run should contain prompt, got: %q", output)
	}
}
```

- [ ] **Step 7: Run test to verify it passes**

Run: `go test ./internal/cli/ -v -count=1`
Expected: PASS (all tests including new ones)

Run: `go test ./... -count=1`
Expected: PASS (no regressions)

- [ ] **Step 8: Commit**

```bash
git add internal/cli/gen.go internal/cli/edit.go internal/cli/helpers.go internal/cli/gen_test.go internal/cli/edit_test.go
git commit -m "cli: wire gen/edit to resolve credentials from auth manager"
```

---

### Task 8: `potaco use` Command (Non-Interactive)

**Files:**
- Create: `internal/cli/use_cmd.go`
- Create: `internal/cli/use_cmd_test.go`

**Interfaces:**
- Consumes: `auth.New()`, `AuthManager.SetActiveProvider(provider, model)` (from Task 4)
- Produces: `useCmd` (`potaco use <provider> [--model <model>]`)

- [ ] **Step 1: Write the failing test**

```go
// internal/cli/use_cmd_test.go
package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestUseCommandExists(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "use" || strings.HasPrefix(cmd.Use, "use ") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("root command should have 'use' subcommand")
	}
}

func TestUseSwitchesProvider(t *testing.T) {
	resetRootCmdFlags(t)
	resetAuthAddFlags(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)

	// Add two providers
	rootCmd.SetArgs([]string{"auth", "add", "openai", "--api-key", "sk-1", "--force"})
	rootCmd.Execute()
	resetAuthAddFlags(t)
	resetRootCmdFlags(t)
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)

	rootCmd.SetArgs([]string{"auth", "add", "fal", "--api-key", "fal-1", "--force"})
	rootCmd.Execute()
	resetAuthAddFlags(t)
	resetRootCmdFlags(t)
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)

	// Use openai
	rootCmd.SetArgs([]string{"use", "openai"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("use openai: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "openai") {
		t.Errorf("output should mention openai, got: %q", output)
	}
}

func TestUseWithModel(t *testing.T) {
	resetRootCmdFlags(t)
	resetAuthAddFlags(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)

	rootCmd.SetArgs([]string{"auth", "add", "openai", "--api-key", "sk-1", "--force"})
	rootCmd.Execute()
	resetAuthAddFlags(t)
	resetRootCmdFlags(t)
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)

	rootCmd.SetArgs([]string{"use", "openai", "--model", "dall-e-3"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("use with model: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "dall-e-3") {
		t.Errorf("output should mention model, got: %q", output)
	}
}

func TestUseNoArgs(t *testing.T) {
	resetRootCmdFlags(t)
	resetAuthAddFlags(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)

	rootCmd.SetArgs([]string{"use"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("use without args should error in non-interactive mode")
	}
}

func TestUseUnknownProvider(t *testing.T) {
	resetRootCmdFlags(t)
	resetAuthAddFlags(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)

	rootCmd.SetArgs([]string{"use", "nonexistent"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("use with unknown provider should error")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cli/ -run TestUse -v`
Expected: FAIL with undefined `useCmd`

- [ ] **Step 3: Write minimal implementation**

```go
// internal/cli/use_cmd.go
package cli

import (
	"fmt"

	"github.com/ncxton/potaco/internal/auth"
	"github.com/spf13/cobra"
)

var useCmd = &cobra.Command{
	Use:   "use [provider]",
	Short: "Switch the active provider and optionally set the model",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runUse,
}

func init() {
	useCmd.Flags().String("model", "", "set the model for this provider")
	rootCmd.AddCommand(useCmd)
}

func runUse(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return configError(fmt.Errorf("specify a provider: potaco use <provider>"))
	}

	providerName := args[0]
	model, _ := cmd.Flags().GetString("model")

	mgr, err := auth.New()
	if err != nil {
		return configError(fmt.Errorf("init auth: %w", err))
	}

	if err := mgr.SetActiveProvider(providerName, model); err != nil {
		return configError(fmt.Errorf("switch provider: %w", err))
	}

	if model != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Switched to provider '%s' with model '%s'.\n", providerName, model)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Switched to provider '%s'.\n", providerName)
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/cli/ -run TestUse -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/cli/use_cmd.go internal/cli/use_cmd_test.go
git commit -m "cli: add potaco use command for switching active provider"
```

---

### Task 9: Final Verification

**Files:** No new files; verify everything

- [ ] **Step 1: Run full test suite**

Run: `go test ./... -count=1 -v`
Expected: All tests PASS

- [ ] **Step 2: Build the binary**

Run: `go build -o potaco .`
Expected: Successful build

- [ ] **Step 3: Smoke test auth add**

Run: `HOME=/tmp/potaco-test ./potaco auth add openai --api-key sk-test --force`
Expected: "Provider 'openai' added successfully."

- [ ] **Step 4: Smoke test auth list**

Run: `HOME=/tmp/potaco-test ./potaco auth list`
Expected: Lists openai provider

- [ ] **Step 5: Smoke test use**

Run: `HOME=/tmp/potaco-test ./potaco use openai`
Expected: "Switched to provider 'openai'."

- [ ] **Step 6: Smoke test gen dry-run**

Run: `HOME=/tmp/potaco-test ./potaco gen --prompt "a cat" --dry-run`
Expected: JSON with endpoint URL, model, prompt

- [ ] **Step 7: Verify gofmt and go vet**

Run: `gofmt -l . && go vet ./...`
Expected: Clean

- [ ] **Step 8: Check LOC of all new files**

Run: `for f in internal/credential/*.go internal/auth/*.go internal/cli/auth_cmd.go internal/cli/use_cmd.go; do loc=$(awk '!/^[[:space:]]*$/ && !/^[[:space:]]*(\/\/)/' "$f" | wc -l); echo "$loc  $f"; done`
Expected: All files under 250 pure LOC

- [ ] **Step 9: Commit if any cleanup needed**

```bash
git add -A && git commit -m "credential: Phase 2 complete - all tests pass"
```
Skip if nothing to commit.
