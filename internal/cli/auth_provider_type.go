package cli

import (
	"fmt"
	"os"

	"github.com/ncxton/potaco/internal/config"
	"github.com/spf13/cobra"
)

func resolveAuthProviderType(providerName, providerType string, cfg *config.MultiProviderConfig) (string, error) {
	if providerType != "" {
		if isAllowedAuthProviderType(providerType) {
			return providerType, nil
		}
		return "", fmt.Errorf("unknown provider type: %s", providerType)
	}
	if cfg != nil {
		if pc, ok := cfg.Providers[providerName]; ok && pc.Type != "" {
			return pc.Type, nil
		}
	}
	if providerName == "custom" {
		return "openai-compatible", nil
	}
	if isKnownProvider(providerName, nil) {
		return providerName, nil
	}
	return "", fmt.Errorf("provider type required for %q: use --type openai-compatible, openai, fal, or vercel", providerName)
}

func isAllowedAuthProviderType(providerType string) bool {
	switch providerType {
	case "openai", "fal", "vercel", "openai-compatible":
		return true
	default:
		return false
	}
}

func shouldPersistAuthBaseURL(cmd *cobra.Command, providerType, baseURL string) bool {
	if baseURL == "" {
		return false
	}
	return providerType == "openai-compatible" || cmd.Flags().Changed("base-url") || os.Getenv("POTACO_BASE_URL") != ""
}
