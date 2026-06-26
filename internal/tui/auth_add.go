package tui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/huh"

	"github.com/ncxton/potaco/internal/adapter"
	"github.com/ncxton/potaco/internal/auth"
)

// RunAuthAdd launches the interactive auth add flow using huh forms.
// It prompts for an API key, verifies the provider, discovers models,
// lets the user pick a model, and stores the credential.
func RunAuthAdd(providerName string) error {
	if providerName == "" {
		return fmt.Errorf("provider name is required")
	}

	known := false
	for _, name := range adapter.List() {
		if name == providerName {
			known = true
			break
		}
	}
	if !known {
		return fmt.Errorf("unknown provider: %s (available: %v)", providerName, adapter.List())
	}

	var apiKey string
	keyForm := newForm(huh.NewGroup(
		huh.NewInput().
			Title(fmt.Sprintf("Enter API key for %s:", providerName)).
			EchoMode(huh.EchoModePassword).
			Value(&apiKey),
	))
	if err := keyForm.Run(); err != nil {
		if isCancelled(err) {
			fmt.Println("Cancelled.")
			return nil
		}
		return fmt.Errorf("key input: %w", err)
	}
	if apiKey == "" {
		return fmt.Errorf("API key is required")
	}

	ad, err := adapter.Get(providerName, apiKey, adapter.AdapterOpts{})
	if err != nil {
		return fmt.Errorf("create adapter: %w", err)
	}

	verifyErr := ad.Verify(context.Background())
	if verifyErr != nil {
		var proceed bool
		confirmForm := newForm(huh.NewGroup(
			huh.NewConfirm().
				Title(fmt.Sprintf("Verification failed: %s\nAdd anyway?", verifyErr)).
				Value(&proceed),
		))
		if err := confirmForm.Run(); err != nil {
			if isCancelled(err) {
				fmt.Println("Cancelled.")
				return nil
			}
			return fmt.Errorf("confirm: %w", err)
		}
		if !proceed {
			return fmt.Errorf("cancelled by user")
		}
	}

	modelID := ""
	models, discoverErr := ad.DiscoverModels(context.Background())
	if discoverErr == nil && len(models) > 0 {
		options := make([]huh.Option[string], 0, len(models))
		for _, m := range models {
			label := m.DisplayName
			if m.SupportsEdit {
				label += " (supports edit)"
			}
			options = append(options, huh.NewOption(label, m.ID))
		}
		selectForm := newForm(huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select a model:").
				Options(options...).
				Value(&modelID),
		))
		if err := selectForm.Run(); err != nil {
			if isCancelled(err) {
				fmt.Println("Cancelled.")
				return nil
			}
			return fmt.Errorf("model select: %w", err)
		}
	}

	mgr, err := auth.New()
	if err != nil {
		return fmt.Errorf("init auth: %w", err)
	}
	if err := mgr.Add(providerName, apiKey, true); err != nil {
		return fmt.Errorf("add provider: %w", err)
	}
	if modelID != "" {
		if err := mgr.SetActiveProvider(providerName, modelID); err != nil {
			return fmt.Errorf("set model: %w", err)
		}
	}

	fmt.Printf("Provider '%s' added successfully.\n", providerName)
	return nil
}
