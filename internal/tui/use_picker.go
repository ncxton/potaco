package tui

import (
	"fmt"

	"github.com/charmbracelet/huh"

	"github.com/ncxton/potaco/internal/auth"
)

// RunUsePicker launches the interactive provider/model picker. It lists all
// connected providers, lets the user select one, optionally change its model,
// and sets the result as the active provider.
func RunUsePicker() error {
	mgr, err := auth.New()
	if err != nil {
		return fmt.Errorf("init auth: %w", err)
	}

	providers := mgr.List()
	if len(providers) == 0 {
		return fmt.Errorf("no providers connected. Use 'potaco auth add <provider>' to connect one")
	}

	providerName, err := pickProvider(providers)
	if err != nil {
		return err
	}

	modelID, err := pickModel(providers, providerName)
	if err != nil {
		return err
	}

	if err := mgr.SetActiveProvider(providerName, modelID); err != nil {
		return fmt.Errorf("set active provider: %w", err)
	}

	fmt.Printf("Switched to provider '%s'.\n", providerName)
	return nil
}

// pickProvider renders a huh form listing connected providers and returns the
// chosen provider name.
func pickProvider(providers []auth.ProviderInfo) (string, error) {
	options := make([]huh.Option[string], 0, len(providers))
	for _, p := range providers {
		label := p.Name
		if p.IsActive {
			label += " (active)"
		}
		label += " - " + p.Model
		options = append(options, huh.NewOption(label, p.Name))
	}

	var providerName string
	form := huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().
			Title("Select a provider:").
			Options(options...).
			Value(&providerName),
	))
	if err := form.Run(); err != nil {
		return "", fmt.Errorf("provider select: %w", err)
	}
	return providerName, nil
}

// pickModel asks whether to change the model for the selected provider and,
// if so, prompts for the new model ID. It returns the model to set, or an
// empty string to keep the existing configured model.
func pickModel(providers []auth.ProviderInfo, providerName string) (string, error) {
	var changeModel bool
	confirmForm := huh.NewForm(huh.NewGroup(
		huh.NewConfirm().
			Title("Change the model for this provider?").
			Value(&changeModel),
	))
	if err := confirmForm.Run(); err != nil {
		return "", fmt.Errorf("confirm model change: %w", err)
	}

	if !changeModel {
		return "", nil
	}

	modelID := currentModel(providers, providerName)
	modelForm := huh.NewForm(huh.NewGroup(
		huh.NewInput().
			Title("Enter model ID:").
			Value(&modelID),
	))
	if err := modelForm.Run(); err != nil {
		return "", fmt.Errorf("model input: %w", err)
	}
	return modelID, nil
}

// currentModel returns the configured model for providerName, or an empty
// string if the provider is not found.
func currentModel(providers []auth.ProviderInfo, providerName string) string {
	for _, p := range providers {
		if p.Name == providerName {
			return p.Model
		}
	}
	return ""
}
