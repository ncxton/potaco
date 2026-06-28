package cli

import "github.com/ncxton/potaco/internal/config"

type authAddInteractiveInput struct {
	providerName     string
	providerTypeFlag string
	apiKey           string
	baseURL          string
	interactive      bool
}

func shouldRunInteractiveAuthAdd(input authAddInteractiveInput) bool {
	if !input.interactive {
		return false
	}
	if input.providerName == "" {
		return true
	}
	if input.providerTypeFlag != "" && !isAllowedAuthProviderType(input.providerTypeFlag) {
		return false
	}
	if input.providerTypeFlag == "" && !isKnownProvider(input.providerName, nil) {
		return true
	}
	if input.apiKey == "" {
		return true
	}
	return authAddRequiresBaseURL(input.providerName, input.providerTypeFlag) && input.baseURL == ""
}

func authAddRequiresBaseURL(providerName, providerType string) bool {
	return config.ProviderRequiresBaseURL(providerName, providerType)
}
