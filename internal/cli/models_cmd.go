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
	"github.com/ncxton/potaco/internal/credential"
	"github.com/spf13/cobra"
)

var modelsCmd = &cobra.Command{
	Use:   "models [provider]",
	Short: "List available image models for the active or specified provider",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runModels,
}

func init() {
	modelsCmd.Flags().String("base-url", "", "override API base URL")
	modelsCmd.Flags().String("api-key", "", "override API key")
	rootCmd.AddCommand(modelsCmd)
}

func runModels(cmd *cobra.Command, args []string) error {
	mgr, err := auth.New()
	if err != nil {
		return configError(fmt.Errorf("init auth: %w", err))
	}

	providerName := ""
	if len(args) > 0 {
		providerName = args[0]
	} else {
		providerName, _, err = mgr.GetActiveProvider()
		if err != nil || providerName == "" {
			return configError(fmt.Errorf("no active provider. Use 'potaco auth add <provider>' to connect one"))
		}
	}

	apiKey := flagString(cmd, "api-key")
	if apiKey == "" {
		if v := os.Getenv("POTACO_API_KEY"); v != "" {
			apiKey = v
		}
	}
	if apiKey == "" && len(args) == 0 {
		k, kErr := mgr.GetActiveAPIKey()
		if kErr == nil {
			apiKey = k
		}
	}
	if apiKey == "" && len(args) > 0 {
		cfg, cfgErr := mgr.LoadConfig()
		if cfgErr == nil && cfg != nil {
			if _, ok := cfg.Providers[providerName]; ok {
				credPath := config.DefaultCredentialPath()
				saltPath := config.DefaultSaltPath()
				store, storeErr := credential.New(credPath, saltPath)
				if storeErr == nil {
					k, kErr := store.Get(providerName)
					if kErr == nil {
						apiKey = k
					}
				}
			}
		}
	}
	if apiKey == "" {
		return configError(fmt.Errorf("provider %q is not connected. Use 'potaco auth add %s' first", providerName, providerName))
	}

	baseURL := flagString(cmd, "base-url")
	opts := adapter.AdapterOpts{BaseURL: baseURL}
	ad, err := adapter.Get(providerName, apiKey, opts)
	if err != nil {
		return configError(fmt.Errorf("create adapter: %w", err))
	}

	jsonMode, _ := cmd.Root().PersistentFlags().GetBool("json")

	models, err := ad.DiscoverModels(context.Background())
	if err != nil {
		return apiError(fmt.Errorf("discover models: %w", err))
	}

	out := cmd.OutOrStdout()

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
