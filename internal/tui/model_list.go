package tui

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/ncxton/potaco/internal/adapter"
	"github.com/ncxton/potaco/internal/auth"
)

// modelPicker selects a model from the discovered list. It is abstracted so
// tests can substitute a deterministic picker for the interactive Bubble Tea
// component.
type modelPicker func(providerName string, models []adapter.Model) (string, error)

// RunModelList launches the interactive model list for the given provider.
// If providerName is empty, it uses the active provider from auth config.
// If apiKey is empty, it retrieves it from the credential store.
// If baseURL is empty, it resolves it from the provider config.
// The selected model is persisted via SetActiveProvider.
func RunModelList(providerName, apiKey, baseURL string) error {
	return runModelListWithPicker(providerName, apiKey, baseURL, pickModelInteractive)
}

func runModelListWithPicker(providerName, apiKey, baseURL string, picker modelPicker) error {
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
		apiKey, err = mgr.GetAPIKey(providerName)
		if err != nil {
			return fmt.Errorf("provider %q is not connected. Use 'potaco auth add %s' first", providerName, providerName)
		}
	}

	if baseURL == "" {
		cfg, cfgErr := mgr.LoadConfig()
		if cfgErr == nil && cfg != nil {
			if pc, ok := cfg.Providers[providerName]; ok && pc.BaseURL != "" {
				baseURL = pc.BaseURL
			}
		}
	}

	ad, err := adapter.Get(providerName, apiKey, adapter.AdapterOpts{BaseURL: baseURL})
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

	selected, err := picker(providerName, models)
	if err != nil {
		if isCancelled(err) {
			fmt.Println("Cancelled.")
			return nil
		}
		return err
	}
	if selected == "" {
		return nil
	}

	if err := mgr.SetActiveProvider(providerName, selected); err != nil {
		return fmt.Errorf("set active provider: %w", err)
	}
	fmt.Printf("Switched to model '%s'.\n", selected)
	return nil
}

// pickModelInteractive renders a Bubble Tea search program listing the
// discovered models and returns the selected model ID. The user can
// type to filter the list in real-time and navigate with arrow keys.
// Returns huh.ErrUserAborted if the user cancels with Esc or Ctrl-C.
// Returns an empty string when the filter matches no models and the user
// presses Enter.
func pickModelInteractive(providerName string, models []adapter.Model) (string, error) {
	_ = providerName
	m := newSearchModel(models)
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	result, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("model search: %w", err)
	}
	if sm, ok := result.(*searchModel); ok {
		if sm.quitted {
			return "", huh.ErrUserAborted
		}
		return sm.selected, nil
	}
	return "", fmt.Errorf("unexpected model type")
}
