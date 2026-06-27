package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/ncxton/potaco/internal/adapter"
	"github.com/ncxton/potaco/internal/auth"
	"github.com/ncxton/potaco/internal/config"
	"github.com/ncxton/potaco/internal/tui"
	"github.com/spf13/cobra"
)

var modelsCmd = &cobra.Command{
	Use:   "models [provider]",
	Short: "Pick a model for the active or specified provider",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runModels,
}

var modelsListCmd = &cobra.Command{
	Use:   "list [provider]",
	Short: "List available models for the active or specified provider",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runModelsList,
}

func init() {
	modelsCmd.PersistentFlags().String("base-url", "", "override API base URL")
	modelsCmd.PersistentFlags().String("api-key", "", "override API key")
	modelsCmd.AddCommand(modelsListCmd)
	rootCmd.AddCommand(modelsCmd)
}

type resolvedModelsProvider struct {
	ProviderName string
	AdapterType  string
	APIKey       string
	BaseURL      string
}

func runModels(cmd *cobra.Command, args []string) error {
	resolved, err := resolveModelsProvider(cmd, args)
	if err != nil {
		return err
	}

	if tui.IsInteractive() {
		return tui.RunModelList(resolved.ProviderName, resolved.APIKey, resolved.BaseURL)
	}

	return printModels(cmd, resolved)
}

func runModelsList(cmd *cobra.Command, args []string) error {
	resolved, err := resolveModelsProvider(cmd, args)
	if err != nil {
		return err
	}
	return printModels(cmd, resolved)
}

// resolveModelsProvider resolves the provider, API key, and base URL for the
// models command and its list subcommand. The provider comes from the first
// positional argument, or the active provider when no argument is given.
// The API key and base URL follow flag > env > config > preset precedence.
func resolveModelsProvider(cmd *cobra.Command, args []string) (resolvedModelsProvider, error) {
	mgr, err := auth.New()
	if err != nil {
		return resolvedModelsProvider{}, configUserErr(
			"Could not load configuration.",
			"Check that ~/.potaco/ is readable.",
			fmt.Errorf("init auth: %w", err),
		)
	}

	cfg, _ := mgr.LoadConfig()
	providerName := ""
	if len(args) > 0 {
		providerName = args[0]
		if !isKnownProvider(providerName, cfg) {
			return resolvedModelsProvider{}, configUserErr(
				fmt.Sprintf("Unknown provider '%s'.", providerName),
				"Run 'potaco auth list' to see connected providers.",
				fmt.Errorf("unknown provider: %s", providerName),
			)
		}
	} else {
		providerName, _, err = mgr.GetActiveProvider()
		if err != nil || providerName == "" {
			return resolvedModelsProvider{}, configUserErr(
				"No active provider configured.",
				"Run 'potaco auth add <provider>' to connect one.",
				fmt.Errorf("no active provider"),
			)
		}
	}

	apiKey, err := resolveModelsAPIKey(cmd, mgr, providerName)
	if err != nil {
		return resolvedModelsProvider{}, configUserErr(
			fmt.Sprintf("Provider '%s' is not connected.", providerName),
			fmt.Sprintf("Run 'potaco auth add %s' to store an API key.", providerName),
			fmt.Errorf("provider %q is not connected: %w", providerName, err),
		)
	}

	pc := config.ProviderConfig{}
	if cfg != nil {
		if configured, ok := cfg.Providers[providerName]; ok {
			pc = configured
		}
	}
	providerType := config.ResolveProviderType(providerName, pc)
	baseURL := resolveBaseURL(cmd, providerName, cfg)
	if providerType == "openai-compatible" && baseURL == "" {
		return resolvedModelsProvider{}, configUserErr(
			"A base URL is required for OpenAI-compatible providers.",
			"Use --base-url, set POTACO_BASE_URL, or run 'potaco config set --base-url <url>'.",
			fmt.Errorf("base URL required for provider %s", providerName),
		)
	}

	return resolvedModelsProvider{
		ProviderName: providerName,
		AdapterType:  config.AdapterType(providerType),
		APIKey:       apiKey,
		BaseURL:      baseURL,
	}, nil
}

func isKnownProvider(name string, cfg *config.MultiProviderConfig) bool {
	for _, n := range adapter.List() {
		if n == name {
			return true
		}
	}
	if cfg != nil {
		_, ok := cfg.Providers[name]
		return ok
	}
	return false
}

func resolveModelsAPIKey(cmd *cobra.Command, mgr *auth.AuthManager, providerName string) (string, error) {
	if v := flagString(cmd, "api-key"); v != "" {
		return v, nil
	}
	if v := os.Getenv("POTACO_API_KEY"); v != "" {
		return v, nil
	}

	cfg, err := mgr.LoadConfig()
	if err != nil {
		return "", err
	}
	if cfg == nil || cfg.Providers == nil {
		return "", fmt.Errorf("provider not configured")
	}
	if _, ok := cfg.Providers[providerName]; !ok {
		return "", fmt.Errorf("provider not configured")
	}

	return mgr.GetAPIKey(providerName)
}

func printModels(cmd *cobra.Command, resolved resolvedModelsProvider) error {
	opts := adapter.AdapterOpts{BaseURL: resolved.BaseURL}
	ad, err := adapter.Get(resolved.AdapterType, resolved.APIKey, opts)
	if err != nil {
		return configUserErr(
			fmt.Sprintf("Could not connect to provider '%s'.", resolved.ProviderName),
			"Check that the provider name is correct and the provider is registered.",
			fmt.Errorf("create adapter: %w", err),
		)
	}

	models, err := ad.DiscoverModels(context.Background())
	if err != nil {
		return apiUserErr(
			"Could not discover models.",
			"Check your API key, base URL, and network connection.",
			fmt.Errorf("discover models: %w", err),
		)
	}

	out := cmd.OutOrStdout()
	jsonMode, _ := cmd.Root().PersistentFlags().GetBool("json")
	if jsonMode {
		return printModelsJSON(out, models)
	}
	return printModelsText(out, models)
}

func printModelsText(out io.Writer, models []adapter.Model) error {
	if len(models) == 0 {
		fmt.Fprintln(out, "No models found.")
		return nil
	}
	fmt.Fprintf(out, "%-40s %-20s %s\n", "MODEL ID", "DISPLAY NAME", "CAPABILITIES")
	for _, m := range models {
		editBadge := ""
		if m.SupportsEdit {
			editBadge = " [edit]"
		}
		caps := fmt.Sprintf("%v", m.Capabilities)
		fmt.Fprintf(out, "%-40s %-20s%s %s\n", m.ID, m.DisplayName, editBadge, caps)
	}
	return nil
}

func printModelsJSON(out io.Writer, models []adapter.Model) error {
	type modelJSON struct {
		ID           string   `json:"id"`
		DisplayName  string   `json:"display_name"`
		SupportsGen  bool     `json:"supports_gen"`
		SupportsEdit bool     `json:"supports_edit"`
		Capabilities []string `json:"capabilities"`
	}
	items := make([]modelJSON, 0, len(models))
	for _, m := range models {
		items = append(items, modelJSON{
			ID:           m.ID,
			DisplayName:  m.DisplayName,
			SupportsGen:  m.SupportsGen,
			SupportsEdit: m.SupportsEdit,
			Capabilities: m.Capabilities,
		})
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal JSON: %w", err)
	}
	fmt.Fprintln(out, string(data))
	return nil
}
