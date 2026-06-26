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
			if providerName != "" {
				k, kErr := mgr.GetAPIKey(providerName)
				if kErr == nil {
					apiKey = k
				}
			} else {
				k, kErr := mgr.GetActiveAPIKey()
				if kErr == nil {
					apiKey = k
				}
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

	fmt.Printf("\nSelected: %s\n", selected)
	return nil
}

// pickModelInteractive renders a Bubble Tea search program listing the
// discovered models and returns the selected model ID. The user can
// type to filter the list in real-time and navigate with arrow keys.
// Returns huh.ErrUserAborted if the user cancels with Esc or Ctrl-C.
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
