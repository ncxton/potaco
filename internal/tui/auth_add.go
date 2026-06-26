package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/ncxton/potaco/internal/adapter"
	"github.com/ncxton/potaco/internal/auth"
)

// errCancelled is returned by form helpers when the user aborts a TUI flow.
// It is normalized to a nil error by the callers so the command exits cleanly.
var errCancelled = errors.New("cancelled")

// RunAuthAdd launches the interactive auth add flow using huh forms.
// It prompts for an API key, verifies the provider, discovers models,
// lets the user pick a model, and stores the credential.
// If providerName is empty, shows a provider picker first.
func RunAuthAdd(providerName string) error {
	providerName, err := ensureProvider(providerName)
	if err != nil {
		return normalizeCancel(err)
	}
	if providerName == "" {
		return nil
	}
	if !isKnownProvider(providerName) {
		return fmt.Errorf("unknown provider: %s (available: %v)", providerName, adapter.List())
	}

	baseURL, err := promptBaseURL(providerName)
	if err != nil {
		return normalizeCancel(err)
	}

	apiKey, err := promptAPIKey(providerName)
	if err != nil {
		return normalizeCancel(err)
	}

	ad, err := adapter.Get(providerName, apiKey, adapter.AdapterOpts{BaseURL: baseURL})
	if err != nil {
		return fmt.Errorf("create adapter: %w", err)
	}

	verifyErr := ad.Verify(context.Background())
	if verifyErr != nil {
		proceed, err := confirmVerification(verifyErr)
		if err != nil {
			return normalizeCancel(err)
		}
		if !proceed {
			return fmt.Errorf("cancelled by user")
		}
	}

	modelID := ""
	models, discoverErr := ad.DiscoverModels(context.Background())
	if discoverErr == nil {
		modelID, err = promptModel(models)
		if err != nil {
			return normalizeCancel(err)
		}
	}

	return addProvider(providerName, apiKey, baseURL, modelID)
}

// normalizeCancel converts errCancelled into a nil error so the CLI exits
// silently after the user aborts a TUI flow.
func normalizeCancel(err error) error {
	if errors.Is(err, errCancelled) {
		return nil
	}
	return err
}

// ensureProvider returns the provider name, prompting for one when the caller
// passes an empty string in interactive mode. In non-interactive mode an empty
// name is an error.
func ensureProvider(providerName string) (string, error) {
	if providerName != "" {
		return providerName, nil
	}
	if !IsInteractive() {
		return "", fmt.Errorf("specify a provider: potaco auth add <provider>")
	}
	return promptProvider()
}

// promptProvider shows a picker with all registered providers.
func promptProvider() (string, error) {
	available := adapter.List()
	if len(available) == 0 {
		return "", fmt.Errorf("no providers available")
	}
	options := make([]huh.Option[string], 0, len(available))
	for _, name := range available {
		options = append(options, huh.NewOption(name, name))
	}
	var providerName string
	form := newForm(huh.NewGroup(
		huh.NewSelect[string]().
			Title("Select a provider to connect:").
			Options(options...).
			Value(&providerName),
	))
	if err := runForm(form, "provider select"); err != nil {
		return "", err
	}
	return providerName, nil
}

// isKnownProvider reports whether name is a registered provider.
func isKnownProvider(name string) bool {
	for _, n := range adapter.List() {
		if n == name {
			return true
		}
	}
	return false
}

// promptBaseURL prompts for a base URL when the provider is custom.
// For other providers it returns an empty string.
func promptBaseURL(providerName string) (string, error) {
	if providerName != "custom" {
		return "", nil
	}
	var baseURL string
	form := newForm(huh.NewGroup(
		huh.NewInput().
			Title("Enter base URL for the custom provider:").
			Value(&baseURL),
	))
	if err := runForm(form, "base URL input"); err != nil {
		return "", err
	}
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" {
		return "", fmt.Errorf("base URL is required for the custom provider")
	}
	return baseURL, nil
}

// promptAPIKey prompts for the provider API key.
func promptAPIKey(providerName string) (string, error) {
	var apiKey string
	form := newForm(huh.NewGroup(
		huh.NewInput().
			Title(fmt.Sprintf("Enter API key for %s:", providerName)).
			EchoMode(huh.EchoModePassword).
			Value(&apiKey),
	))
	if err := runForm(form, "key input"); err != nil {
		return "", err
	}
	if apiKey == "" {
		return "", fmt.Errorf("API key is required")
	}
	return apiKey, nil
}

// runForm runs a huh form and maps user aborts to errCancelled.
func runForm(form *huh.Form, label string) error {
	if err := form.Run(); err != nil {
		if isCancelled(err) {
			fmt.Println("Cancelled.")
			return errCancelled
		}
		return fmt.Errorf("%s: %w", label, err)
	}
	return nil
}

// confirmVerification asks whether to add a provider after verification failed.
func confirmVerification(verifyErr error) (bool, error) {
	var proceed bool
	form := newForm(huh.NewGroup(
		huh.NewConfirm().
			Title(fmt.Sprintf("Verification failed: %s\nAdd anyway?", verifyErr)).
			Value(&proceed),
	))
	if err := runForm(form, "confirm"); err != nil {
		return false, err
	}
	return proceed, nil
}

// promptModel shows a model picker when discovery succeeds.
func promptModel(models []adapter.Model) (string, error) {
	if len(models) == 0 {
		return "", nil
	}
	options := make([]huh.Option[string], 0, len(models))
	for _, m := range models {
		label := m.DisplayName
		if m.SupportsEdit {
			label += " (supports edit)"
		}
		options = append(options, huh.NewOption(label, m.ID))
	}
	var modelID string
	form := newForm(huh.NewGroup(
		huh.NewSelect[string]().
			Title("Select a model:").
			Options(options...).
			Value(&modelID),
	))
	if err := runForm(form, "model select"); err != nil {
		return "", err
	}
	return modelID, nil
}

// addProvider stores the credential and config for the connected provider.
func addProvider(providerName, apiKey, baseURL, modelID string) error {
	mgr, err := auth.New()
	if err != nil {
		return fmt.Errorf("init auth: %w", err)
	}
	if err := mgr.Add(providerName, apiKey); err != nil {
		return fmt.Errorf("add provider: %w", err)
	}
	if providerName == "custom" && baseURL != "" {
		if err := mgr.SetBaseURL(providerName, baseURL); err != nil {
			return fmt.Errorf("set base URL: %w", err)
		}
	}
	if modelID != "" {
		if err := mgr.SetActiveProvider(providerName, modelID); err != nil {
			return fmt.Errorf("set model: %w", err)
		}
	}
	fmt.Printf("Provider '%s' added successfully.\n", providerName)
	return nil
}
