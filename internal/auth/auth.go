// internal/auth/auth.go
package auth

import (
	"fmt"

	"github.com/ncxton/potaco/internal/config"
	"github.com/ncxton/potaco/internal/credential"
)

// ProviderInfo holds display information about a connected provider.
type ProviderInfo struct {
	Name     string
	Model    string
	BaseURL  string
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

// LoadConfig reads the multi-provider config file.
func (m *AuthManager) LoadConfig() (*config.MultiProviderConfig, error) {
	return config.LoadMultiProvider(m.configPath)
}

// saveConfig writes the multi-provider config file.
func (m *AuthManager) saveConfig(cfg *config.MultiProviderConfig) error {
	return config.SaveMultiProvider(m.configPath, cfg)
}

// Add stores the API key for a provider, creates a config entry, and
// sets it as the active provider.
func (m *AuthManager) Add(provider, apiKey string) error {
	return m.AddProvider(provider, provider, apiKey)
}

// AddProvider stores the API key for a provider key using the given provider
// type for adapter resolution, creates a config entry, and sets it active.
func (m *AuthManager) AddProvider(provider, providerType, apiKey string) error {
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

	pc := cfg.Providers[provider]
	if providerType == "" {
		providerType = config.ResolveProviderType(provider, pc)
	}
	pc.Type = providerType
	if pc.Retries == 0 {
		pc.Retries = 2
	}
	if pc.Timeout == 0 {
		pc.Timeout = 120
	}
	cfg.Providers[provider] = pc

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
			BaseURL:  pc.BaseURL,
			IsActive: cfg.ActiveProvider == name,
		}
		// Check if key exists in credential store
		for _, s := range stored {
			if s == name {
				info.HasKey = true
				break
			}
		}
		// Populate AddedAt from the credential store. If the credential
		// is missing (key not stored), leave AddedAt empty.
		if cred, err := m.store.GetCredential(name); err == nil {
			info.AddedAt = cred.AddedAt.Format("2006-01-02")
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

// SetBaseURL persists a base URL for the given provider. It is separate
// from Add so that Add stays focused on credentials and the basic config
// entry; callers that need to store a base URL (e.g., the custom provider
// auth flow) can do so explicitly after Add.
func (m *AuthManager) SetBaseURL(provider, baseURL string) error {
	cfg, err := m.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if _, ok := cfg.Providers[provider]; !ok {
		return fmt.Errorf("provider %q is not configured. Use 'potaco auth add %s' first", provider, provider)
	}

	pc := cfg.Providers[provider]
	pc.BaseURL = baseURL
	cfg.Providers[provider] = pc

	return m.saveConfig(cfg)
}

func (m *AuthManager) SetModelEdit(provider, model string, edit bool) error {
	cfg, err := m.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	pc, ok := cfg.Providers[provider]
	if !ok {
		return fmt.Errorf("provider %q is not configured. Use 'potaco auth add %s' first", provider, provider)
	}
	if pc.Models == nil {
		pc.Models = make(map[string]config.ModelConfig)
	}
	mc := pc.Models[model]
	mc.Edit = edit
	pc.Models[model] = mc
	cfg.Providers[provider] = pc

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

// GetAPIKey returns the API key for the specified provider.
func (m *AuthManager) GetAPIKey(provider string) (string, error) {
	return m.store.Get(provider)
}

// GetActiveProvider returns the active provider name and model from config.
func (m *AuthManager) GetActiveProvider() (provider, model string, err error) {
	cfg, err := m.LoadConfig()
	if err != nil {
		return "", "", fmt.Errorf("load config: %w", err)
	}
	return cfg.ActiveProvider, cfg.ActiveModel, nil
}
