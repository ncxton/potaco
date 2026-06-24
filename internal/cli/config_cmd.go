package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ncxton/potaco/internal/config"
	"github.com/ncxton/potaco/internal/provider"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage provider configuration",
}

var configSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set configuration values",
	RunE:  runConfigSet,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE:  runConfigShow,
}

var configListProvidersCmd = &cobra.Command{
	Use:   "list-providers",
	Short: "List available provider presets",
	RunE:  runConfigListProviders,
}

func init() {
	configSetCmd.Flags().String("base-url", "", "API base URL")
	configSetCmd.Flags().String("api-key", "", "API key")
	configSetCmd.Flags().String("model", "", "default model")
	configSetCmd.Flags().Int("retries", 0, "max retry attempts")
	configSetCmd.Flags().String("timeout", "", "request timeout (e.g., 120s)")
	configSetCmd.Flags().String("provider", "", "apply preset defaults (openai, together, fal)")

	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configListProvidersCmd)
	rootCmd.AddCommand(configCmd)
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	path := config.DefaultConfigPath()

	baseURL, _ := cmd.Flags().GetString("base-url")
	apiKey, _ := cmd.Flags().GetString("api-key")
	model, _ := cmd.Flags().GetString("model")
	retries, _ := cmd.Flags().GetInt("retries")
	timeoutStr, _ := cmd.Flags().GetString("timeout")
	providerName, _ := cmd.Flags().GetString("provider")

	if providerName != "" {
		preset, ok := provider.GetPreset(providerName)
		if !ok {
			return fmt.Errorf("unknown provider preset: %s", providerName)
		}
		if baseURL == "" {
			baseURL = preset.BaseURL
		}
		if model == "" {
			model = preset.DefaultModel
		}
	}

	fc := config.FileConfig{}
	fc.Default.BaseURL = baseURL
	fc.Default.APIKey = apiKey
	fc.Default.Model = model
	fc.Default.Retries = retries
	fc.Default.Timeout = timeoutStr

	data, err := yaml.Marshal(&fc)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Configuration saved to %s\n", path)
	return nil
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	path := config.DefaultConfigPath()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintln(cmd.OutOrStdout(), "No configuration file found at", path)
			fmt.Fprintln(cmd.OutOrStdout(), "Use 'potaco config set' to create one.")
			return nil
		}
		return fmt.Errorf("read config: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Config file: %s\n\n", path)
	fmt.Fprint(cmd.OutOrStdout(), string(data))
	return nil
}

func runConfigListProviders(cmd *cobra.Command, args []string) error {
	presets := provider.AllPresets()
	out := cmd.OutOrStdout()

	fmt.Fprintln(out, "Available provider presets:")
	fmt.Fprintln(out)
	for name, preset := range presets {
		fmt.Fprintf(out, "  %s:\n", name)
		fmt.Fprintf(out, "    base_url:      %s\n", preset.BaseURL)
		fmt.Fprintf(out, "    default_model: %s\n", preset.DefaultModel)
		fmt.Fprintf(out, "    sizes:         %v\n", preset.Sizes)
		fmt.Fprintln(out)
	}

	return nil
}
