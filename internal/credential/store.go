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
	key      []byte
	data     credentialData
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
