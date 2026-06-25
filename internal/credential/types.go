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
