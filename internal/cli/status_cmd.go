package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"charm.land/lipgloss/v2"
	"github.com/ncxton/potaco/internal/auth"
	"github.com/ncxton/potaco/internal/config"
	"github.com/ncxton/potaco/internal/tui"
	"github.com/spf13/cobra"
)

var (
	statusLabelStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	statusActiveStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current provider, model, and connection status",
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	mgr, err := auth.New()
	if err != nil {
		return configError(fmt.Errorf("init auth: %w", err))
	}

	provider, model, _ := mgr.GetActiveProvider()
	providers := mgr.List()
	configPath := config.DefaultConfigPath()
	credPath := config.DefaultCredentialPath()

	jsonMode, _ := cmd.Root().PersistentFlags().GetBool("json")
	out := cmd.OutOrStdout()

	if jsonMode {
		return printStatusJSON(out, provider, model, configPath, credPath, providers)
	}

	if provider == "" {
		fmt.Fprintln(out, "No active provider configured.")
		fmt.Fprintln(out, "Use 'potaco auth add <provider>' to connect one.")
	} else if tui.IsTTY() {
		fmt.Fprintf(out, "%s %s\n", statusLabelStyle.Render("Active provider:"), statusActiveStyle.Render(provider))
		fmt.Fprintf(out, "%s %s\n", statusLabelStyle.Render("Active model:"), model)
	} else {
		fmt.Fprintf(out, "Active provider: %s\n", provider)
		fmt.Fprintf(out, "Active model:    %s\n", model)
	}
	fmt.Fprintln(out)
	fmt.Fprintf(out, "Config file:     %s\n", configPath)
	fmt.Fprintf(out, "Credentials:     %s\n", credPath)
	fmt.Fprintln(out)

	if len(providers) == 0 {
		if provider != "" {
			fmt.Fprintln(out, "No providers connected.")
		}
	} else {
		fmt.Fprintln(out, "Connected providers:")
		for _, p := range providers {
			active := ""
			if p.IsActive {
				active = " (active)"
			}
			keyStatus := "missing"
			if p.HasKey {
				keyStatus = "configured"
			}
			added := ""
			if p.AddedAt != "" {
				added = "  added: " + p.AddedAt
			}
			fmt.Fprintf(out, "  %s\t%s\tkey: %s%s%s\n", p.Name, p.Model, keyStatus, active, added)
		}
	}

	return nil
}

func printStatusJSON(out io.Writer, provider, model, configPath, credPath string, providers []auth.ProviderInfo) error {
	type providerJSON struct {
		Name     string `json:"name"`
		Model    string `json:"model"`
		HasKey   bool   `json:"has_key"`
		IsActive bool   `json:"is_active"`
		AddedAt  string `json:"added_at,omitempty"`
	}

	pjs := make([]providerJSON, 0, len(providers))
	for _, p := range providers {
		pjs = append(pjs, providerJSON{
			Name:     p.Name,
			Model:    p.Model,
			HasKey:   p.HasKey,
			IsActive: p.IsActive,
			AddedAt:  p.AddedAt,
		})
	}

	status := map[string]any{
		"active_provider": provider,
		"active_model":    model,
		"config_path":     configPath,
		"credential_path": credPath,
		"providers":       pjs,
	}

	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal JSON: %w", err)
	}
	fmt.Fprintln(out, string(data))
	return nil
}
