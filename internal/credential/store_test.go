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
	// Create a store, write a credential, and verify the file permissions are 0600.
	dir := t.TempDir()
	credPath := filepath.Join(dir, "credentials.enc")
	saltPath := filepath.Join(dir, ".salt")
	s, err := New(credPath, saltPath)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	s.Set("openai", "sk-test")

	info, err := os.Stat(credPath)
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
