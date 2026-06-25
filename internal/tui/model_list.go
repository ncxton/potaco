package tui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/huh"

	"github.com/ncxton/potaco/internal/adapter"
	"github.com/ncxton/potaco/internal/auth"
)

// RunModelList launches the interactive model list for the given provider.
// If providerName is empty, uses the active provider from auth config.
// If apiKey is empty, retrieves it from the credential store.
func RunModelList(providerName, apiKey string) error {
	if providerName == "" || apiKey == "" {
		// Fall back to resolving from auth manager
		mgr, err := auth.New()
		if err != nil {
			return fmt.Errorf("init auth: %w", err)
		}
		if providerName == "" {
			providerName, _, err = mgr.GetActiveProvider()
			if err != nil || providerName == "" {
				return fmt.Errorf("no active provider. Use 'potaco auth add <provider>' to connect one")
			}
		}
		if apiKey == "" {
			k, kErr := mgr.GetActiveAPIKey()
			if kErr == nil {
				apiKey = k
			}
		}
	}
	if apiKey == "" {
		return fmt.Errorf("provider %q is not connected. Use 'potaco auth add %s' first", providerName, providerName)
	}

	ad, err := adapter.Get(providerName, apiKey, adapter.AdapterOpts{})
	if err != nil {
		return fmt.Errorf("create adapter: %w", err)
	}

	models, err := ad.DiscoverModels(context.Background())
	if err != nil {
		return fmt.Errorf("discover models: %w", err)
	}
	if len(models) == 0 {
		return fmt.Errorf("no models found for %s", providerName)
	}

	selected, err := pickModelInteractive(providerName, models)
	if err != nil {
		return err
	}

	params, err := ad.ModelParams(context.Background(), selected)
	if err == nil && len(params) > 0 {
		fmt.Println("\nParameters:")
		for _, p := range params {
			fmt.Printf("  %s (%s) - %s (default: %s)\n", p.Name, p.Type, p.Description, p.Default)
		}
	}

	fmt.Printf("\nSelected: %s\n", selected)
	return nil
}

// pickModelInteractive renders a huh form listing the discovered models and
// returns the selected model ID.
func pickModelInteractive(providerName string, models []adapter.Model) (string, error) {
	options := make([]huh.Option[string], 0, len(models))
	for _, m := range models {
		label := m.DisplayName
		if m.SupportsEdit {
			label += " [edit]"
		}
		label += " - " + m.ID
		options = append(options, huh.NewOption(label, m.ID))
	}

	var selected string
	form := huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().
			Title(fmt.Sprintf("Models for %s:", providerName)).
			Options(options...).
			Value(&selected),
	))
	if err := form.Run(); err != nil {
		return "", fmt.Errorf("model select: %w", err)
	}
	return selected, nil
}
