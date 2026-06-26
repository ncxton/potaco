package tui

import (
	"fmt"

	"github.com/charmbracelet/huh"

	"github.com/ncxton/potaco/internal/auth"
)

// RunAuthRemove launches the interactive auth remove flow.
// If providerName is empty, shows a provider picker first.
// Shows a confirmation prompt before removing. Returns nil when the
// user cancels (pressed Esc), and prints "Cancelled." to stdout.
func RunAuthRemove(providerName string) error {
	mgr, err := auth.New()
	if err != nil {
		return fmt.Errorf("init auth: %w", err)
	}

	// If no provider name given, show picker
	if providerName == "" {
		providers := mgr.List()
		if len(providers) == 0 {
			fmt.Println("No providers connected.")
			return nil
		}

		options := make([]huh.Option[string], 0, len(providers))
		for _, p := range providers {
			label := p.Name
			if p.IsActive {
				label += " (active)"
			}
			label += " - " + p.Model
			options = append(options, huh.NewOption(label, p.Name))
		}

		selectForm := newForm(huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select a provider to remove:").
				Options(options...).
				Value(&providerName),
		))
		if err := selectForm.Run(); err != nil {
			if isCancelled(err) {
				fmt.Println("Cancelled.")
				return nil
			}
			return fmt.Errorf("provider select: %w", err)
		}
	}

	// Confirmation prompt
	confirmed, err := ConfirmAction(fmt.Sprintf("Remove provider '%s' and its credentials?", providerName))
	if err != nil {
		if isCancelled(err) {
			fmt.Println("Cancelled.")
			return nil
		}
		return fmt.Errorf("confirm: %w", err)
	}
	if !confirmed {
		fmt.Println("Cancelled.")
		return nil
	}

	// Execute removal
	if err := mgr.Remove(providerName); err != nil {
		return fmt.Errorf("remove provider: %w", err)
	}

	fmt.Printf("Provider '%s' removed.\n", providerName)
	return nil
}
