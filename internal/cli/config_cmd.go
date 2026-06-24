package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ngct/potaco/internal/config"
	"github.com/ngct/potaco/internal/provider"
	"github.com/spf13/cobra"
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

	// Load existing config or start fresh
	var content string
	if data, err := os.ReadFile(path); err == nil {
		content = string(data)
	}
	_ = content

	// Build new config YAML
	baseURL, _ := cmd.Flags().GetString("base-url")
	apiKey, _ := cmd.Flags().GetString("api-key")
	model, _ := cmd.Flags().GetString("model")
	retries, _ := cmd.Flags().GetInt("retries")
	timeoutStr, _ := cmd.Flags().GetString("timeout")
	providerName, _ := cmd.Flags().GetString("provider")

	// Apply preset if specified
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

	// Build YAML content
	lines := []string{"default:"}
	if baseURL != "" {
		lines = append(lines, fmt.Sprintf("  base_url: %q", baseURL))
	}
	if apiKey != "" {
		lines = append(lines, fmt.Sprintf("  api_key: %q", apiKey))
	}
	if model != "" {
		lines = append(lines, fmt.Sprintf("  model: %q", model))
	}
	if retries > 0 {
		lines = append(lines, fmt.Sprintf("  retries: %d", retries))
	}
	if timeoutStr != "" {
		lines = append(lines, fmt.Sprintf("  timeout: %q", timeoutStr))
	}

	newContent := strings.Join(lines, "\n") + "\n"

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
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
