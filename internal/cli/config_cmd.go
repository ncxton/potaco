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

	fc := config.FileConfig{}
	if existing, err := readConfigFile(path); err != nil {
		return configError(err)
	} else if existing != nil {
		fc = *existing
	}

	if cmd.Flags().Changed("provider") {
		preset, ok := provider.GetPreset(providerName)
		if !ok {
			return configError(fmt.Errorf("unknown provider preset: %s", providerName))
		}
		if !cmd.Flags().Changed("base-url") {
			fc.Default.BaseURL = preset.BaseURL
		}
		if !cmd.Flags().Changed("model") {
			fc.Default.Model = preset.DefaultModel
		}
	}
	if cmd.Flags().Changed("base-url") {
		fc.Default.BaseURL = baseURL
	}
	if cmd.Flags().Changed("api-key") {
		fc.Default.APIKey = apiKey
	}
	if cmd.Flags().Changed("model") {
		fc.Default.Model = model
	}
	if cmd.Flags().Changed("retries") {
		fc.Default.Retries = retries
	}
	if cmd.Flags().Changed("timeout") {
		fc.Default.Timeout = timeoutStr
	}

	data, err := yaml.Marshal(&fc)
	if err != nil {
		return configError(fmt.Errorf("marshal config: %w", err))
	}

	if err := writeConfigFile(path, data); err != nil {
		return configError(err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Configuration saved to %s\n", path)
	return nil
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	path := config.DefaultConfigPath()

	fc, err := readConfigFile(path)
	if err != nil {
		return configError(err)
	}
	if fc == nil {
		fmt.Fprintln(cmd.OutOrStdout(), "No configuration file found at", path)
		fmt.Fprintln(cmd.OutOrStdout(), "Use 'potaco config set' to create one.")
		return nil
	}
	if fc.Default.APIKey != "" {
		fc.Default.APIKey = "REDACTED"
	}
	data, err := yaml.Marshal(fc)
	if err != nil {
		return configError(fmt.Errorf("marshal config: %w", err))
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Config file: %s\n\n", path)
	fmt.Fprint(cmd.OutOrStdout(), string(data))
	return nil
}

func readConfigFile(path string) (*config.FileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}
	var fc config.FileConfig
	if err := yaml.Unmarshal(data, &fc); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &fc, nil
}

func writeConfigFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	if err := os.Chmod(dir, 0700); err != nil {
		return fmt.Errorf("set config directory permissions: %w", err)
	}
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing to write symlinked config file: %s", path)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat config: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".config-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp config: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("set temp config permissions: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp config: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace config: %w", err)
	}
	cleanup = false
	if err := os.Chmod(path, 0600); err != nil {
		return fmt.Errorf("set config permissions: %w", err)
	}
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
