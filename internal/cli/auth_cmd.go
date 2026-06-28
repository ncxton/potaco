package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/ncxton/potaco/internal/adapter"
	"github.com/ncxton/potaco/internal/auth"
	"github.com/ncxton/potaco/internal/config"
	"github.com/ncxton/potaco/internal/tui"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage provider credentials",
}

var authAddCmd = &cobra.Command{
	Use:   "add [provider]",
	Short: "Connect to a provider by storing its API key",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runAuthAdd,
}

var authRemoveCmd = &cobra.Command{
	Use:     "remove [provider]",
	Short:   "Remove a provider's credentials and config",
	Args:    cobra.MaximumNArgs(1),
	RunE:    runAuthRemove,
	Aliases: []string{"rm"},
}

var authListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all connected providers",
	RunE:    runAuthList,
	Aliases: []string{"ls"},
}

func init() {
	authAddCmd.Flags().String("api-key", "", "API key for the provider")
	authAddCmd.Flags().Bool("force", false, "skip provider verification")
	authAddCmd.Flags().String("model", "", "override the default model for this provider")
	authAddCmd.Flags().String("base-url", "", "override API base URL for verification")
	authAddCmd.Flags().String("type", "", "provider adapter type (openai, fal, vercel, openai-compatible)")

	authCmd.AddCommand(authAddCmd)
	authCmd.AddCommand(authRemoveCmd)
	authCmd.AddCommand(authListCmd)
	rootCmd.AddCommand(authCmd)
}

func runAuthAdd(cmd *cobra.Command, args []string) error {
	providerName := ""
	if len(args) > 0 {
		providerName = args[0]
	}

	if providerName == "" {
		if !tui.IsInteractive() {
			return configError(fmt.Errorf("specify a provider: potaco auth add <provider>"))
		}
		return tui.RunAuthAdd("")
	}

	mgr, err := auth.New()
	if err != nil {
		return configError(fmt.Errorf("init auth: %w", err))
	}

	cfg, _ := mgr.LoadConfig()
	baseURL := resolveAuthAddBaseURL(cmd, providerName, cfg)
	providerTypeFlag, _ := cmd.Flags().GetString("type")

	apiKeyFlag, _ := cmd.Flags().GetString("api-key")
	envKey := os.Getenv("POTACO_API_KEY")
	apiKey := apiKeyFlag
	if apiKey == "" {
		apiKey = envKey
	}

	if shouldRunInteractiveAuthAdd(authAddInteractiveInput{
		providerName:     providerName,
		providerTypeFlag: providerTypeFlag,
		apiKey:           apiKey,
		baseURL:          baseURL,
		interactive:      tui.IsInteractive(),
	}) {
		return tui.RunAuthAdd(providerName)
	}

	providerType, err := resolveAuthProviderType(providerName, providerTypeFlag, cfg)
	if err != nil {
		return configError(err)
	}
	adapterType := config.AdapterType(providerType)
	if baseURL == "" && !authAddRequiresBaseURL(providerName, providerType) {
		baseURL = resolveBaseURL(cmd, providerName, cfg)
	}

	if apiKey == "" {
		return configError(fmt.Errorf("API key required: use --api-key or set POTACO_API_KEY"))
	}
	if authAddRequiresBaseURL(providerName, providerType) && baseURL == "" {
		return configUserErr(
			"A base URL is required for this provider.",
			"Use --base-url or set POTACO_BASE_URL.",
			fmt.Errorf("base URL required for provider %s", providerName),
		)
	}

	force, _ := cmd.Flags().GetBool("force")
	if !force {
		ad, err := adapter.Get(adapterType, apiKey, adapter.AdapterOpts{BaseURL: baseURL})
		if err != nil {
			return configError(fmt.Errorf("create adapter: %w", err))
		}
		if err := ad.Verify(context.Background()); err != nil {
			return apiError(fmt.Errorf("verification failed: %w\nUse --force to add anyway", err))
		}
	}

	if err := mgr.AddProvider(providerName, providerType, apiKey); err != nil {
		return configError(fmt.Errorf("add provider: %w", err))
	}

	if shouldPersistAuthBaseURL(cmd, providerType, baseURL) {
		if err := mgr.SetBaseURL(providerName, baseURL); err != nil {
			return configError(fmt.Errorf("set base URL: %w", err))
		}
	}

	model, _ := cmd.Flags().GetString("model")
	if model != "" {
		if err := mgr.SetActiveProvider(providerName, model); err != nil {
			return configError(fmt.Errorf("set model: %w", err))
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Provider '%s' added successfully.\n", providerName)
	fmt.Fprintf(cmd.OutOrStdout(), "Use 'potaco use %s' to switch to it.\n", providerName)
	return nil
}

func resolveAuthAddBaseURL(cmd *cobra.Command, providerName string, cfg *config.MultiProviderConfig) string {
	if cmd.Flags().Changed("base-url") {
		return strings.TrimRight(flagString(cmd, "base-url"), "/")
	}
	if v := os.Getenv("POTACO_BASE_URL"); v != "" {
		return strings.TrimRight(v, "/")
	}
	if cfg != nil {
		if pc, ok := cfg.Providers[providerName]; ok && pc.BaseURL != "" {
			return strings.TrimRight(pc.BaseURL, "/")
		}
	}
	return ""
}

func runAuthRemove(cmd *cobra.Command, args []string) error {
	providerName := ""
	if len(args) > 0 {
		providerName = args[0]
	}

	if !tui.IsInteractive() {
		if providerName == "" {
			return configError(fmt.Errorf("specify a provider: potaco auth remove <provider>"))
		}

		mgr, err := auth.New()
		if err != nil {
			return configError(fmt.Errorf("init auth: %w", err))
		}
		if err := mgr.Remove(providerName); err != nil {
			return configError(fmt.Errorf("remove provider: %w", err))
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Provider '%s' removed.\n", providerName)
		return nil
	}

	return tui.RunAuthRemove(providerName)
}

func runAuthList(cmd *cobra.Command, args []string) error {
	mgr, err := auth.New()
	if err != nil {
		return configError(fmt.Errorf("init auth: %w", err))
	}

	providers := mgr.List()
	jsonMode, _ := cmd.Root().PersistentFlags().GetBool("json")
	out := cmd.OutOrStdout()

	if jsonMode {
		type providerJSON struct {
			Name     string `json:"name"`
			Model    string `json:"model"`
			BaseURL  string `json:"base_url,omitempty"`
			HasKey   bool   `json:"has_key"`
			IsActive bool   `json:"is_active"`
		}
		items := make([]providerJSON, 0, len(providers))
		for _, p := range providers {
			items = append(items, providerJSON{
				Name:     p.Name,
				Model:    p.Model,
				BaseURL:  p.BaseURL,
				HasKey:   p.HasKey,
				IsActive: p.IsActive,
			})
		}
		data, err := json.MarshalIndent(items, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal JSON: %w", err)
		}
		fmt.Fprintln(out, string(data))
		return nil
	}

	if len(providers) == 0 {
		fmt.Fprintln(out, "No providers connected. Use 'potaco auth add <provider>' to connect one.")
		return nil
	}

	if tui.IsTTY() {
		titleStyle := lipgloss.NewStyle().Bold(true)
		activeStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
		keyOkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
		keyMissingStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203"))

		fmt.Fprintln(out, titleStyle.Render("Connected providers:"))
		fmt.Fprintln(out)
		for _, p := range providers {
			name := p.Name
			if p.IsActive {
				name = activeStyle.Render(p.Name + " (active)")
			}
			keyStatus := keyOkStyle.Render("configured")
			if !p.HasKey {
				keyStatus = keyMissingStyle.Render("missing")
			}
			fmt.Fprintf(out, "  %s  %s  key: %s\n", name, p.Model, keyStatus)
			if p.BaseURL != "" {
				fmt.Fprintf(out, "    base_url: %s\n", p.BaseURL)
			}
		}
		return nil
	}

	fmt.Fprintln(out, "Connected providers:")
	fmt.Fprintln(out)
	for _, p := range providers {
		active := ""
		if p.IsActive {
			active = " (active)"
		}
		keyStatus := "missing"
		if p.HasKey {
			keyStatus = "configured"
		}
		fmt.Fprintf(out, "  %s\t%s\tkey: %s%s\n", p.Name, p.Model, keyStatus, active)
		if p.BaseURL != "" {
			fmt.Fprintf(out, "    base_url: %s\n", p.BaseURL)
		}
	}
	return nil
}
